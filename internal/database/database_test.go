package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestConnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Errorf("ping after connect: %v", err)
	}
}
