package database_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rupivbluegreen/pactline/internal/database"
)

// testdb returns a connected pool and runs migrations once. Each caller
// is expected to scope its data with unique uuids per test rather than
// rolling back transactions.
func testdb(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		if isPostgresUnavailable(err) {
			t.Skipf("postgres unavailable, skipping integration test: %v", err)
		}
		t.Fatalf("connect: %v", err)
	}
	if err := database.MigrateUp(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool.Pool
}

// isPostgresUnavailable detects driver-level connection failures so the
// test suite can be run on machines without a live Postgres.
func isPostgresUnavailable(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout")
}
