package database_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestContractDocuments_CreateAndGetLatest(t *testing.T) {
	pool := &database.Pool{Pool: testdb(t)}
	cdRepo := database.NewContractDocumentRepo(pool)
	cRepo := database.NewContractRepo(pool)
	ctx := context.Background()

	orgID, userID, ctID := seedOrgWithNDA(t, pool)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	c, err := cRepo.CreateTx(ctx, tx, &core.Contract{
		OrganizationID: orgID, ContractTypeID: ctID,
		Title: "Doc test", Status: core.ContractStatusIntake, OwnerUserID: userID,
	})
	if err != nil {
		t.Fatalf("create contract: %v", err)
	}
	d, err := cdRepo.CreateTx(ctx, tx, &core.ContractDocument{
		OrganizationID: orgID, ContractID: c.ID,
		StorageKey: orgID.String() + "/" + c.ID.String() + "/x.pdf",
		MimeType:   core.MimePDF,
		SHA256:     []byte("0123456789abcdef0123456789abcdef"),
		ByteSize:   1234,
	})
	if err != nil {
		t.Fatalf("create doc: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	got, err := cdRepo.GetLatestForContract(ctx, orgID, c.ID)
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if got.ID != d.ID {
		t.Errorf("expected %s, got %s", d.ID, got.ID)
	}

	if err := cdRepo.UpdateParsedText(ctx, orgID, d.ID, "hello world", 3); err != nil {
		t.Fatalf("update parsed: %v", err)
	}
	got2, _ := cdRepo.GetLatestForContract(ctx, orgID, c.ID)
	if got2.ParsedText == nil || *got2.ParsedText != "hello world" {
		t.Errorf("parsed_text not updated")
	}
	if got2.PageCount == nil || *got2.PageCount != 3 {
		t.Errorf("page_count not updated")
	}

	if _, err := cdRepo.GetLatestForContract(ctx, orgID, uuid.New()); err != core.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
