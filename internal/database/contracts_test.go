package database_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

// seedOrgWithNDA inserts a fresh org + user + nda contract_type and returns
// their IDs. Used by every test that needs a contract row.
func seedOrgWithNDA(t *testing.T, pool *database.Pool) (orgID, userID, ctID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	orgID = uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO organizations (id, slug, name) VALUES ($1, $2, $3)`,
		orgID, "c-"+orgID.String()[:8], "C Co"); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	userID = uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email) VALUES ($1, $2)`,
		userID, "c-"+userID.String()[:8]+"@test.example.com"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	ctID = uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO contract_types (id, organization_id, slug, name) VALUES ($1, $2, $3, $4)`,
		ctID, orgID, core.ContractTypeSlugNDA, "NDA"); err != nil {
		t.Fatalf("insert ct: %v", err)
	}
	return
}

func TestContractRepo_CreateGetList(t *testing.T) {
	pool := &database.Pool{Pool: testdb(t)}
	repo := database.NewContractRepo(pool)
	ctx := context.Background()

	orgID, userID, ctID := seedOrgWithNDA(t, pool)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	c, err := repo.CreateTx(ctx, tx, &core.Contract{
		OrganizationID: orgID, ContractTypeID: ctID,
		Title: "Test NDA", Status: core.ContractStatusIntake, OwnerUserID: userID,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if c.Status != core.ContractStatusIntake {
		t.Errorf("status: %s", c.Status)
	}

	got, err := repo.GetByID(ctx, orgID, c.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "Test NDA" {
		t.Errorf("title: %s", got.Title)
	}

	list, err := repo.ListByOrg(ctx, orgID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, item := range list {
		if item.ID == c.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("created contract not in list")
	}

	// Other org sees nothing.
	other := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO organizations (id, slug, name) VALUES ($1, $2, $3)`,
		other, "o-"+other.String()[:8], "O"); err != nil {
		t.Fatalf("insert other: %v", err)
	}
	otherList, err := repo.ListByOrg(ctx, other)
	if err != nil {
		t.Fatalf("list other: %v", err)
	}
	for _, item := range otherList {
		if item.ID == c.ID {
			t.Errorf("cross-tenant leak: contract %s visible to other org", c.ID)
		}
	}
}

func TestContractRepo_UpdateStatus(t *testing.T) {
	pool := &database.Pool{Pool: testdb(t)}
	repo := database.NewContractRepo(pool)
	ctx := context.Background()

	orgID, userID, ctID := seedOrgWithNDA(t, pool)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	c, err := repo.CreateTx(ctx, tx, &core.Contract{
		OrganizationID: orgID, ContractTypeID: ctID,
		Title: "Update test", Status: core.ContractStatusIntake, OwnerUserID: userID,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	prev, err := repo.UpdateStatus(ctx, orgID, c.ID, core.ContractStatusParsing)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if prev != core.ContractStatusIntake {
		t.Errorf("expected prev=intake, got %s", prev)
	}
	got, _ := repo.GetByID(ctx, orgID, c.ID)
	if got.Status != core.ContractStatusParsing {
		t.Errorf("expected status=parsing, got %s", got.Status)
	}
}
