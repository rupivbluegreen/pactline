package activities

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/core"
)

type PersistParsedInput struct {
	OrganizationID uuid.UUID
	DocumentID     uuid.UUID
	Text           string
	PageCount      int32
}

func (a *Activities) PersistParsedText(ctx context.Context, in PersistParsedInput) error {
	return a.ContractDocuments.UpdateParsedText(
		ctx, in.OrganizationID, in.DocumentID, in.Text, int(in.PageCount),
	)
}

type PersistExtractedInput struct {
	OrganizationID uuid.UUID
	Fields         []core.ExtractedField
}

func (a *Activities) PersistExtractedFields(ctx context.Context, in PersistExtractedInput) error {
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := a.ExtractedFields.UpsertBatchTx(ctx, tx, in.Fields); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
