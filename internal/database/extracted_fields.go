package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rupivbluegreen/pactline/internal/core"
)

type ExtractedFieldRepo struct{ pool *Pool }

func NewExtractedFieldRepo(p *Pool) *ExtractedFieldRepo { return &ExtractedFieldRepo{pool: p} }

// UpsertBatchTx inserts (or updates on conflict) all fields for one
// (contract_id, document_id) atomically. Idempotent on retry — Temporal
// activities re-run the same UpsertBatch and converge to the same row set.
func (r *ExtractedFieldRepo) UpsertBatchTx(ctx context.Context, tx pgx.Tx, fields []core.ExtractedField) error {
	if len(fields) == 0 {
		return nil
	}
	for i := range fields {
		f := &fields[i]
		_, err := tx.Exec(ctx, `
			INSERT INTO extracted_fields (
				organization_id, contract_id, document_id, field_name,
				field_value, field_value_json, page_or_paragraph,
				span_start, span_end, model_id, prompt_version
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (contract_id, document_id, field_name) DO UPDATE SET
				field_value = EXCLUDED.field_value,
				field_value_json = EXCLUDED.field_value_json,
				page_or_paragraph = EXCLUDED.page_or_paragraph,
				span_start = EXCLUDED.span_start,
				span_end = EXCLUDED.span_end,
				model_id = EXCLUDED.model_id,
				prompt_version = EXCLUDED.prompt_version,
				extracted_at = now()
		`,
			f.OrganizationID, f.ContractID, f.DocumentID, f.FieldName,
			f.FieldValue, f.FieldValueJSON, f.PageOrParagraph,
			f.SpanStart, f.SpanEnd, f.ModelID, f.PromptVersion)
		if err != nil {
			return fmt.Errorf("upsert extracted_field %s: %w", f.FieldName, err)
		}
	}
	return nil
}

func (r *ExtractedFieldRepo) ListForContract(ctx context.Context, orgID, contractID uuid.UUID) ([]core.ExtractedField, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, organization_id, contract_id, document_id, field_name,
		       field_value, field_value_json, page_or_paragraph, span_start, span_end,
		       model_id, prompt_version, extracted_at
		FROM extracted_fields
		WHERE organization_id = $1 AND contract_id = $2
		ORDER BY field_name
	`, orgID, contractID)
	if err != nil {
		return nil, fmt.Errorf("list extracted_fields: %w", err)
	}
	defer rows.Close()

	var out []core.ExtractedField
	for rows.Next() {
		var f core.ExtractedField
		if err := rows.Scan(&f.ID, &f.OrganizationID, &f.ContractID, &f.DocumentID, &f.FieldName,
			&f.FieldValue, &f.FieldValueJSON, &f.PageOrParagraph, &f.SpanStart, &f.SpanEnd,
			&f.ModelID, &f.PromptVersion, &f.ExtractedAt); err != nil {
			return nil, fmt.Errorf("scan extracted_field: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
