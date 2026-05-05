package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func MigrateUp(ctx context.Context, pool *Pool) error {
	db := stdlib.OpenDBFromPool(pool.Pool)
	defer db.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS)
	if err != nil {
		// ErrNoMigrations is returned when the embedded directory contains no
		// .sql files; normal during Phase 0 before Task 3 adds the first
		// migration.
		if errors.Is(err, goose.ErrNoMigrations) {
			return nil
		}
		return fmt.Errorf("goose provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
