package database_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestExtractedFields_UpsertAndList(t *testing.T) {
	pool := &database.Pool{Pool: testdb(t)}
	cRepo := database.NewContractRepo(pool)
	cdRepo := database.NewContractDocumentRepo(pool)
	efRepo := database.NewExtractedFieldRepo(pool)
	ctx := context.Background()

	orgID, userID, ctID := seedOrgWithNDA(t, pool)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	c, err := cRepo.CreateTx(ctx, tx, &core.Contract{
		OrganizationID: orgID, ContractTypeID: ctID,
		Title: "EF test", Status: core.ContractStatusParsing, OwnerUserID: userID,
	})
	if err != nil {
		t.Fatalf("create c: %v", err)
	}
	d, err := cdRepo.CreateTx(ctx, tx, &core.ContractDocument{
		OrganizationID: orgID, ContractID: c.ID,
		StorageKey: "x", MimeType: core.MimePDF, SHA256: []byte("a"), ByteSize: 1,
	})
	if err != nil {
		t.Fatalf("create d: %v", err)
	}
	fields := []core.ExtractedField{
		{OrganizationID: orgID, ContractID: c.ID, DocumentID: d.ID,
			FieldName: "parties", FieldValue: "Acme & Beta",
			PageOrParagraph: "page:1", SpanStart: 0, SpanEnd: 11,
			ModelID: "anthropic/claude-3-5-sonnet", PromptVersion: "v1"},
		{OrganizationID: orgID, ContractID: c.ID, DocumentID: d.ID,
			FieldName: "effective_date", FieldValue: "2026-05-01",
			PageOrParagraph: "page:1", SpanStart: 30, SpanEnd: 40,
			ModelID: "anthropic/claude-3-5-sonnet", PromptVersion: "v1"},
	}
	if err := efRepo.UpsertBatchTx(ctx, tx, fields); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	list, err := efRepo.ListForContract(ctx, orgID, c.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(list))
	}
	if list[0].FieldName != "effective_date" {
		t.Errorf("ordering: first field name %s", list[0].FieldName)
	}

	// Idempotent re-upsert with updated value.
	tx2, _ := pool.Begin(ctx)
	fields[0].FieldValue = "Acme & Beta (revised)"
	if err := efRepo.UpsertBatchTx(ctx, tx2, fields); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	if err := tx2.Commit(ctx); err != nil {
		t.Fatalf("commit2: %v", err)
	}
	list2, _ := efRepo.ListForContract(ctx, orgID, c.ID)
	if len(list2) != 2 {
		t.Errorf("expected still 2 rows after re-upsert, got %d", len(list2))
	}
	for _, f := range list2 {
		if f.FieldName == "parties" && f.FieldValue != "Acme & Beta (revised)" {
			t.Errorf("expected revised value, got %q", f.FieldValue)
		}
	}

	// Cross-tenant: other org sees no rows for this contract.
	other := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO organizations (id, slug, name) VALUES ($1, $2, $3)`,
		other, "of-"+other.String()[:8], "OF"); err != nil {
		t.Fatalf("insert other: %v", err)
	}
	otherList, _ := efRepo.ListForContract(ctx, other, c.ID)
	if len(otherList) != 0 {
		t.Errorf("cross-tenant leak: %d rows visible to other org", len(otherList))
	}
}
