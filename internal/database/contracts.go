package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rupivbluegreen/pactline/internal/core"
)

type ContractRepo struct{ pool *Pool }

func NewContractRepo(p *Pool) *ContractRepo { return &ContractRepo{pool: p} }

// CreateTx inserts a contract inside an existing transaction.
func (r *ContractRepo) CreateTx(ctx context.Context, tx pgx.Tx, c *core.Contract) (*core.Contract, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO contracts (organization_id, contract_type_id, title, status, owner_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, organization_id, contract_type_id, title, status, owner_user_id, created_at, updated_at
	`, c.OrganizationID, c.ContractTypeID, c.Title, c.Status, c.OwnerUserID)

	var out core.Contract
	if err := row.Scan(&out.ID, &out.OrganizationID, &out.ContractTypeID, &out.Title, &out.Status, &out.OwnerUserID, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, fmt.Errorf("insert contract: %w", err)
	}
	return &out, nil
}

func (r *ContractRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*core.Contract, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, organization_id, contract_type_id, title, status, owner_user_id, created_at, updated_at
		FROM contracts WHERE organization_id = $1 AND id = $2
	`, orgID, id)
	var c core.Contract
	if err := row.Scan(&c.ID, &c.OrganizationID, &c.ContractTypeID, &c.Title, &c.Status, &c.OwnerUserID, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("get contract: %w", err)
	}
	return &c, nil
}

func (r *ContractRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]core.Contract, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, organization_id, contract_type_id, title, status, owner_user_id, created_at, updated_at
		FROM contracts WHERE organization_id = $1 ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list contracts: %w", err)
	}
	defer rows.Close()

	var out []core.Contract
	for rows.Next() {
		var c core.Contract
		if err := rows.Scan(&c.ID, &c.OrganizationID, &c.ContractTypeID, &c.Title, &c.Status, &c.OwnerUserID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan contract: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateStatus sets the contract's status and bumps updated_at. Returns the
// previous status (read via a CTE before the UPDATE) for audit before/after.
func (r *ContractRepo) UpdateStatus(ctx context.Context, orgID, id uuid.UUID, next core.ContractStatus) (core.ContractStatus, error) {
	var prev core.ContractStatus
	row := r.pool.QueryRow(ctx, `
		WITH old AS (
			SELECT status FROM contracts WHERE organization_id = $1 AND id = $2
		)
		UPDATE contracts SET status = $3, updated_at = now()
		WHERE organization_id = $1 AND id = $2
		RETURNING (SELECT status FROM old)
	`, orgID, id, next)
	if err := row.Scan(&prev); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", core.ErrNotFound
		}
		return "", fmt.Errorf("update contract status: %w", err)
	}
	return prev, nil
}
