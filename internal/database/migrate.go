package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func MigrateUp(ctx context.Context, pool *Pool) error {
	cfg := pool.Pool.Config().ConnConfig
	db := stdlib.OpenDB(*cfg)
	defer db.Close()

	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		if errors.Is(err, goose.ErrNoMigrationFiles) {
			return nil
		}
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
