// Package database wraps pgx for tenant-scoped Postgres access.
//
// Phase 0: empty. Phase 1: pgxpool wiring + TenantScoped(ctx) helper
// that injects organization_id from request context. Goose migrations
// live in /migrations.
package database
