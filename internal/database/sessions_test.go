package database_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestSessionRepo_CreateAndGet(t *testing.T) {
	pool := testdb(t)
	uRepo := database.NewUserRepo(&database.Pool{Pool: pool})
	sRepo := database.NewSessionRepo(&database.Pool{Pool: pool})
	ctx := context.Background()

	u, _, err := uRepo.CreateOrGetByEmail(ctx, uniqueEmail("s-"+t.Name()))
	if err != nil {
		t.Fatalf("user: %v", err)
	}

	tok, sess, err := sRepo.Create(ctx, u.ID, time.Hour)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if tok == "" {
		t.Error("empty token")
	}

	got, err := sRepo.GetByToken(ctx, tok)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("id mismatch")
	}

	if err := sRepo.Revoke(ctx, sess.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := sRepo.GetByToken(ctx, tok); !errors.Is(err, core.ErrSessionRevoked) {
		t.Errorf("expected ErrSessionRevoked, got %v", err)
	}
}

func TestSessionRepo_Expired(t *testing.T) {
	pool := testdb(t)
	uRepo := database.NewUserRepo(&database.Pool{Pool: pool})
	sRepo := database.NewSessionRepo(&database.Pool{Pool: pool})
	ctx := context.Background()
	u, _, err := uRepo.CreateOrGetByEmail(ctx, uniqueEmail("se-"+t.Name()))
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	tok, _, err := sRepo.Create(ctx, u.ID, -time.Second)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := sRepo.GetByToken(ctx, tok); !errors.Is(err, core.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestMagicLinkRepo_ConsumeOnce(t *testing.T) {
	pool := testdb(t)
	mRepo := database.NewMagicLinkRepo(&database.Pool{Pool: pool})
	ctx := context.Background()
	email := uniqueEmail("ml-" + t.Name())

	tok, err := mRepo.Create(ctx, email, 15*time.Minute)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := mRepo.Consume(ctx, tok)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if got != email {
		t.Errorf("email: got %s want %s", got, email)
	}

	if _, err := mRepo.Consume(ctx, tok); !errors.Is(err, core.ErrTokenConsumed) {
		t.Errorf("expected ErrTokenConsumed on second consume, got %v", err)
	}
}

func TestMagicLinkRepo_Expired(t *testing.T) {
	pool := testdb(t)
	mRepo := database.NewMagicLinkRepo(&database.Pool{Pool: pool})
	ctx := context.Background()

	tok, err := mRepo.Create(ctx, uniqueEmail("exp-"+t.Name()), -time.Second)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := mRepo.Consume(ctx, tok); !errors.Is(err, core.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestMagicLinkRepo_InvalidToken(t *testing.T) {
	pool := testdb(t)
	mRepo := database.NewMagicLinkRepo(&database.Pool{Pool: pool})
	ctx := context.Background()
	if _, err := mRepo.Consume(ctx, "definitely-not-a-token"); !errors.Is(err, core.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}
