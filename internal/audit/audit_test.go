package audit_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rupivbluegreen/pactline/internal/audit"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func mustPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		if isPostgresUnavailable(err) {
			t.Skipf("postgres unavailable: %v", err)
		}
		t.Fatalf("connect: %v", err)
	}
	if err := database.MigrateUp(ctx, p); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { p.Close() })
	return p.Pool
}

func isPostgresUnavailable(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout")
}

func insertOrg(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, slug string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, slug, name) VALUES ($1, $2, $3)`, id, slug, slug)
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
}

func insertUser(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, email string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email) VALUES ($1, $2)`, id, email)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
}

func TestWrite_AppendsToChain(t *testing.T) {
	pool := mustPool(t)
	ctx := context.Background()
	_ = time.Now // suppress import lint

	org := uuid.New()
	user := uuid.New()
	insertOrg(t, pool, org, "test-"+org.String()[:8])
	insertUser(t, pool, user, "test-"+user.String()[:8]+"@example.com")

	if err := audit.Write(ctx, pool, audit.Event{
		OrganizationID: &org,
		ActorUserID:    &user,
		Action:         audit.ActionUserCreated,
		EntityType:     "user",
		EntityID:       &user,
		After:          map[string]string{"email": "test@example.com"},
	}); err != nil {
		t.Fatalf("write 1: %v", err)
	}

	if err := audit.Write(ctx, pool, audit.Event{
		OrganizationID: &org,
		ActorUserID:    &user,
		Action:         audit.ActionOrganizationCreated,
		EntityType:     "organization",
		EntityID:       &org,
		After:          map[string]string{"name": "Test Co"},
	}); err != nil {
		t.Fatalf("write 2: %v", err)
	}

	row := pool.QueryRow(ctx, `
		SELECT
		  (SELECT event_hash FROM audit_events WHERE organization_id=$1 AND action='user.created' LIMIT 1),
		  (SELECT prev_hash  FROM audit_events WHERE organization_id=$1 AND action='organization.created' LIMIT 1)
	`, org)
	var firstHash, secondPrev []byte
	if err := row.Scan(&firstHash, &secondPrev); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if string(firstHash) != string(secondPrev) {
		t.Errorf("chain broken: first=%x second_prev=%x", firstHash, secondPrev)
	}
}

func TestWrite_ImmutableTable(t *testing.T) {
	pool := mustPool(t)
	ctx := context.Background()

	org := uuid.New()
	insertOrg(t, pool, org, "imm-"+org.String()[:8])

	if err := audit.Write(ctx, pool, audit.Event{
		OrganizationID: &org,
		Action:         audit.ActionUserCreated,
		EntityType:     "user",
		EntityID:       &org,
	}); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM audit_events WHERE organization_id=$1`, org); err == nil {
		t.Errorf("expected error on DELETE, got none")
	}
	if _, err := pool.Exec(ctx, `UPDATE audit_events SET action='x' WHERE organization_id=$1`, org); err == nil {
		t.Errorf("expected error on UPDATE, got none")
	}
}
