package database

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rupivbluegreen/pactline/internal/core"
)

type SessionRepo struct{ pool *Pool }

func NewSessionRepo(p *Pool) *SessionRepo { return &SessionRepo{pool: p} }

// SessionToken is the plaintext value handed to the client. Server stores
// only sha256(token) in sessions.token_hash.
type SessionToken string

func newRandomToken() (SessionToken, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return SessionToken(base64.RawURLEncoding.EncodeToString(buf)), nil
}

func HashToken(t SessionToken) []byte {
	sum := sha256.Sum256([]byte(t))
	return sum[:]
}

// Create issues a new session and returns the plaintext token (the
// server stores only its hash).
func (r *SessionRepo) Create(ctx context.Context, userID uuid.UUID, ttl time.Duration) (SessionToken, *core.Session, error) {
	tok, err := newRandomToken()
	if err != nil {
		return "", nil, fmt.Errorf("rand: %w", err)
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, expires_at, created_at, revoked_at
	`, userID, HashToken(tok), time.Now().Add(ttl))
	var s core.Session
	if err := row.Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt, &s.RevokedAt); err != nil {
		return "", nil, fmt.Errorf("insert session: %w", err)
	}
	return tok, &s, nil
}

// GetByToken returns the session for a presented plaintext token, or
// errors with ErrNotFound / ErrSessionRevoked / ErrTokenExpired.
func (r *SessionRepo) GetByToken(ctx context.Context, tok SessionToken) (*core.Session, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, expires_at, created_at, revoked_at
		FROM sessions WHERE token_hash = $1
	`, HashToken(tok))
	var s core.Session
	if err := row.Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt, &s.RevokedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	if s.RevokedAt != nil {
		return nil, core.ErrSessionRevoked
	}
	if time.Now().After(s.ExpiresAt) {
		return nil, core.ErrTokenExpired
	}
	return &s, nil
}

func (r *SessionRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}
