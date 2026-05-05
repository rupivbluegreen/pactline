package middleware_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/api/middleware"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func mustPool(t *testing.T) *database.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		if isInfraUnavailable(err) {
			t.Skipf("db unavailable: %v", err)
		}
		t.Fatalf("connect: %v", err)
	}
	if err := database.MigrateUp(ctx, p); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { p.Close() })
	return p
}

func isInfraUnavailable(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout")
}

func TestAuth_NoToken(t *testing.T) {
	p := mustPool(t)
	mw := middleware.Auth(database.NewSessionRepo(p), database.NewMembershipRepo(p))
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuth_ValidBearer(t *testing.T) {
	p := mustPool(t)
	users := database.NewUserRepo(p)
	sess := database.NewSessionRepo(p)
	memb := database.NewMembershipRepo(p)
	ctx := context.Background()

	u, _, _ := users.CreateOrGetByEmail(ctx, "mw-"+t.Name()+"@example.com")
	tok, _, err := sess.Create(ctx, u.ID, time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	called := false
	mw := middleware.Auth(sess, memb)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		uid, ok := database.UserID(r.Context())
		if !ok || uid != u.ID {
			t.Errorf("ctx user mismatch: got=%v ok=%v", uid, ok)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+string(tok))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !called {
		t.Error("handler not called")
	}
}

func TestAuth_RevokedSessionReturns401(t *testing.T) {
	p := mustPool(t)
	users := database.NewUserRepo(p)
	sess := database.NewSessionRepo(p)
	memb := database.NewMembershipRepo(p)
	ctx := context.Background()

	u, _, _ := users.CreateOrGetByEmail(ctx, "mw2-"+t.Name()+"@example.com")
	tok, s, err := sess.Create(ctx, u.ID, time.Hour)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := sess.Revoke(ctx, s.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	mw := middleware.Auth(sess, memb)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("handler should not run")
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+string(tok))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
