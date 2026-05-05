package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/auth"
	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/email"
)

func newSvc(t *testing.T) *auth.Service {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		if isInfraUnavailable(err) {
			t.Skipf("infra unavailable: %v", err)
		}
		t.Fatalf("connect: %v", err)
	}
	if err := database.MigrateUp(ctx, p); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { p.Close() })

	if isMailpitUnavailable() {
		t.Skip("mailpit unavailable on localhost:1025")
	}
	return auth.NewService(p,
		database.NewUserRepo(p), database.NewMagicLinkRepo(p),
		database.NewSessionRepo(p), email.NewSender(), "http://localhost:8000",
	)
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

func isMailpitUnavailable() bool {
	conn, err := net.DialTimeout("tcp", "localhost:1025", 500*time.Millisecond)
	if err != nil {
		return true
	}
	_ = conn.Close()
	return false
}

func extractTokenFromMailpit(t *testing.T, to string) database.MagicLinkToken {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:8025/api/v1/search?query=to%3A" + to)
		if err == nil {
			var body struct {
				Messages []struct {
					ID string `json:"ID"`
				} `json:"messages"`
			}
			err := json.NewDecoder(resp.Body).Decode(&body)
			resp.Body.Close()
			if err == nil && len(body.Messages) > 0 {
				resp2, err := http.Get("http://localhost:8025/api/v1/message/" + body.Messages[0].ID)
				if err == nil {
					var msg struct {
						Text string `json:"Text"`
					}
					err := json.NewDecoder(resp2.Body).Decode(&msg)
					resp2.Body.Close()
					if err == nil {
						i := strings.Index(msg.Text, "token=")
						if i >= 0 {
							rest := msg.Text[i+len("token="):]
							end := strings.IndexAny(rest, " \r\n\t")
							if end < 0 {
								end = len(rest)
							}
							return database.MagicLinkToken(rest[:end])
						}
					}
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("token not found")
	return ""
}

func TestRequestAndVerifyMagicLink(t *testing.T) {
	svc := newSvc(t)
	ctx := context.Background()
	addr := strings.ToLower("auth-" + t.Name() + "@example.com")

	if err := svc.RequestMagicLink(ctx, addr); err != nil {
		t.Fatalf("request: %v", err)
	}
	tok := extractTokenFromMailpit(t, addr)

	res, err := svc.VerifyMagicLink(ctx, tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if res.SessionToken == "" || res.User.Email != addr {
		t.Errorf("bad result: %+v", res)
	}

	if _, err := svc.VerifyMagicLink(ctx, tok); !errors.Is(err, core.ErrTokenConsumed) {
		t.Errorf("expected ErrTokenConsumed, got %v", err)
	}
}

func TestLogout(t *testing.T) {
	svc := newSvc(t)
	ctx := context.Background()
	addr := strings.ToLower("logout-" + t.Name() + "@example.com")

	if err := svc.RequestMagicLink(ctx, addr); err != nil {
		t.Fatalf("request: %v", err)
	}
	tok := extractTokenFromMailpit(t, addr)
	res, err := svc.VerifyMagicLink(ctx, tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}

	if err := svc.Logout(ctx, res.SessionToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
}
