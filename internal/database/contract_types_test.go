package database_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestContractTypes_CreateAndGet(t *testing.T) {
	pool := testdb(t)
	ctx := context.Background()
	repo := database.NewContractTypeRepo(&database.Pool{Pool: pool})

	orgID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO organizations (id, slug, name) VALUES ($1, $2, $3)`,
		orgID, "ct-"+orgID.String()[:8], "CT Co"); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ct, err := repo.CreateTx(ctx, tx, orgID, core.ContractTypeSlugNDA, "NDA")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	got, err := repo.GetBySlug(ctx, orgID, core.ContractTypeSlugNDA)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != ct.ID {
		t.Errorf("id mismatch: %s vs %s", got.ID, ct.ID)
	}

	if _, err := repo.GetBySlug(ctx, orgID, "missing"); err != core.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
