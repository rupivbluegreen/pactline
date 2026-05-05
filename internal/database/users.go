package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rupivbluegreen/pactline/internal/core"
)

type UserRepo struct{ pool *Pool }

func NewUserRepo(p *Pool) *UserRepo { return &UserRepo{pool: p} }

// CreateOrGetByEmail returns the user for this email, creating one if
// none exists. Returns (user, created, err) — created=true when this
// call inserted a new row.
func (r *UserRepo) CreateOrGetByEmail(ctx context.Context, email string) (*core.User, bool, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (email) VALUES ($1)
		ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		RETURNING id, email, created_at, last_login_at, (xmax = 0)
	`, email)
	var u core.User
	var created bool
	if err := row.Scan(&u.ID, &u.Email, &u.CreatedAt, &u.LastLoginAt, &created); err != nil {
		return nil, false, fmt.Errorf("upsert user: %w", err)
	}
	return &u, created, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*core.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, created_at, last_login_at FROM users WHERE id = $1
	`, id)
	var u core.User
	if err := row.Scan(&u.ID, &u.Email, &u.CreatedAt, &u.LastLoginAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}
