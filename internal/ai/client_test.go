package ai_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/ai"
)

func TestHealth_LiveSidecar(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cli, err := ai.Dial(ctx, "localhost:50051")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer cli.Close()

	status, err := cli.Health(ctx)
	if err != nil {
		if isUnavailable(err) {
			t.Skipf("ai sidecar unavailable: %v", err)
		}
		t.Fatalf("health: %v", err)
	}
	if status != "ok" {
		t.Errorf("status: %q", status)
	}
}

func isUnavailable(err error) bool {
	var ne *net.OpError
	if errors.As(err, &ne) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "Unavailable") ||
		strings.Contains(msg, "i/o timeout")
}
