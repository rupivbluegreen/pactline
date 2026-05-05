package storage_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/storage"
)

func newClient(t *testing.T) *storage.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cfg := storage.ConfigFromEnv()
	cfg.Bucket = "pactline-test-" + uuid.NewString()[:8]
	c, err := storage.New(ctx, cfg)
	if err != nil {
		if isMinioDown(err) {
			t.Skipf("minio unavailable: %v", err)
		}
		t.Fatalf("client: %v", err)
	}
	return c
}

func isMinioDown(err error) bool {
	var ne *net.OpError
	if errors.As(err, &ne) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "dial tcp")
}

func TestPutGet(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()
	key := "test/" + uuid.NewString() + ".txt"
	body := []byte("hello pactline")

	if err := c.Put(ctx, key, "text/plain", body); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, err := c.Get(ctx, key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("body mismatch: %q vs %q", got, body)
	}
}

func TestSignedURL(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()
	key := "test/" + uuid.NewString() + ".txt"
	body := []byte("signed url payload")
	if err := c.Put(ctx, key, "text/plain", body); err != nil {
		t.Fatalf("put: %v", err)
	}

	u, err := c.SignedURL(ctx, key, 60*time.Second)
	if err != nil {
		t.Fatalf("signed url: %v", err)
	}
	if u.Host == "" {
		t.Fatalf("empty host")
	}

	resp, err := http.Get(u.String())
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: %d", resp.StatusCode)
	}
}
