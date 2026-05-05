package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestSlugFromName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Acme Corp", "acme-corp"},
		{"  Hello World!  ", "hello-world"},
		{"!@#$", "org"},
		{"Mid Size Co.", "mid-size-co"},
	}
	for _, c := range cases {
		if got := database.SlugFromName(c.in); got != c.want {
			t.Errorf("SlugFromName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestOrgRepo_CreateAndConflict(t *testing.T) {
	pool := testdb(t)
	repo := database.NewOrganizationRepo(&database.Pool{Pool: pool})
	ctx := context.Background()

	o1, err := repo.Create(ctx, "Acme "+t.Name()+" "+uuid.NewString()[:8], "")
	if err != nil {
		t.Fatalf("create 1: %v", err)
	}
	if o1.Slug == "" {
		t.Error("expected non-empty slug")
	}

	if _, err := repo.Create(ctx, "Other Name", o1.Slug); !errors.Is(err, core.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists on slug conflict, got %v", err)
	}
}

func TestMembershipRepo_Create_ListForUser(t *testing.T) {
	pool := testdb(t)
	uRepo := database.NewUserRepo(&database.Pool{Pool: pool})
	oRepo := database.NewOrganizationRepo(&database.Pool{Pool: pool})
	mRepo := database.NewMembershipRepo(&database.Pool{Pool: pool})
	ctx := context.Background()

	u, _, err := uRepo.CreateOrGetByEmail(ctx, "m-"+uuid.NewString()[:8]+"@test.example.com")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	o, err := oRepo.Create(ctx, "Acme "+t.Name()+" "+uuid.NewString()[:8], "")
	if err != nil {
		t.Fatalf("org: %v", err)
	}

	if _, err := mRepo.Create(ctx, u.ID, o.ID, core.RoleOwner); err != nil {
		t.Fatalf("create membership: %v", err)
	}

	ms, err := mRepo.ListForUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, m := range ms {
		if m.OrganizationID == o.ID && m.Role == core.RoleOwner {
			found = true
		}
	}
	if !found {
		t.Errorf("expected owner membership, got %+v", ms)
	}
}
