package database

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/rupivbluegreen/pactline/internal/core"
)

type MagicLinkRepo struct{ pool *Pool }

func NewMagicLinkRepo(p *Pool) *MagicLinkRepo { return &MagicLinkRepo{pool: p} }

type MagicLinkToken string

func newMagicLinkToken() (MagicLinkToken, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return MagicLinkToken(base64.RawURLEncoding.EncodeToString(buf)), nil
}

func hashLinkToken(t MagicLinkToken) []byte {
	return HashToken(SessionToken(t)) // sha256 either way
}

// Create returns the plaintext magic-link token.
func (r *MagicLinkRepo) Create(ctx context.Context, email string, ttl time.Duration) (MagicLinkToken, error) {
	tok, err := newMagicLinkToken()
	if err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO magic_links (email, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, email, hashLinkToken(tok), time.Now().Add(ttl))
	if err != nil {
		return "", fmt.Errorf("insert magic link: %w", err)
	}
	return tok, nil
}

// Consume marks the link used and returns the email it was issued for.
// Errors with ErrInvalidToken / ErrTokenExpired / ErrTokenConsumed.
func (r *MagicLinkRepo) Consume(ctx context.Context, tok MagicLinkToken) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		SELECT email, expires_at, used_at FROM magic_links
		WHERE token_hash = $1 FOR UPDATE
	`, hashLinkToken(tok))
	var email string
	var expiresAt time.Time
	var usedAt *time.Time
	if err := row.Scan(&email, &expiresAt, &usedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", core.ErrInvalidToken
		}
		return "", fmt.Errorf("query link: %w", err)
	}
	if usedAt != nil {
		return "", core.ErrTokenConsumed
	}
	if time.Now().After(expiresAt) {
		return "", core.ErrTokenExpired
	}
	if _, err := tx.Exec(ctx,
		`UPDATE magic_links SET used_at = now() WHERE token_hash = $1`,
		hashLinkToken(tok)); err != nil {
		return "", fmt.Errorf("mark used: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}
	return email, nil
}
