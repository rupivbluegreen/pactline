package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rupivbluegreen/pactline/internal/core"
)

type ContractTypeRepo struct{ pool *Pool }

func NewContractTypeRepo(p *Pool) *ContractTypeRepo { return &ContractTypeRepo{pool: p} }

// CreateTx inserts a contract type inside an existing transaction.
func (r *ContractTypeRepo) CreateTx(ctx context.Context, tx pgx.Tx, orgID uuid.UUID, slug, name string) (*core.ContractType, error) {
	var ct core.ContractType
	err := tx.QueryRow(ctx, `
		INSERT INTO contract_types (organization_id, slug, name)
		VALUES ($1, $2, $3)
		RETURNING id, organization_id, slug, name, created_at
	`, orgID, slug, name).Scan(&ct.ID, &ct.OrganizationID, &ct.Slug, &ct.Name, &ct.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert contract_type: %w", err)
	}
	return &ct, nil
}

// GetBySlug returns the contract type for a given (org, slug).
func (r *ContractTypeRepo) GetBySlug(ctx context.Context, orgID uuid.UUID, slug string) (*core.ContractType, error) {
	var ct core.ContractType
	err := r.pool.QueryRow(ctx, `
		SELECT id, organization_id, slug, name, created_at
		FROM contract_types WHERE organization_id = $1 AND slug = $2
	`, orgID, slug).Scan(&ct.ID, &ct.OrganizationID, &ct.Slug, &ct.Name, &ct.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("get contract_type: %w", err)
	}
	return &ct, nil
}
