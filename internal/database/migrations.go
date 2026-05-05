package database

import "embed"

// `all:migrations` (rather than `migrations/*.sql`) so the embed compiles
// before Task 3 adds the first .sql file. Go's embed errors on zero glob
// matches; `all:` matches the directory itself plus any files including
// .gitkeep. Once migrations exist this directive could be tightened to
// `migrations/*.sql`, but `all:` continues to work either way.
//
//go:embed all:migrations
var migrationsFS embed.FS
