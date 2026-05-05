package database_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestConnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		if isPostgresUnavailable(err) {
			t.Skipf("postgres unavailable, skipping integration test: %v", err)
		}
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Errorf("ping after connect: %v", err)
	}
}

func isPostgresUnavailable(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "connect: connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout")
}

func TestMigrateUp_AppliesSchema(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		if isPostgresUnavailable(err) {
			t.Skipf("postgres unavailable, skipping integration test: %v", err)
		}
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := database.MigrateUp(ctx, pool); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	// Sanity: all 6 tables created by the migration are present.
	var count int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name IN
		      ('users','organizations','memberships','sessions','magic_links','audit_events')
	`).Scan(&count); err != nil {
		t.Fatalf("schema check: %v", err)
	}
	if count != 6 {
		t.Errorf("expected 6 tables, got %d", count)
	}
}
