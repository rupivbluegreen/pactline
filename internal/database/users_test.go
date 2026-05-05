package database_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/database"
)

func uniqueEmail(prefix string) string {
	return prefix + "-" + uuid.NewString()[:8] + "@test.example.com"
}

func TestUserRepo_CreateOrGet(t *testing.T) {
	pool := testdb(t)
	repo := database.NewUserRepo(&database.Pool{Pool: pool})
	ctx := context.Background()
	email := uniqueEmail("u-" + t.Name())

	u1, created1, err := repo.CreateOrGetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if !created1 {
		t.Error("expected created=true on first call")
	}

	u2, created2, err := repo.CreateOrGetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if created2 {
		t.Error("expected created=false on second call")
	}
	if u1.ID != u2.ID {
		t.Errorf("ids differ: %s vs %s", u1.ID, u2.ID)
	}
}

func TestUserRepo_UpdateLastLogin(t *testing.T) {
	pool := testdb(t)
	repo := database.NewUserRepo(&database.Pool{Pool: pool})
	ctx := context.Background()
	u, _, err := repo.CreateOrGetByEmail(ctx, uniqueEmail("ul-"+t.Name()))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.LastLoginAt != nil {
		t.Error("expected LastLoginAt nil on creation")
	}
	if err := repo.UpdateLastLogin(ctx, u.ID); err != nil {
		t.Fatalf("update last login: %v", err)
	}
	got, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.LastLoginAt == nil {
		t.Error("expected LastLoginAt set after UpdateLastLogin")
	}
}
