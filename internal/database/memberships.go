package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/core"
)

type MembershipRepo struct{ pool *Pool }

func NewMembershipRepo(p *Pool) *MembershipRepo { return &MembershipRepo{pool: p} }

func (r *MembershipRepo) Create(ctx context.Context, userID, orgID uuid.UUID, role core.Role) (*core.Membership, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO memberships (user_id, organization_id, role) VALUES ($1, $2, $3)
		RETURNING id, user_id, organization_id, role, created_at
	`, userID, orgID, string(role))
	var m core.Membership
	var roleStr string
	if err := row.Scan(&m.ID, &m.UserID, &m.OrganizationID, &roleStr, &m.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert membership: %w", err)
	}
	m.Role = core.Role(roleStr)
	return &m, nil
}

func (r *MembershipRepo) ListForUser(ctx context.Context, userID uuid.UUID) ([]core.Membership, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, organization_id, role, created_at
		FROM memberships WHERE user_id = $1 ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()

	var out []core.Membership
	for rows.Next() {
		var m core.Membership
		var roleStr string
		if err := rows.Scan(&m.ID, &m.UserID, &m.OrganizationID, &roleStr, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Role = core.Role(roleStr)
		out = append(out, m)
	}
	return out, rows.Err()
}
