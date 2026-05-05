package organizations_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/organizations"
)

func mustPool(t *testing.T) *database.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		if isInfraUnavailable(err) {
			t.Skipf("db unavailable: %v", err)
		}
		t.Fatalf("connect: %v", err)
	}
	if err := database.MigrateUp(ctx, p); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { p.Close() })
	return p
}

func isInfraUnavailable(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout")
}

func TestCreateForUser(t *testing.T) {
	p := mustPool(t)
	ctx := context.Background()
	users := database.NewUserRepo(p)
	mems := database.NewMembershipRepo(p)
	svc := organizations.NewService(p)

	u, _, err := users.CreateOrGetByEmail(ctx, "org-"+t.Name()+"@example.com")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	o, err := svc.CreateForUser(ctx, u.ID, "Acme "+t.Name()+" "+uuid.NewString()[:8], "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if o.Slug == "" {
		t.Error("empty slug")
	}

	got, _ := mems.ListForUser(ctx, u.ID)
	found := false
	for _, m := range got {
		if m.OrganizationID == o.ID && m.Role == core.RoleOwner {
			found = true
		}
	}
	if !found {
		t.Errorf("owner membership not found: %+v", got)
	}
}

func TestCreateForUser_SlugConflict(t *testing.T) {
	p := mustPool(t)
	ctx := context.Background()
	users := database.NewUserRepo(p)
	svc := organizations.NewService(p)

	u, _, err := users.CreateOrGetByEmail(ctx, "orgc-"+t.Name()+"@example.com")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	first, err := svc.CreateForUser(ctx, u.ID, "Acme "+t.Name()+" "+uuid.NewString()[:8], "")
	if err != nil {
		t.Fatalf("create 1: %v", err)
	}
	if _, err := svc.CreateForUser(ctx, u.ID, "Different name", first.Slug); !errors.Is(err, core.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}
}
