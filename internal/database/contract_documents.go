package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rupivbluegreen/pactline/internal/core"
)

type ContractDocumentRepo struct{ pool *Pool }

func NewContractDocumentRepo(p *Pool) *ContractDocumentRepo {
	return &ContractDocumentRepo{pool: p}
}

func (r *ContractDocumentRepo) CreateTx(ctx context.Context, tx pgx.Tx, d *core.ContractDocument) (*core.ContractDocument, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO contract_documents (organization_id, contract_id, storage_key, mime_type, sha256, byte_size)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, organization_id, contract_id, storage_key, mime_type, sha256, byte_size, parsed_text, page_count, created_at
	`, d.OrganizationID, d.ContractID, d.StorageKey, d.MimeType, d.SHA256, d.ByteSize)

	var out core.ContractDocument
	if err := row.Scan(&out.ID, &out.OrganizationID, &out.ContractID, &out.StorageKey, &out.MimeType, &out.SHA256, &out.ByteSize, &out.ParsedText, &out.PageCount, &out.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert contract_document: %w", err)
	}
	return &out, nil
}

func (r *ContractDocumentRepo) GetLatestForContract(ctx context.Context, orgID, contractID uuid.UUID) (*core.ContractDocument, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, organization_id, contract_id, storage_key, mime_type, sha256, byte_size, parsed_text, page_count, created_at
		FROM contract_documents
		WHERE organization_id = $1 AND contract_id = $2
		ORDER BY created_at DESC LIMIT 1
	`, orgID, contractID)
	var d core.ContractDocument
	if err := row.Scan(&d.ID, &d.OrganizationID, &d.ContractID, &d.StorageKey, &d.MimeType, &d.SHA256, &d.ByteSize, &d.ParsedText, &d.PageCount, &d.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("get latest document: %w", err)
	}
	return &d, nil
}

// UpdateParsedText sets parsed_text and page_count on the document. Idempotent
// — repeated calls with the same values converge.
func (r *ContractDocumentRepo) UpdateParsedText(ctx context.Context, orgID, id uuid.UUID, text string, pageCount int) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE contract_documents SET parsed_text = $3, page_count = $4
		WHERE organization_id = $1 AND id = $2
	`, orgID, id, text, pageCount)
	if err != nil {
		return fmt.Errorf("update parsed text: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}
	return nil
}
