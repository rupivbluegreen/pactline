package database

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rupivbluegreen/pactline/internal/core"
)

type OrganizationRepo struct{ pool *Pool }

func NewOrganizationRepo(p *Pool) *OrganizationRepo { return &OrganizationRepo{pool: p} }

func (r *OrganizationRepo) Create(ctx context.Context, name, slug string) (*core.Organization, error) {
	if slug == "" {
		slug = SlugFromName(name)
	}
	row := r.pool.QueryRow(ctx, `
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
	return &o, nil
}

func (r *OrganizationRepo) GetByID(ctx context.Context, id uuid.UUID) (*core.Organization, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, slug, name, created_at FROM organizations WHERE id = $1`, id)
	var o core.Organization
	if err := row.Scan(&o.ID, &o.Slug, &o.Name, &o.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("get org: %w", err)
	}
	return &o, nil
}

var slugInvalid = regexp.MustCompile(`[^a-z0-9-]+`)

func SlugFromName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	s = slugInvalid.ReplaceAllString(s, "")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "org"
	}
	return s
}
