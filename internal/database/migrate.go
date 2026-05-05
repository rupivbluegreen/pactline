package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func MigrateUp(ctx context.Context, pool *Pool) error {
	db := stdlib.OpenDBFromPool(pool.Pool)
	defer db.Close()

	// migrationsFS is embedded with "all:migrations", so the .sql files live at
	// migrations/<file>.sql inside the FS. fs.Sub gives goose a sub-FS rooted
	// at that directory so it finds the files at the top level.
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migrations sub-fs: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, sub)
	if err != nil {
		// ErrNoMigrations is returned when the embedded directory contains no
		// .sql files; tolerated here so the binary starts cleanly before any
		// migration is added.
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
