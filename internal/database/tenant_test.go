package database_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestTenantScoped_Unset(t *testing.T) {
	if _, ok := database.TenantScoped(context.Background()); ok {
		t.Error("expected ok=false on bare context")
	}
}

func TestTenantScoped_Set(t *testing.T) {
	want := uuid.New()
	ctx := database.WithOrgID(context.Background(), want)
	got, ok := database.TenantScoped(ctx)
	if !ok || got != want {
		t.Errorf("got (%s, %v), want (%s, true)", got, ok, want)
	}
}

func TestUserID_Roundtrip(t *testing.T) {
	want := uuid.New()
	ctx := database.WithUserID(context.Background(), want)
	got, ok := database.UserID(ctx)
	if !ok || got != want {
		t.Errorf("got (%s, %v), want (%s, true)", got, ok, want)
	}
}
