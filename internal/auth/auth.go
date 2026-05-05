// Package auth issues magic links and exchanges them for sessions.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rupivbluegreen/pactline/internal/audit"
	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/email"
)

const (
	MagicLinkTTL = 15 * time.Minute
	SessionTTL   = 30 * 24 * time.Hour
)

type Service struct {
	users   *database.UserRepo
	links   *database.MagicLinkRepo
	sess    *database.SessionRepo
	mails   *email.Sender
	pool    *database.Pool
	baseURL string
}

func NewService(
	pool *database.Pool, users *database.UserRepo, links *database.MagicLinkRepo,
	sess *database.SessionRepo, mails *email.Sender, baseURL string,
) *Service {
	return &Service{users: users, links: links, sess: sess, mails: mails, pool: pool, baseURL: baseURL}
}

// RequestMagicLink creates the user (if new), creates a magic_link, emails the URL.
func (s *Service) RequestMagicLink(ctx context.Context, emailAddr string) error {
	emailAddr = strings.TrimSpace(strings.ToLower(emailAddr))
	if !strings.Contains(emailAddr, "@") {
		return fmt.Errorf("invalid email")
	}

	u, created, err := s.users.CreateOrGetByEmail(ctx, emailAddr)
	if err != nil {
		return err
	}
	if created {
		if err := audit.Write(ctx, s.pool, audit.Event{
			ActorUserID: &u.ID,
			Action:      audit.ActionUserCreated,
			EntityType:  "user",
			EntityID:    &u.ID,
			After:       map[string]string{"email": u.Email},
		}); err != nil {
			return err
		}
	}

	tok, err := s.links.Create(ctx, emailAddr, MagicLinkTTL)
	if err != nil {
		return err
	}
	if err := audit.Write(ctx, s.pool, audit.Event{
		ActorUserID: &u.ID,
		Action:      audit.ActionMagicLinkRequested,
		EntityType:  "magic_link",
		After:       map[string]string{"email": emailAddr},
	}); err != nil {
		return err
	}

	link := s.baseURL + "/auth/verify?token=" + string(tok)
	body := "Sign in to Pactline:\n\n" + link + "\n\nThis link expires in 15 minutes."
	return s.mails.Send(ctx, emailAddr, "Pactline sign-in link", body)
}

// VerifyResult is everything the client needs after a successful verify.
type VerifyResult struct {
	SessionToken database.SessionToken
	User         *core.User
	ExpiresAt    time.Time
}

func (s *Service) VerifyMagicLink(ctx context.Context, tok database.MagicLinkToken) (*VerifyResult, error) {
	emailAddr, err := s.links.Consume(ctx, tok)
	if err != nil {
		return nil, err
	}
	u, _, err := s.users.CreateOrGetByEmail(ctx, emailAddr)
	if err != nil {
		return nil, err
	}
	if err := audit.Write(ctx, s.pool, audit.Event{
		ActorUserID: &u.ID,
		Action:      audit.ActionMagicLinkConsumed,
		EntityType:  "magic_link",
		After:       map[string]string{"email": emailAddr},
	}); err != nil {
		return nil, err
	}
	if err := s.users.UpdateLastLogin(ctx, u.ID); err != nil {
		return nil, err
	}

	sessTok, sess, err := s.sess.Create(ctx, u.ID, SessionTTL)
	if err != nil {
		return nil, err
	}
	if err := audit.Write(ctx, s.pool, audit.Event{
		ActorUserID: &u.ID,
		Action:      audit.ActionSessionCreated,
		EntityType:  "session",
		EntityID:    &sess.ID,
	}); err != nil {
		return nil, err
	}

	return &VerifyResult{SessionToken: sessTok, User: u, ExpiresAt: sess.ExpiresAt}, nil
}

func (s *Service) Logout(ctx context.Context, tok database.SessionToken) error {
	sess, err := s.sess.GetByToken(ctx, tok)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) || errors.Is(err, core.ErrSessionRevoked) || errors.Is(err, core.ErrTokenExpired) {
			return nil
		}
		return err
	}
	if err := s.sess.Revoke(ctx, sess.ID); err != nil {
		return err
	}
	if err := audit.Write(ctx, s.pool, audit.Event{
		ActorUserID: &sess.UserID,
		Action:      audit.ActionSessionRevoked,
		EntityType:  "session",
		EntityID:    &sess.ID,
	}); err != nil {
		return err
	}
	return nil
}
