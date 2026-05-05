// Package organizations creates orgs with an owner membership atomically.
package organizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rupivbluegreen/pactline/internal/audit"
	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

type Service struct {
	pool         *database.Pool
	contractType *database.ContractTypeRepo
}

func NewService(pool *database.Pool) *Service {
	return &Service{pool: pool, contractType: database.NewContractTypeRepo(pool)}
}

// CreateForUser creates the org, adds the user as owner, seeds the NDA
// contract type, and writes audit events. All rows are committed in one tx.
func (s *Service) CreateForUser(ctx context.Context, userID uuid.UUID, name, slug string) (*core.Organization, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if slug == "" {
		slug = database.SlugFromName(name)
	}
	row := tx.QueryRow(ctx, `
		INSERT INTO organizations (slug, name) VALUES ($1, $2)
		RETURNING id, slug, name, created_at
	`, slug, name)
	var o core.Organization
	if err := row.Scan(&o.ID, &o.Slug, &o.Name, &o.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, core.ErrAlreadyExists
		}
		return nil, fmt.Errorf("insert org: %w", err)
	}

	mrow := tx.QueryRow(ctx, `
		INSERT INTO memberships (user_id, organization_id, role) VALUES ($1, $2, 'owner')
		RETURNING id
	`, userID, o.ID)
	var memID uuid.UUID
	if err := mrow.Scan(&memID); err != nil {
		return nil, fmt.Errorf("insert membership: %w", err)
	}

	ct, err := s.contractType.CreateTx(ctx, tx, o.ID, core.ContractTypeSlugNDA, "NDA")
	if err != nil {
		return nil, fmt.Errorf("seed contract type: %w", err)
	}

	if err := audit.Write(ctx, tx, audit.Event{
		OrganizationID: &o.ID, ActorUserID: &userID,
		Action: audit.ActionOrganizationCreated, EntityType: "organization",
		EntityID: &o.ID, After: map[string]string{"slug": o.Slug, "name": o.Name},
	}); err != nil {
		return nil, err
	}
	if err := audit.Write(ctx, tx, audit.Event{
		OrganizationID: &o.ID, ActorUserID: &userID,
		Action: audit.ActionMembershipCreated, EntityType: "membership",
		EntityID: &memID, After: map[string]string{"role": "owner"},
	}); err != nil {
		return nil, err
	}
	if err := audit.Write(ctx, tx, audit.Event{
		OrganizationID: &o.ID, ActorUserID: &userID,
		Action: audit.ActionContractTypeCreated, EntityType: "contract_type",
		EntityID: &ct.ID, After: map[string]string{"slug": ct.Slug, "name": ct.Name},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &o, nil
}
