# Phase 1 Story 1 — Signup → Organization → Empty Contracts List Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** New user signs in via magic link, creates an organization, lands on an empty contracts list. Exercises the full multi-tenant chassis (auth, sessions, hash-chained audit, tenant-scoped queries) end-to-end.

**Architecture:** Polyglot per ADR 0003 — Go core (`cmd/api`, `internal/*`), Python sidecars idle this story, Next.js 15 frontend. Postgres 16 with pgvector via pgx. Goose-driven migrations embedded in the Go binary. Hash-chained audit log enforced by a Postgres trigger. Magic-link issuer is bespoke; bearer/cookie session middleware mirrors statebound's shape. No AI / Temporal / DocuSign in this story.

**Tech Stack:** Go 1.22+ (chi/v5, pgx/v5, jackc/tern or pressly/goose, golang-migrate), Postgres 16+ with `pgcrypto`, Mailpit for dev SMTP, Next.js 15 / React 19 server components, openapi-fetch, openapi-typescript.

---

## File map (built across the tasks)

```
migrations/
  00001_init_schema.sql                   tables + audit trigger function
internal/
  audit/
    audit.go                               Event, Action constants, Write
    audit_test.go                          chain integrity tests
  database/
    database.go                            pgxpool wrapper, Connect/Close
    tenant.go                              TenantScoped(ctx) helper
    migrate.go                             goose runner with embed.FS
    migrations.go                          //go:embed migrations/*.sql
    users.go                               UserRepo
    organizations.go                       OrganizationRepo
    memberships.go                         MembershipRepo
    sessions.go                            SessionRepo
    magic_links.go                         MagicLinkRepo
    *_test.go                              repo integration tests
    testdb_test.go                         per-test transaction-rolled-back fixture
  core/
    user.go                                User type
    organization.go                        Organization type
    membership.go                          Membership, Role enum
    session.go                             Session type
    magic_link.go                          MagicLink type
    errors.go                              (already exists; extended with domain errors)
  auth/
    auth.go                                RequestMagicLink, VerifyMagicLink, Logout
    auth_test.go
  email/
    email.go                               SMTP via net/smtp
    email_test.go
  organizations/
    organizations.go                       Create (slug, owner membership)
    organizations_test.go
  api/
    router.go                              (modified — register new routes)
    middleware/
      auth.go                              bearer/cookie session middleware
      auth_test.go
    handlers/
      auth_handlers.go                     /auth/magic-link/* + /logout
      me.go                                GET /me
      organizations.go                     POST /organizations
      contracts.go                         GET /contracts (empty)
      *_test.go                            httptest coverage
cmd/api/main.go                            wire pool, migrations, services
schemas/openapi.yaml                       6 new operations
apps/web/src/
  api/types.ts                             regenerated
  app/
    page.tsx                               redirect to /signin or /contracts
    layout.tsx                             header with user/org/logout
    signin/page.tsx
    signin/sent/page.tsx
    auth/verify/page.tsx
    onboarding/organization/page.tsx
    contracts/page.tsx
  lib/server-session.ts                    SSR session-cookie reader
```

---

## Task 1: pgxpool wiring + database connection

**Files:**
- Modify: `go.mod` (add pgx)
- Modify: `internal/database/database.go` (replace stub)
- Test: `internal/database/database_test.go`

- [ ] **Step 1: Add pgx dependency**

```bash
GOTOOLCHAIN=local go get github.com/jackc/pgx/v5@v5.7.1 github.com/jackc/pgx/v5/pgxpool@v5.7.1
GOTOOLCHAIN=local go mod tidy
```

- [ ] **Step 2: Replace internal/database/database.go**

```go
// Package database wraps pgx for tenant-scoped Postgres access.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	*pgxpool.Pool
}

func Connect(ctx context.Context, dsn string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Pool{Pool: p}, nil
}

func DSNFromEnv() string {
	// host/port/user/password/db all overridable via env
	host := envOr("PGHOST", "localhost")
	port := envOr("PGPORT", "5433")
	user := envOr("PGUSER", "pactline")
	pass := envOr("PGPASSWORD", "pactline")
	db := envOr("PGDATABASE", "pactline")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, db)
}

func envOr(k, fallback string) string {
	if v := getenv(k); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 3: Add the os import shim**

Append to `internal/database/database.go`:

```go
import "os"

func getenv(k string) string { return os.Getenv(k) }
```

(Or merge into the imports block; the helper is a one-liner so we can just inline `os.Getenv`. The two-step lets the test stub the env without monkeypatching. Inline `os.Getenv` is fine — simplify.)

Replace `func envOr` with:

```go
func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
```

And remove the `getenv` indirection. Add `"os"` to the existing imports block.

- [ ] **Step 4: Write the failing connection test**

Create `internal/database/database_test.go`:

```go
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
```

- [ ] **Step 5: Run test**

```bash
GOTOOLCHAIN=local go test ./internal/database/...
```

Expected: PASS (Postgres is up via `make up`).

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/database/database.go internal/database/database_test.go
git commit -m "feat(database): pgxpool connection wiring with env-driven DSN"
```

---

## Task 2: Goose migrations runner

**Files:**
- Modify: `go.mod` (add goose)
- Create: `internal/database/migrations.go`
- Create: `internal/database/migrate.go`
- Modify: `internal/database/database_test.go` (add migrate test)

- [ ] **Step 1: Add goose**

```bash
GOTOOLCHAIN=local go get github.com/pressly/goose/v3@v3.22.1
GOTOOLCHAIN=local go mod tidy
```

- [ ] **Step 2: Create internal/database/migrations.go**

```go
package database

import "embed"

//go:embed migrations/*.sql
var migrationsFS embed.FS
```

- [ ] **Step 3: Create internal/database/migrations/ symlink-style include**

Goose's `embed.FS` requires the migrations to live alongside the package. Move `migrations/` → keep root copy AND mirror inside the package:

Actually simpler: change `//go:embed` to point at the repo-root `migrations/`. Go's embed cannot reach above the package dir. So we keep migrations in `internal/database/migrations/`. Move the existing root `migrations/.gitkeep` and update infrastructure references.

Run:

```bash
mkdir -p internal/database/migrations
git mv migrations/.gitkeep internal/database/migrations/.gitkeep
rmdir migrations 2>/dev/null || true
```

- [ ] **Step 4: Create internal/database/migrate.go**

```go
package database

import (
	"context"
	"database/sql"
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
		return fmt.Errorf("goose up: %w", err)
	}
	_ = sql.ErrNoRows
	return nil
}
```

- [ ] **Step 5: Write migration smoke test**

Append to `internal/database/database_test.go`:

```go
func TestMigrateUp_NoMigrations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	// Empty migrations dir is a no-op; this just exercises the wiring.
	if err := database.MigrateUp(ctx, pool); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
}
```

- [ ] **Step 6: Run test**

```bash
GOTOOLCHAIN=local go test ./internal/database/...
```

Expected: PASS (no migrations yet, no-op).

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum internal/database/migrate.go internal/database/migrations.go internal/database/migrations/.gitkeep
git commit -m "feat(database): goose migration runner with embedded FS"
```

---

## Task 3: First migration — schema + audit chain trigger

**Files:**
- Create: `internal/database/migrations/00001_init_schema.sql`

- [ ] **Step 1: Write the migration**

Create `internal/database/migrations/00001_init_schema.sql`:

```sql
-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         citext NOT NULL UNIQUE,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_login_at timestamptz
);

CREATE TABLE organizations (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       citext NOT NULL UNIQUE,
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE memberships (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role            text NOT NULL CHECK (role IN ('owner','admin','reviewer','approver','viewer')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, organization_id)
);
CREATE INDEX memberships_user_idx ON memberships(user_id);
CREATE INDEX memberships_org_idx  ON memberships(organization_id);

CREATE TABLE sessions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  bytea NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    revoked_at  timestamptz
);
CREATE INDEX sessions_user_idx ON sessions(user_id);

CREATE TABLE magic_links (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email       citext NOT NULL,
    token_hash  bytea NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    used_at     timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX magic_links_email_idx ON magic_links(email);

CREATE TABLE audit_events (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid REFERENCES organizations(id) ON DELETE RESTRICT,
    actor_user_id   uuid REFERENCES users(id) ON DELETE RESTRICT,
    action          text NOT NULL,
    entity_type     text NOT NULL,
    entity_id       uuid,
    before          jsonb,
    after           jsonb,
    prev_hash       bytea,
    event_hash      bytea NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_events_org_created_idx ON audit_events(organization_id, created_at);
CREATE INDEX audit_events_chain_idx ON audit_events(organization_id, created_at DESC);

-- canonical_event_payload returns the bytes that get hashed for a given row.
-- Stable across re-imports: same logical event -> same bytes -> same hash.
CREATE OR REPLACE FUNCTION canonical_event_payload(
    p_org uuid, p_actor uuid, p_action text, p_entity_type text,
    p_entity_id uuid, p_before jsonb, p_after jsonb, p_created_at timestamptz
) RETURNS bytea LANGUAGE sql IMMUTABLE AS $$
    SELECT convert_to(
        json_build_object(
            'organization_id', p_org,
            'actor_user_id', p_actor,
            'action', p_action,
            'entity_type', p_entity_type,
            'entity_id', p_entity_id,
            'before', p_before,
            'after', p_after,
            'created_at', to_char(p_created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        )::text,
        'UTF8'
    );
$$;

-- BEFORE INSERT trigger: looks up prev event in same chain (per organization_id),
-- sets prev_hash and event_hash on the new row.
CREATE OR REPLACE FUNCTION audit_event_set_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_prev bytea;
BEGIN
    SELECT event_hash INTO v_prev
    FROM audit_events
    WHERE organization_id IS NOT DISTINCT FROM NEW.organization_id
    ORDER BY created_at DESC, id DESC
    LIMIT 1;

    NEW.prev_hash := v_prev;
    NEW.event_hash := digest(
        coalesce(v_prev, ''::bytea) ||
        canonical_event_payload(
            NEW.organization_id, NEW.actor_user_id, NEW.action,
            NEW.entity_type, NEW.entity_id, NEW.before, NEW.after, NEW.created_at
        ),
        'sha256'
    );
    RETURN NEW;
END;
$$;

CREATE TRIGGER audit_events_hash_trigger
BEFORE INSERT ON audit_events
FOR EACH ROW EXECUTE FUNCTION audit_event_set_hash();

-- Block UPDATE / DELETE on audit_events at the table level.
CREATE OR REPLACE FUNCTION audit_events_immutable() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_events is append-only';
END;
$$;
CREATE TRIGGER audit_events_no_update BEFORE UPDATE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_immutable();
CREATE TRIGGER audit_events_no_delete BEFORE DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_immutable();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS audit_events_no_delete ON audit_events;
DROP TRIGGER IF EXISTS audit_events_no_update ON audit_events;
DROP TRIGGER IF EXISTS audit_events_hash_trigger ON audit_events;
DROP FUNCTION IF EXISTS audit_events_immutable();
DROP FUNCTION IF EXISTS audit_event_set_hash();
DROP FUNCTION IF EXISTS canonical_event_payload(uuid, uuid, text, text, uuid, jsonb, jsonb, timestamptz);
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS magic_links;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
```

- [ ] **Step 2: Apply migration**

```bash
docker compose -f infra/docker-compose.yml exec -T postgres psql -U pactline -d pactline -c '\dt'
# baseline; expect empty
```

Then run via Go (after Task 4's tests below trigger MigrateUp), or apply manually for now:

```bash
PGPASSWORD=pactline psql -h localhost -p 5433 -U pactline -d pactline -f internal/database/migrations/00001_init_schema.sql
```

- [ ] **Step 3: Verify**

```bash
PGPASSWORD=pactline psql -h localhost -p 5433 -U pactline -d pactline -c '\dt'
```

Expected: 6 tables — `users`, `organizations`, `memberships`, `sessions`, `magic_links`, `audit_events`.

```bash
PGPASSWORD=pactline psql -h localhost -p 5433 -U pactline -d pactline \
  -c "INSERT INTO audit_events (action, entity_type) VALUES ('test.event','test'); SELECT encode(event_hash,'hex') FROM audit_events;"
```

Expected: a sha256 hex string.

```bash
PGPASSWORD=pactline psql -h localhost -p 5433 -U pactline -d pactline \
  -c "DELETE FROM audit_events;"
```

Expected: error `audit_events is append-only`.

- [ ] **Step 4: Roll back, then let goose apply it**

```bash
PGPASSWORD=pactline psql -h localhost -p 5433 -U pactline -d pactline -c "
  -- manual cleanup since the down migration is what goose runs
  DROP TRIGGER IF EXISTS audit_events_no_delete ON audit_events;
  DROP TRIGGER IF EXISTS audit_events_no_update ON audit_events;
  DROP TRIGGER IF EXISTS audit_events_hash_trigger ON audit_events;
  DROP TABLE IF EXISTS audit_events, magic_links, sessions, memberships, organizations, users CASCADE;
  DROP FUNCTION IF EXISTS audit_events_immutable(), audit_event_set_hash();
  DROP FUNCTION IF EXISTS canonical_event_payload(uuid, uuid, text, text, uuid, jsonb, jsonb, timestamptz);
"
```

Then run Task 1's test (which calls MigrateUp):

```bash
GOTOOLCHAIN=local go test ./internal/database/... -run TestMigrateUp_NoMigrations
```

Wait — that test was for empty migrations. Update the test name and assertion in the next task.

- [ ] **Step 5: Commit migration**

```bash
git add internal/database/migrations/00001_init_schema.sql
git commit -m "feat(database): initial schema with hash-chained audit trigger"
```

---

## Task 4: Audit writer + chain integrity tests

**Files:**
- Replace: `internal/audit/audit.go`
- Create: `internal/audit/audit_test.go`
- Create: `internal/database/testdb_test.go`

- [ ] **Step 1: Replace internal/audit/audit.go**

```go
// Package audit is the hash-chained immutable audit log.
//
// Every state change writes one Event via Write. The Postgres trigger
// audit_events_hash_trigger sets prev_hash and event_hash on insert.
package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Action is the canonical name of an audited operation.
type Action string

const (
	ActionUserCreated         Action = "user.created"
	ActionMagicLinkRequested  Action = "magic_link.requested"
	ActionMagicLinkConsumed   Action = "magic_link.consumed"
	ActionSessionCreated      Action = "session.created"
	ActionSessionRevoked      Action = "session.revoked"
	ActionOrganizationCreated Action = "organization.created"
	ActionMembershipCreated   Action = "membership.created"
)

// Event is one row in audit_events. prev_hash and event_hash are computed
// by the database trigger; do not set them in app code.
type Event struct {
	OrganizationID *uuid.UUID
	ActorUserID    *uuid.UUID
	Action         Action
	EntityType     string
	EntityID       *uuid.UUID
	Before         any
	After          any
}

// Querier is the subset of pgx that audit needs. Both *pgxpool.Pool and
// pgx.Tx satisfy it.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgx.CommandTag, error)
}

func Write(ctx context.Context, q Querier, e Event) error {
	beforeJSON, err := jsonValue(e.Before)
	if err != nil {
		return fmt.Errorf("encode before: %w", err)
	}
	afterJSON, err := jsonValue(e.After)
	if err != nil {
		return fmt.Errorf("encode after: %w", err)
	}

	_, err = q.Exec(ctx, `
		INSERT INTO audit_events (
			organization_id, actor_user_id, action, entity_type, entity_id, before, after
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, e.OrganizationID, e.ActorUserID, string(e.Action), e.EntityType, e.EntityID, beforeJSON, afterJSON)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func jsonValue(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}
```

- [ ] **Step 2: Add uuid + pgx imports to go.mod**

```bash
GOTOOLCHAIN=local go get github.com/google/uuid@v1.6.0
GOTOOLCHAIN=local go mod tidy
```

- [ ] **Step 3: Create internal/database/testdb_test.go (shared test fixture)**

```go
package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rupivbluegreen/pactline/internal/database"
)

// testdb returns a connected pool and runs migrations once per process.
// Each test caller is expected to wrap its work in a transaction it rolls
// back, OR to clean up its own rows.
func testdb(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		t.Skipf("postgres unavailable, skipping integration test: %v", err)
	}
	if err := database.MigrateUp(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool.Pool
}
```

- [ ] **Step 4: Create internal/audit/audit_test.go**

```go
package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rupivbluegreen/pactline/internal/audit"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func mustPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := database.MigrateUp(ctx, p); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { p.Close() })
	return p.Pool
}

func TestWrite_AppendsToChain(t *testing.T) {
	pool := mustPool(t)
	ctx := context.Background()

	// Clean any prior audit rows for this test's chain (NULL org chain).
	// We can't DELETE; mark this test isolated by using a synthetic org.
	org := uuid.New()
	user := uuid.New()
	insertOrg(t, pool, org, "test-"+org.String()[:8])
	insertUser(t, pool, user, "test-"+user.String()[:8]+"@example.com")

	if err := audit.Write(ctx, pool, audit.Event{
		OrganizationID: &org,
		ActorUserID:    &user,
		Action:         audit.ActionUserCreated,
		EntityType:     "user",
		EntityID:       &user,
		After:          map[string]string{"email": "test@example.com"},
	}); err != nil {
		t.Fatalf("write 1: %v", err)
	}

	if err := audit.Write(ctx, pool, audit.Event{
		OrganizationID: &org,
		ActorUserID:    &user,
		Action:         audit.ActionOrganizationCreated,
		EntityType:     "organization",
		EntityID:       &org,
		After:          map[string]string{"name": "Test Co"},
	}); err != nil {
		t.Fatalf("write 2: %v", err)
	}

	// Verify chain: second event's prev_hash equals first event's event_hash.
	row := pool.QueryRow(ctx, `
		SELECT
		  (SELECT event_hash FROM audit_events WHERE organization_id=$1 AND action='user.created' LIMIT 1),
		  (SELECT prev_hash FROM audit_events WHERE organization_id=$1 AND action='organization.created' LIMIT 1)
	`, org)
	var firstHash, secondPrev []byte
	if err := row.Scan(&firstHash, &secondPrev); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if string(firstHash) != string(secondPrev) {
		t.Errorf("chain broken: first=%x second_prev=%x", firstHash, secondPrev)
	}
	time.Sleep(0) // suppress unused-import warning if any
}

func TestWrite_ImmutableTable(t *testing.T) {
	pool := mustPool(t)
	ctx := context.Background()

	org := uuid.New()
	insertOrg(t, pool, org, "imm-"+org.String()[:8])

	if err := audit.Write(ctx, pool, audit.Event{
		OrganizationID: &org,
		Action:         audit.ActionUserCreated,
		EntityType:     "user",
		EntityID:       &org,
	}); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM audit_events WHERE organization_id=$1`, org); err == nil {
		t.Errorf("expected error on DELETE, got none")
	}
	if _, err := pool.Exec(ctx, `UPDATE audit_events SET action='x' WHERE organization_id=$1`, org); err == nil {
		t.Errorf("expected error on UPDATE, got none")
	}
}

func insertOrg(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, slug string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO organizations (id, slug, name) VALUES ($1, $2, $3)`, id, slug, slug)
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
}

func insertUser(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, email string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email) VALUES ($1, $2)`, id, email)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
}
```

- [ ] **Step 5: Run tests**

```bash
GOTOOLCHAIN=local go test ./internal/audit/... -v
```

Expected: PASS for both `TestWrite_AppendsToChain` and `TestWrite_ImmutableTable`.

- [ ] **Step 6: Commit**

```bash
git add internal/audit/audit.go internal/audit/audit_test.go internal/database/testdb_test.go go.mod go.sum
git commit -m "feat(audit): hash-chained Write with chain integrity + immutability tests"
```

---

## Task 5: Core domain types

**Files:**
- Modify: `internal/core/errors.go` (add domain errors)
- Create: `internal/core/user.go`
- Create: `internal/core/organization.go`
- Create: `internal/core/membership.go`
- Create: `internal/core/session.go`
- Create: `internal/core/magic_link.go`

- [ ] **Step 1: Replace internal/core/errors.go**

```go
// Package core holds pactline's domain types and the base error type.
package core

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrInvalidToken    = errors.New("invalid token")
	ErrTokenExpired    = errors.New("token expired")
	ErrTokenConsumed   = errors.New("token already consumed")
	ErrSessionRevoked  = errors.New("session revoked")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrNoOrganization  = errors.New("no organization context")
)
```

- [ ] **Step 2: Create internal/core/user.go**

```go
package core

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID
	Email       string
	CreatedAt   time.Time
	LastLoginAt *time.Time
}
```

- [ ] **Step 3: Create internal/core/organization.go**

```go
package core

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID        uuid.UUID
	Slug      string
	Name      string
	CreatedAt time.Time
}
```

- [ ] **Step 4: Create internal/core/membership.go**

```go
package core

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner    Role = "owner"
	RoleAdmin    Role = "admin"
	RoleReviewer Role = "reviewer"
	RoleApprover Role = "approver"
	RoleViewer   Role = "viewer"
)

type Membership struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Role           Role
	CreatedAt      time.Time
}
```

- [ ] **Step 5: Create internal/core/session.go**

```go
package core

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}
```

- [ ] **Step 6: Create internal/core/magic_link.go**

```go
package core

import (
	"time"

	"github.com/google/uuid"
)

type MagicLink struct {
	ID        uuid.UUID
	Email     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
```

- [ ] **Step 7: Verify build**

```bash
GOTOOLCHAIN=local go build ./...
```

Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add internal/core/
git commit -m "feat(core): User, Organization, Membership, Session, MagicLink types + domain errors"
```

---

## Task 6: TenantScoped context helper

**Files:**
- Create: `internal/database/tenant.go`
- Create: `internal/database/tenant_test.go`

- [ ] **Step 1: Create internal/database/tenant.go**

```go
package database

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey int

const (
	ctxKeyUser ctxKey = iota
	ctxKeyOrg
)

func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKeyUser, id)
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxKeyUser).(uuid.UUID)
	return v, ok
}

func WithOrgID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKeyOrg, id)
}

// TenantScoped returns the organization_id for the current request, or
// (uuid.Nil, false) when none is set. Repositories that touch tenant
// tables MUST call this and inject the filter into every query.
func TenantScoped(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxKeyOrg).(uuid.UUID)
	return v, ok
}
```

- [ ] **Step 2: Create internal/database/tenant_test.go**

```go
package database_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestTenantScoped_Unset(t *testing.T) {
	if _, ok := database.TenantScoped(context.Background()); ok {
		t.Error("expected ok=false on bare context")
	}
}

func TestTenantScoped_Set(t *testing.T) {
	want := uuid.New()
	ctx := database.WithOrgID(context.Background(), want)
	got, ok := database.TenantScoped(ctx)
	if !ok || got != want {
		t.Errorf("got (%s, %v), want (%s, true)", got, ok, want)
	}
}

func TestUserID_Roundtrip(t *testing.T) {
	want := uuid.New()
	ctx := database.WithUserID(context.Background(), want)
	got, ok := database.UserID(ctx)
	if !ok || got != want {
		t.Errorf("got (%s, %v), want (%s, true)", got, ok, want)
	}
}
```

- [ ] **Step 3: Run tests**

```bash
GOTOOLCHAIN=local go test ./internal/database/... -run 'TestTenantScoped|TestUserID' -v
```

Expected: 3 PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/database/tenant.go internal/database/tenant_test.go
git commit -m "feat(database): TenantScoped context helper"
```

---

## Task 7: Users repository

**Files:**
- Create: `internal/database/users.go`
- Create: `internal/database/users_test.go`

- [ ] **Step 1: Create internal/database/users.go**

```go
package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rupivbluegreen/pactline/internal/core"
)

type UserRepo struct{ pool *Pool }

func NewUserRepo(p *Pool) *UserRepo { return &UserRepo{pool: p} }

// CreateOrGetByEmail returns the user for this email, creating one if
// none exists. Returns (user, created, err) — created=true when this
// call inserted a new row.
func (r *UserRepo) CreateOrGetByEmail(ctx context.Context, email string) (*core.User, bool, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (email) VALUES ($1)
		ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		RETURNING id, email, created_at, last_login_at, (xmax = 0)
	`, email)
	var u core.User
	var created bool
	if err := row.Scan(&u.ID, &u.Email, &u.CreatedAt, &u.LastLoginAt, &created); err != nil {
		return nil, false, fmt.Errorf("upsert user: %w", err)
	}
	return &u, created, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*core.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, created_at, last_login_at FROM users WHERE id = $1
	`, id)
	var u core.User
	if err := row.Scan(&u.ID, &u.Email, &u.CreatedAt, &u.LastLoginAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}
```

- [ ] **Step 2: Create internal/database/users_test.go**

```go
package database_test

import (
	"context"
	"testing"

	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestUserRepo_CreateOrGet(t *testing.T) {
	pool := testdb(t)
	repo := database.NewUserRepo(&database.Pool{Pool: pool})
	ctx := context.Background()
	email := uniqueEmail(t)

	u1, created1, err := repo.CreateOrGetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if !created1 {
		t.Error("expected created=true on first call")
	}

	u2, created2, err := repo.CreateOrGetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if created2 {
		t.Error("expected created=false on second call")
	}
	if u1.ID != u2.ID {
		t.Errorf("ids differ: %s vs %s", u1.ID, u2.ID)
	}
}

func uniqueEmail(t *testing.T) string {
	t.Helper()
	return "u-" + t.Name() + "@test.example.com"
}
```

- [ ] **Step 3: Run tests**

```bash
GOTOOLCHAIN=local go test ./internal/database/... -run TestUserRepo -v
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/database/users.go internal/database/users_test.go
git commit -m "feat(database): UserRepo with idempotent CreateOrGetByEmail"
```

---

## Task 8: Organization + Membership repositories

**Files:**
- Create: `internal/database/organizations.go`
- Create: `internal/database/memberships.go`
- Create: `internal/database/organizations_test.go`

- [ ] **Step 1: Create internal/database/organizations.go**

```go
package database

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rupivbluegreen/pactline/internal/core"
)

type OrganizationRepo struct{ pool *Pool }

func NewOrganizationRepo(p *Pool) *OrganizationRepo { return &OrganizationRepo{pool: p} }

func (r *OrganizationRepo) Create(ctx context.Context, name, slug string) (*core.Organization, error) {
	if slug == "" {
		slug = SlugFromName(name)
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO organizations (slug, name) VALUES ($1, $2)
		RETURNING id, slug, name, created_at
	`, slug, name)
	var o core.Organization
	if err := row.Scan(&o.ID, &o.Slug, &o.Name, &o.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, core.ErrAlreadyExists
		}
		return nil, fmt.Errorf("insert org: %w", err)
	}
	return &o, nil
}

func (r *OrganizationRepo) GetByID(ctx context.Context, id uuid.UUID) (*core.Organization, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, slug, name, created_at FROM organizations WHERE id = $1`, id)
	var o core.Organization
	if err := row.Scan(&o.ID, &o.Slug, &o.Name, &o.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("get org: %w", err)
	}
	return &o, nil
}

var slugInvalid = regexp.MustCompile(`[^a-z0-9-]+`)

func SlugFromName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	s = slugInvalid.ReplaceAllString(s, "")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "org"
	}
	return s
}
```

- [ ] **Step 2: Create internal/database/memberships.go**

```go
package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rupivbluegreen/pactline/internal/core"
)

type MembershipRepo struct{ pool *Pool }

func NewMembershipRepo(p *Pool) *MembershipRepo { return &MembershipRepo{pool: p} }

func (r *MembershipRepo) Create(ctx context.Context, userID, orgID uuid.UUID, role core.Role) (*core.Membership, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO memberships (user_id, organization_id, role) VALUES ($1, $2, $3)
		RETURNING id, user_id, organization_id, role, created_at
	`, userID, orgID, string(role))
	var m core.Membership
	var roleStr string
	if err := row.Scan(&m.ID, &m.UserID, &m.OrganizationID, &roleStr, &m.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert membership: %w", err)
	}
	m.Role = core.Role(roleStr)
	return &m, nil
}

func (r *MembershipRepo) ListForUser(ctx context.Context, userID uuid.UUID) ([]core.Membership, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, organization_id, role, created_at
		FROM memberships WHERE user_id = $1 ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()

	var out []core.Membership
	for rows.Next() {
		var m core.Membership
		var roleStr string
		if err := rows.Scan(&m.ID, &m.UserID, &m.OrganizationID, &roleStr, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Role = core.Role(roleStr)
		out = append(out, m)
	}
	return out, rows.Err()
}
```

- [ ] **Step 3: Create internal/database/organizations_test.go**

```go
package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestSlugFromName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Acme Corp", "acme-corp"},
		{"  Hello World!  ", "hello-world"},
		{"!@#$", "org"},
		{"Mid Size Co.", "mid-size-co"},
	}
	for _, c := range cases {
		if got := database.SlugFromName(c.in); got != c.want {
			t.Errorf("SlugFromName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestOrgRepo_CreateAndConflict(t *testing.T) {
	pool := testdb(t)
	repo := database.NewOrganizationRepo(&database.Pool{Pool: pool})
	ctx := context.Background()

	o1, err := repo.Create(ctx, "Acme "+t.Name(), "")
	if err != nil {
		t.Fatalf("create 1: %v", err)
	}
	if o1.Slug == "" {
		t.Error("expected non-empty slug")
	}

	if _, err := repo.Create(ctx, "Other Name", o1.Slug); !errors.Is(err, core.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists on slug conflict, got %v", err)
	}
}

func TestMembershipRepo_Create_ListForUser(t *testing.T) {
	pool := testdb(t)
	uRepo := database.NewUserRepo(&database.Pool{Pool: pool})
	oRepo := database.NewOrganizationRepo(&database.Pool{Pool: pool})
	mRepo := database.NewMembershipRepo(&database.Pool{Pool: pool})
	ctx := context.Background()

	u, _, err := uRepo.CreateOrGetByEmail(ctx, "m-"+t.Name()+"@test.example.com")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	o, err := oRepo.Create(ctx, "Acme "+t.Name(), "")
	if err != nil {
		t.Fatalf("org: %v", err)
	}

	if _, err := mRepo.Create(ctx, u.ID, o.ID, core.RoleOwner); err != nil {
		t.Fatalf("create membership: %v", err)
	}

	ms, err := mRepo.ListForUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(ms) != 1 || ms[0].OrganizationID != o.ID || ms[0].Role != core.RoleOwner {
		t.Errorf("memberships: %+v", ms)
	}
}
```

- [ ] **Step 4: Run tests**

```bash
GOTOOLCHAIN=local go test ./internal/database/... -run 'TestSlug|TestOrgRepo|TestMembership' -v
```

Expected: 3 PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/database/organizations.go internal/database/memberships.go internal/database/organizations_test.go
git commit -m "feat(database): OrganizationRepo + MembershipRepo with slug generation"
```

---

## Task 9: Sessions + Magic links repositories

**Files:**
- Create: `internal/database/sessions.go`
- Create: `internal/database/magic_links.go`
- Create: `internal/database/sessions_test.go`

- [ ] **Step 1: Create internal/database/sessions.go**

```go
package database

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rupivbluegreen/pactline/internal/core"
)

type SessionRepo struct{ pool *Pool }

func NewSessionRepo(p *Pool) *SessionRepo { return &SessionRepo{pool: p} }

// SessionToken is the plaintext value handed to the client. Server stores
// only sha256(token) in sessions.token_hash.
type SessionToken string

func newRandomToken() (SessionToken, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return SessionToken(base64.RawURLEncoding.EncodeToString(buf)), nil
}

func HashToken(t SessionToken) []byte {
	sum := sha256.Sum256([]byte(t))
	return sum[:]
}

// Create issues a new session and returns the plaintext token (the
// server only stores its hash).
func (r *SessionRepo) Create(ctx context.Context, userID uuid.UUID, ttl time.Duration) (SessionToken, *core.Session, error) {
	tok, err := newRandomToken()
	if err != nil {
		return "", nil, fmt.Errorf("rand: %w", err)
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, expires_at, created_at, revoked_at
	`, userID, HashToken(tok), time.Now().Add(ttl))
	var s core.Session
	if err := row.Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt, &s.RevokedAt); err != nil {
		return "", nil, fmt.Errorf("insert session: %w", err)
	}
	return tok, &s, nil
}

// GetByToken returns the session for a presented plaintext token, or
// errors with ErrNotFound / ErrSessionRevoked / ErrTokenExpired.
func (r *SessionRepo) GetByToken(ctx context.Context, tok SessionToken) (*core.Session, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, expires_at, created_at, revoked_at
		FROM sessions WHERE token_hash = $1
	`, HashToken(tok))
	var s core.Session
	if err := row.Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt, &s.RevokedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	if s.RevokedAt != nil {
		return nil, core.ErrSessionRevoked
	}
	if time.Now().After(s.ExpiresAt) {
		return nil, core.ErrTokenExpired
	}
	return &s, nil
}

func (r *SessionRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}
```

- [ ] **Step 2: Create internal/database/magic_links.go**

```go
package database

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rupivbluegreen/pactline/internal/core"
)

type MagicLinkRepo struct{ pool *Pool }

func NewMagicLinkRepo(p *Pool) *MagicLinkRepo { return &MagicLinkRepo{pool: p} }

type MagicLinkToken string

func newMagicLinkToken() (MagicLinkToken, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return MagicLinkToken(base64.RawURLEncoding.EncodeToString(buf)), nil
}

func hashLinkToken(t MagicLinkToken) []byte {
	return HashToken(SessionToken(t)) // sha256 either way
}

// Create returns the plaintext magic-link token.
func (r *MagicLinkRepo) Create(ctx context.Context, email string, ttl time.Duration) (MagicLinkToken, error) {
	tok, err := newMagicLinkToken()
	if err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO magic_links (email, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, email, hashLinkToken(tok), time.Now().Add(ttl))
	if err != nil {
		return "", fmt.Errorf("insert magic link: %w", err)
	}
	return tok, nil
}

// Consume marks the link used and returns the email it was issued for.
// Errors with ErrInvalidToken / ErrTokenExpired / ErrTokenConsumed.
func (r *MagicLinkRepo) Consume(ctx context.Context, tok MagicLinkToken) (string, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT email, expires_at, used_at FROM magic_links
		WHERE token_hash = $1 FOR UPDATE
	`, hashLinkToken(tok))
	var email string
	var expiresAt time.Time
	var usedAt *time.Time
	if err := row.Scan(&email, &expiresAt, &usedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", core.ErrInvalidToken
		}
		return "", fmt.Errorf("query link: %w", err)
	}
	if usedAt != nil {
		return "", core.ErrTokenConsumed
	}
	if time.Now().After(expiresAt) {
		return "", core.ErrTokenExpired
	}
	if _, err := r.pool.Exec(ctx,
		`UPDATE magic_links SET used_at = now() WHERE token_hash = $1`,
		hashLinkToken(tok)); err != nil {
		return "", fmt.Errorf("mark used: %w", err)
	}
	return email, nil
}
```

- [ ] **Step 3: Create internal/database/sessions_test.go**

```go
package database_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestSessionRepo_CreateAndGet(t *testing.T) {
	pool := testdb(t)
	uRepo := database.NewUserRepo(&database.Pool{Pool: pool})
	sRepo := database.NewSessionRepo(&database.Pool{Pool: pool})
	ctx := context.Background()

	u, _, err := uRepo.CreateOrGetByEmail(ctx, "s-"+t.Name()+"@test.example.com")
	if err != nil {
		t.Fatalf("user: %v", err)
	}

	tok, sess, err := sRepo.Create(ctx, u.ID, time.Hour)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if tok == "" {
		t.Error("empty token")
	}

	got, err := sRepo.GetByToken(ctx, tok)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("id mismatch")
	}

	if err := sRepo.Revoke(ctx, sess.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := sRepo.GetByToken(ctx, tok); !errors.Is(err, core.ErrSessionRevoked) {
		t.Errorf("expected ErrSessionRevoked, got %v", err)
	}
}

func TestMagicLinkRepo_ConsumeOnce(t *testing.T) {
	pool := testdb(t)
	mRepo := database.NewMagicLinkRepo(&database.Pool{Pool: pool})
	ctx := context.Background()
	email := "ml-" + t.Name() + "@test.example.com"

	tok, err := mRepo.Create(ctx, email, 15*time.Minute)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := mRepo.Consume(ctx, tok)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if got != email {
		t.Errorf("email: got %s want %s", got, email)
	}

	if _, err := mRepo.Consume(ctx, tok); !errors.Is(err, core.ErrTokenConsumed) {
		t.Errorf("expected ErrTokenConsumed on second consume, got %v", err)
	}
}

func TestMagicLinkRepo_Expired(t *testing.T) {
	pool := testdb(t)
	mRepo := database.NewMagicLinkRepo(&database.Pool{Pool: pool})
	ctx := context.Background()

	tok, err := mRepo.Create(ctx, "exp-"+t.Name()+"@test.example.com", -time.Second)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := mRepo.Consume(ctx, tok); !errors.Is(err, core.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}
```

- [ ] **Step 4: Run tests**

```bash
GOTOOLCHAIN=local go test ./internal/database/... -run 'TestSessionRepo|TestMagicLinkRepo' -v
```

Expected: 3 PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/database/sessions.go internal/database/magic_links.go internal/database/sessions_test.go
git commit -m "feat(database): SessionRepo + MagicLinkRepo with sha256 token storage"
```

---

## Task 10: Email service (SMTP via Mailpit)

**Files:**
- Create: `internal/email/email.go`
- Create: `internal/email/email_test.go`

- [ ] **Step 1: Create internal/email/email.go**

```go
// Package email sends transactional emails. Defaults to Mailpit's SMTP
// listener in dev (localhost:1025) — config via env in prod.
package email

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
)

type Sender struct {
	host string
	port string
	from string
}

func NewSender() *Sender {
	return &Sender{
		host: envOr("SMTP_HOST", "localhost"),
		port: envOr("SMTP_PORT", "1025"),
		from: envOr("PACTLINE_FROM_EMAIL", "noreply@pactline.local"),
	}
}

func (s *Sender) Send(_ context.Context, to, subject, body string) error {
	addr := s.host + ":" + s.port
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body,
	))
	return smtp.SendMail(addr, nil, s.from, []string{to}, msg)
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 2: Create internal/email/email_test.go**

```go
package email_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/email"
)

func TestSender_DeliversToMailpit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping smtp integration in short mode")
	}
	s := email.NewSender()
	to := "mailpit-test-" + t.Name() + "@example.com"
	subject := "pactline-test-" + t.Name()

	if err := s.Send(context.Background(), to, subject, "hello from test"); err != nil {
		t.Fatalf("send: %v", err)
	}

	// Poll mailpit for the message.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:8025/api/v1/messages?query=" + subject)
		if err == nil {
			defer resp.Body.Close()
			var body struct {
				Messages []struct {
					To []struct{ Address string } `json:"To"`
				} `json:"messages"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err == nil {
				for _, m := range body.Messages {
					for _, addr := range m.To {
						if strings.EqualFold(addr.Address, to) {
							return
						}
					}
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("message not found in mailpit")
}
```

- [ ] **Step 3: Run test**

```bash
GOTOOLCHAIN=local go test ./internal/email/... -v
```

Expected: PASS (Mailpit on 1025 / 8025).

- [ ] **Step 4: Commit**

```bash
git add internal/email/email.go internal/email/email_test.go
git commit -m "feat(email): Mailpit-backed SMTP sender"
```

---

## Task 11: Auth service — request + verify magic link, logout

**Files:**
- Create: `internal/auth/auth.go`
- Create: `internal/auth/auth_test.go`

- [ ] **Step 1: Create internal/auth/auth.go**

```go
// Package auth issues magic links and exchanges them for sessions.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rupivbluegreen/pactline/internal/audit"
	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/email"
)

const (
	MagicLinkTTL = 15 * time.Minute
	SessionTTL   = 30 * 24 * time.Hour
)

type Service struct {
	users   *database.UserRepo
	links   *database.MagicLinkRepo
	sess    *database.SessionRepo
	mails   *email.Sender
	pool    *database.Pool
	baseURL string
}

func NewService(
	pool *database.Pool, users *database.UserRepo, links *database.MagicLinkRepo,
	sess *database.SessionRepo, mails *email.Sender, baseURL string,
) *Service {
	return &Service{users: users, links: links, sess: sess, mails: mails, pool: pool, baseURL: baseURL}
}

// RequestMagicLink creates the user (if new), creates a magic_link, emails the URL.
func (s *Service) RequestMagicLink(ctx context.Context, emailAddr string) error {
	emailAddr = strings.TrimSpace(strings.ToLower(emailAddr))
	if !strings.Contains(emailAddr, "@") {
		return fmt.Errorf("invalid email")
	}

	u, created, err := s.users.CreateOrGetByEmail(ctx, emailAddr)
	if err != nil {
		return err
	}
	if created {
		if err := audit.Write(ctx, s.pool, audit.Event{
			ActorUserID: &u.ID,
			Action:      audit.ActionUserCreated,
			EntityType:  "user",
			EntityID:    &u.ID,
			After:       map[string]string{"email": u.Email},
		}); err != nil {
			return err
		}
	}

	tok, err := s.links.Create(ctx, emailAddr, MagicLinkTTL)
	if err != nil {
		return err
	}
	if err := audit.Write(ctx, s.pool, audit.Event{
		ActorUserID: &u.ID,
		Action:      audit.ActionMagicLinkRequested,
		EntityType:  "magic_link",
		After:       map[string]string{"email": emailAddr},
	}); err != nil {
		return err
	}

	link := s.baseURL + "/auth/verify?token=" + string(tok)
	body := "Sign in to Pactline:\n\n" + link + "\n\nThis link expires in 15 minutes."
	return s.mails.Send(ctx, emailAddr, "Pactline sign-in link", body)
}

// VerifyResult is everything the client needs after a successful verify.
type VerifyResult struct {
	SessionToken database.SessionToken
	User         *core.User
	ExpiresAt    time.Time
}

func (s *Service) VerifyMagicLink(ctx context.Context, tok database.MagicLinkToken) (*VerifyResult, error) {
	emailAddr, err := s.links.Consume(ctx, tok)
	if err != nil {
		return nil, err
	}
	u, _, err := s.users.CreateOrGetByEmail(ctx, emailAddr)
	if err != nil {
		return nil, err
	}
	if err := audit.Write(ctx, s.pool, audit.Event{
		ActorUserID: &u.ID,
		Action:      audit.ActionMagicLinkConsumed,
		EntityType:  "magic_link",
		After:       map[string]string{"email": emailAddr},
	}); err != nil {
		return nil, err
	}
	if err := s.users.UpdateLastLogin(ctx, u.ID); err != nil {
		return nil, err
	}

	sessTok, sess, err := s.sess.Create(ctx, u.ID, SessionTTL)
	if err != nil {
		return nil, err
	}
	if err := audit.Write(ctx, s.pool, audit.Event{
		ActorUserID: &u.ID,
		Action:      audit.ActionSessionCreated,
		EntityType:  "session",
		EntityID:    &sess.ID,
	}); err != nil {
		return nil, err
	}

	return &VerifyResult{SessionToken: sessTok, User: u, ExpiresAt: sess.ExpiresAt}, nil
}

func (s *Service) Logout(ctx context.Context, tok database.SessionToken) error {
	sess, err := s.sess.GetByToken(ctx, tok)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) || errors.Is(err, core.ErrSessionRevoked) || errors.Is(err, core.ErrTokenExpired) {
			return nil
		}
		return err
	}
	if err := s.sess.Revoke(ctx, sess.ID); err != nil {
		return err
	}
	if err := audit.Write(ctx, s.pool, audit.Event{
		ActorUserID: &sess.UserID,
		Action:      audit.ActionSessionRevoked,
		EntityType:  "session",
		EntityID:    &sess.ID,
	}); err != nil {
		return err
	}
	return nil
}
```

- [ ] **Step 2: Create internal/auth/auth_test.go**

```go
package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/auth"
	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/email"
)

func newSvc(t *testing.T) (*auth.Service, *database.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		t.Skipf("db unavailable: %v", err)
	}
	if err := database.MigrateUp(ctx, p); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { p.Close() })

	users := database.NewUserRepo(p)
	links := database.NewMagicLinkRepo(p)
	sess := database.NewSessionRepo(p)
	return auth.NewService(p, users, links, sess, email.NewSender(), "http://localhost:8000"), p
}

func extractTokenFromMailpit(t *testing.T, to string) database.MagicLinkToken {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:8025/api/v1/search?query=to%3A" + to)
		if err == nil {
			defer resp.Body.Close()
			var body struct {
				Messages []struct {
					ID string `json:"ID"`
				} `json:"messages"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err == nil && len(body.Messages) > 0 {
				resp2, err := http.Get("http://localhost:8025/api/v1/message/" + body.Messages[0].ID)
				if err == nil {
					defer resp2.Body.Close()
					var msg struct {
						Text string `json:"Text"`
					}
					if err := json.NewDecoder(resp2.Body).Decode(&msg); err == nil {
						i := strings.Index(msg.Text, "token=")
						if i >= 0 {
							rest := msg.Text[i+len("token="):]
							end := strings.IndexAny(rest, " \r\n")
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
	svc, _ := newSvc(t)
	ctx := context.Background()
	addr := "auth-" + t.Name() + "@example.com"

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

	// Second verify with same token must fail.
	if _, err := svc.VerifyMagicLink(ctx, tok); !errors.Is(err, core.ErrTokenConsumed) {
		t.Errorf("expected ErrTokenConsumed, got %v", err)
	}
}

func TestLogout(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	addr := "logout-" + t.Name() + "@example.com"

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
```

- [ ] **Step 3: Run tests**

```bash
GOTOOLCHAIN=local go test ./internal/auth/... -v
```

Expected: 2 PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/auth/
git commit -m "feat(auth): magic-link request + verify + logout with audit events"
```

---

## Task 12: Auth middleware (cookie + bearer)

**Files:**
- Create: `internal/api/middleware/auth.go`
- Create: `internal/api/middleware/auth_test.go`

- [ ] **Step 1: Create internal/api/middleware/auth.go**

```go
// Package middleware holds chi-compatible HTTP middleware for the API.
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

const SessionCookieName = "pactline_session"

// Auth attaches the user (and current organization, if any) to the request
// context. Returns 401 if no valid session.
func Auth(sessions *database.SessionRepo, memberships *database.MembershipRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := tokenFromRequest(r)
			if tok == "" {
				http.Error(w, "unauthenticated", http.StatusUnauthorized)
				return
			}
			sess, err := sessions.GetByToken(r.Context(), database.SessionToken(tok))
			if err != nil {
				switch {
				case errors.Is(err, core.ErrNotFound),
					errors.Is(err, core.ErrSessionRevoked),
					errors.Is(err, core.ErrTokenExpired):
					http.Error(w, "unauthenticated", http.StatusUnauthorized)
				default:
					http.Error(w, "internal error", http.StatusInternalServerError)
				}
				return
			}

			ctx := database.WithUserID(r.Context(), sess.UserID)

			ms, err := memberships.ListForUser(ctx, sess.UserID)
			if err == nil && len(ms) > 0 {
				ctx = database.WithOrgID(ctx, ms[0].OrganizationID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func tokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(SessionCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
```

- [ ] **Step 2: Create internal/api/middleware/auth_test.go**

```go
package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/api/middleware"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func mustPool(t *testing.T) *database.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		t.Skipf("db unavailable: %v", err)
	}
	if err := database.MigrateUp(ctx, p); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { p.Close() })
	return p
}

func TestAuth_NoToken(t *testing.T) {
	p := mustPool(t)
	mw := middleware.Auth(database.NewSessionRepo(p), database.NewMembershipRepo(p))
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuth_ValidBearer(t *testing.T) {
	p := mustPool(t)
	users := database.NewUserRepo(p)
	sess := database.NewSessionRepo(p)
	memb := database.NewMembershipRepo(p)
	ctx := context.Background()

	u, _, _ := users.CreateOrGetByEmail(ctx, "mw-"+t.Name()+"@example.com")
	tok, _, err := sess.Create(ctx, u.ID, time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	called := false
	mw := middleware.Auth(sess, memb)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		uid, ok := database.UserID(r.Context())
		if !ok || uid != u.ID {
			t.Errorf("ctx user mismatch: got=%v ok=%v", uid, ok)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+string(tok))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !called {
		t.Error("handler not called")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
GOTOOLCHAIN=local go test ./internal/api/middleware/... -v
```

Expected: 2 PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/api/middleware/
git commit -m "feat(api): bearer/cookie auth middleware"
```

---

## Task 13: Auth handlers + /me

**Files:**
- Create: `internal/api/handlers/auth_handlers.go`
- Create: `internal/api/handlers/me.go`
- Create: `internal/api/handlers/auth_handlers_test.go`

- [ ] **Step 1: Create internal/api/handlers/auth_handlers.go**

```go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/rupivbluegreen/pactline/internal/api/middleware"
	"github.com/rupivbluegreen/pactline/internal/auth"
	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

type AuthHandlers struct {
	Svc *auth.Service
}

type magicLinkRequestBody struct {
	Email string `json:"email"`
}

func (h *AuthHandlers) RequestMagicLink(w http.ResponseWriter, r *http.Request) {
	var body magicLinkRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := h.Svc.RequestMagicLink(r.Context(), body.Email); err != nil {
		http.Error(w, "request failed", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type verifyRequestBody struct {
	Token string `json:"token"`
}

type verifyResponse struct {
	SessionToken string    `json:"session_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       string    `json:"user_id"`
	UserEmail    string    `json:"user_email"`
}

func (h *AuthHandlers) Verify(w http.ResponseWriter, r *http.Request) {
	var body verifyRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	res, err := h.Svc.VerifyMagicLink(r.Context(), database.MagicLinkToken(body.Token))
	if err != nil {
		switch {
		case errors.Is(err, core.ErrInvalidToken),
			errors.Is(err, core.ErrTokenExpired),
			errors.Is(err, core.ErrTokenConsumed):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    string(res.SessionToken),
		Expires:  res.ExpiresAt,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, verifyResponse{
		SessionToken: string(res.SessionToken),
		ExpiresAt:    res.ExpiresAt,
		UserID:       res.User.ID.String(),
		UserEmail:    res.User.Email,
	})
}

func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	tok, _ := r.Cookie(middleware.SessionCookieName)
	if tok != nil {
		_ = h.Svc.Logout(r.Context(), database.SessionToken(tok.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Expires:  time.Unix(0, 0),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 2: Create internal/api/handlers/me.go**

```go
package handlers

import (
	"net/http"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

type MeHandler struct {
	Users       *database.UserRepo
	Memberships *database.MembershipRepo
	Orgs        *database.OrganizationRepo
}

type meResponse struct {
	UserID         string                  `json:"user_id"`
	UserEmail      string                  `json:"user_email"`
	CurrentOrgID   *string                 `json:"current_organization_id,omitempty"`
	CurrentOrgName *string                 `json:"current_organization_name,omitempty"`
	Memberships    []membershipResponseRow `json:"memberships"`
}

type membershipResponseRow struct {
	OrganizationID   string `json:"organization_id"`
	OrganizationSlug string `json:"organization_slug"`
	OrganizationName string `json:"organization_name"`
	Role             string `json:"role"`
}

func (h *MeHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid, ok := database.UserID(r.Context())
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	u, err := h.Users.GetByID(r.Context(), uid)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ms, err := h.Memberships.ListForUser(r.Context(), uid)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	out := meResponse{UserID: u.ID.String(), UserEmail: u.Email}
	for _, m := range ms {
		o, err := h.Orgs.GetByID(r.Context(), m.OrganizationID)
		if err != nil {
			continue
		}
		out.Memberships = append(out.Memberships, membershipResponseRow{
			OrganizationID: o.ID.String(), OrganizationSlug: o.Slug,
			OrganizationName: o.Name, Role: string(m.Role),
		})
	}
	if orgID, ok := database.TenantScoped(r.Context()); ok {
		if o, err := h.Orgs.GetByID(r.Context(), orgID); err == nil {
			id := o.ID.String()
			out.CurrentOrgID = &id
			out.CurrentOrgName = &o.Name
		}
	}
	_ = core.User{} // silence unused import if any
	writeJSON(w, http.StatusOK, out)
}
```

- [ ] **Step 3: Create internal/api/handlers/auth_handlers_test.go**

```go
package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/api/handlers"
	"github.com/rupivbluegreen/pactline/internal/auth"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/email"
)

func newSvc(t *testing.T) *auth.Service {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		t.Skipf("db unavailable: %v", err)
	}
	if err := database.MigrateUp(ctx, p); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { p.Close() })
	return auth.NewService(p,
		database.NewUserRepo(p), database.NewMagicLinkRepo(p),
		database.NewSessionRepo(p), email.NewSender(), "http://localhost:8000",
	)
}

func TestAuthHandlers_Request(t *testing.T) {
	h := &handlers.AuthHandlers{Svc: newSvc(t)}
	body := `{"email":"req-` + t.Name() + `@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/magic-link/request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.RequestMagicLink(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Errorf("got %d, want 202", rec.Code)
	}
}

func TestAuthHandlers_VerifyBadToken(t *testing.T) {
	h := &handlers.AuthHandlers{Svc: newSvc(t)}
	body := `{"token":"definitely-not-a-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/magic-link/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Verify(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rec.Code)
	}
	var errBody map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&errBody)
}
```

- [ ] **Step 4: Run tests**

```bash
GOTOOLCHAIN=local go test ./internal/api/handlers/... -v
```

Expected: existing handler tests + 2 new PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/api/handlers/auth_handlers.go internal/api/handlers/me.go internal/api/handlers/auth_handlers_test.go
git commit -m "feat(api): magic-link request/verify/logout handlers + GET /me"
```

---

## Task 14: Organizations service + handler

**Files:**
- Create: `internal/organizations/organizations.go`
- Create: `internal/organizations/organizations_test.go`
- Create: `internal/api/handlers/organizations.go`

- [ ] **Step 1: Create internal/organizations/organizations.go**

```go
// Package organizations creates orgs with an owner membership atomically.
package organizations

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rupivbluegreen/pactline/internal/audit"
	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

type Service struct {
	pool *database.Pool
	orgs *database.OrganizationRepo
	mems *database.MembershipRepo
}

func NewService(pool *database.Pool, orgs *database.OrganizationRepo, mems *database.MembershipRepo) *Service {
	return &Service{pool: pool, orgs: orgs, mems: mems}
}

// CreateForUser creates the org, adds the user as owner, and writes audit
// events. Both rows are committed in one transaction.
func (s *Service) CreateForUser(ctx context.Context, userID uuid.UUID, name, slug string) (*core.Organization, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if slug == "" {
		slug = database.SlugFromName(name)
	}
	row := tx.QueryRow(ctx, `
		INSERT INTO organizations (slug, name) VALUES ($1, $2)
		RETURNING id, slug, name, created_at
	`, slug, name)
	var o core.Organization
	if err := row.Scan(&o.ID, &o.Slug, &o.Name, &o.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert org: %w", err)
	}

	mrow := tx.QueryRow(ctx, `
		INSERT INTO memberships (user_id, organization_id, role) VALUES ($1, $2, 'owner')
		RETURNING id
	`, userID, o.ID)
	var memID uuid.UUID
	if err := mrow.Scan(&memID); err != nil {
		return nil, fmt.Errorf("insert membership: %w", err)
	}

	if err := audit.Write(ctx, tx, audit.Event{
		OrganizationID: &o.ID, ActorUserID: &userID,
		Action: audit.ActionOrganizationCreated, EntityType: "organization",
		EntityID: &o.ID, After: map[string]string{"slug": o.Slug, "name": o.Name},
	}); err != nil {
		return nil, err
	}
	if err := audit.Write(ctx, tx, audit.Event{
		OrganizationID: &o.ID, ActorUserID: &userID,
		Action: audit.ActionMembershipCreated, EntityType: "membership",
		EntityID: &memID, After: map[string]string{"role": "owner"},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &o, nil
}
```

- [ ] **Step 2: Create internal/organizations/organizations_test.go**

```go
package organizations_test

import (
	"context"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/organizations"
)

func TestCreateForUser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		t.Skipf("db unavailable: %v", err)
	}
	if err := database.MigrateUp(ctx, p); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer p.Close()

	users := database.NewUserRepo(p)
	orgs := database.NewOrganizationRepo(p)
	mems := database.NewMembershipRepo(p)
	svc := organizations.NewService(p, orgs, mems)

	u, _, _ := users.CreateOrGetByEmail(ctx, "org-"+t.Name()+"@example.com")
	o, err := svc.CreateForUser(ctx, u.ID, "Acme "+t.Name(), "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if o.Slug == "" {
		t.Error("empty slug")
	}

	ms, _ := mems.ListForUser(ctx, u.ID)
	found := false
	for _, m := range ms {
		if m.OrganizationID == o.ID && m.Role == "owner" {
			found = true
		}
	}
	if !found {
		t.Errorf("owner membership not found: %+v", ms)
	}
}
```

- [ ] **Step 3: Create internal/api/handlers/organizations.go**

```go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/organizations"
)

type OrgsHandlers struct {
	Svc *organizations.Service
}

type createOrgBody struct {
	Name string `json:"name"`
	Slug string `json:"slug,omitempty"`
}

type orgResponse struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func (h *OrgsHandlers) Create(w http.ResponseWriter, r *http.Request) {
	uid, ok := database.UserID(r.Context())
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	var body createOrgBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	o, err := h.Svc.CreateForUser(r.Context(), uid, body.Name, body.Slug)
	if err != nil {
		if errors.Is(err, core.ErrAlreadyExists) {
			http.Error(w, "slug taken", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, orgResponse{ID: o.ID.String(), Slug: o.Slug, Name: o.Name})
}
```

- [ ] **Step 4: Run tests**

```bash
GOTOOLCHAIN=local go test ./internal/organizations/... -v
GOTOOLCHAIN=local go build ./...
```

Expected: PASS, clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/organizations/ internal/api/handlers/organizations.go
git commit -m "feat(organizations): create-for-user with owner membership in one tx"
```

---

## Task 15: Contracts list handler (empty) + multi-tenant isolation test

**Files:**
- Create: `internal/api/handlers/contracts.go`
- Create: `internal/api/handlers/contracts_test.go`

- [ ] **Step 1: Create internal/api/handlers/contracts.go**

```go
package handlers

import (
	"net/http"

	"github.com/rupivbluegreen/pactline/internal/database"
)

type ContractsHandler struct{}

type contractsResponse struct {
	Contracts []any `json:"contracts"`
}

func (h *ContractsHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := database.TenantScoped(r.Context()); !ok {
		http.Error(w, "no organization", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, contractsResponse{Contracts: []any{}})
}
```

- [ ] **Step 2: Create internal/api/handlers/contracts_test.go**

```go
package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rupivbluegreen/pactline/internal/api/handlers"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestContractsList_NoOrg(t *testing.T) {
	h := &handlers.ContractsHandler{}
	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/contracts", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestContractsList_Empty(t *testing.T) {
	h := &handlers.ContractsHandler{}
	ctx := database.WithOrgID(context.Background(), uuid.New())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contracts", nil).WithContext(ctx)
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Error("empty body")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
GOTOOLCHAIN=local go test ./internal/api/handlers/... -run TestContracts -v
```

Expected: 2 PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/api/handlers/contracts.go internal/api/handlers/contracts_test.go
git commit -m "feat(api): GET /contracts (empty list, tenant-scoped)"
```

---

## Task 16: Wire main.go + router + OpenAPI spec

**Files:**
- Modify: `cmd/api/main.go`
- Modify: `internal/api/router.go`
- Modify: `schemas/openapi.yaml`
- Run: `make generate-types`

- [ ] **Step 1: Replace internal/api/router.go**

```go
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/rupivbluegreen/pactline/internal/api/handlers"
	authmw "github.com/rupivbluegreen/pactline/internal/api/middleware"
	"github.com/rupivbluegreen/pactline/internal/database"
)

type Deps struct {
	Sessions    *database.SessionRepo
	Memberships *database.MembershipRepo

	Auth      *handlers.AuthHandlers
	Me        *handlers.MeHandler
	Orgs      *handlers.OrgsHandlers
	Contracts *handlers.ContractsHandler
}

func Router(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", handlers.Health)
	r.Get("/hello", handlers.Hello)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/magic-link/request", d.Auth.RequestMagicLink)
		r.Post("/magic-link/verify", d.Auth.Verify)
		r.Post("/logout", d.Auth.Logout)
	})

	r.Group(func(r chi.Router) {
		r.Use(authmw.Auth(d.Sessions, d.Memberships))
		r.Get("/me", d.Me.Me)
		r.Post("/organizations", d.Orgs.Create)
		r.Get("/contracts", d.Contracts.List)
	})

	return r
}
```

- [ ] **Step 2: Replace cmd/api/main.go**

```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rupivbluegreen/pactline/internal/api"
	"github.com/rupivbluegreen/pactline/internal/api/handlers"
	"github.com/rupivbluegreen/pactline/internal/auth"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/email"
	"github.com/rupivbluegreen/pactline/internal/organizations"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	addr := envOr("PACTLINE_API_ADDR", ":8000")
	baseURL := envOr("PACTLINE_BASE_URL", "http://localhost:3000")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.MigrateUp(ctx, pool); err != nil {
		slog.Error("migrate up", "err", err)
		os.Exit(1)
	}

	users := database.NewUserRepo(pool)
	orgs := database.NewOrganizationRepo(pool)
	mems := database.NewMembershipRepo(pool)
	sess := database.NewSessionRepo(pool)
	links := database.NewMagicLinkRepo(pool)
	mails := email.NewSender()

	authSvc := auth.NewService(pool, users, links, sess, mails, baseURL)
	orgSvc := organizations.NewService(pool, orgs, mems)

	deps := api.Deps{
		Sessions:    sess,
		Memberships: mems,
		Auth:        &handlers.AuthHandlers{Svc: authSvc},
		Me:          &handlers.MeHandler{Users: users, Memberships: mems, Orgs: orgs},
		Orgs:        &handlers.OrgsHandlers{Svc: orgSvc},
		Contracts:   &handlers.ContractsHandler{},
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.Router(deps),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("api listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("api crashed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("api shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("api shutdown failed", "err", err)
	}
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 3: Update existing handler test for the new Router signature**

The previous `internal/api/handlers/handlers_test.go` calls `api.Router()` with no args. Update to:

```go
api.Router(api.Deps{
    Auth:      &handlers.AuthHandlers{},
    Me:        &handlers.MeHandler{},
    Orgs:      &handlers.OrgsHandlers{},
    Contracts: &handlers.ContractsHandler{},
})
```

But the existing test only exercises `/healthz` and `/hello` which don't use any deps. Pass zero-value deps (all nils). The handlers in those routes don't dereference deps, so zero value is fine.

Actually that's still a test correctness issue — middleware-protected routes with nil Sessions/Memberships will panic when called. The unauthenticated `/healthz` and `/hello` are fine; they bypass the auth group.

Update `internal/api/handlers/handlers_test.go` to use:

```go
import (
    "github.com/rupivbluegreen/pactline/internal/api/handlers"
)

func newRouter() http.Handler {
    return api.Router(api.Deps{})  // health and hello don't touch deps
}
```

And replace the three calls to `api.Router()` with `newRouter()`.

- [ ] **Step 4: Update schemas/openapi.yaml**

Replace contents with:

```yaml
openapi: 3.1.0
info:
  title: Pactline API
  version: 0.0.1
  description: |
    Contract lifecycle platform built on durable workflows.
servers:
  - url: http://localhost:8000
    description: Local development
paths:
  /healthz:
    get:
      operationId: healthz
      tags: [health]
      summary: Liveness probe
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: { $ref: '#/components/schemas/HealthResponse' }
  /hello:
    get:
      operationId: hello
      tags: [meta]
      summary: Hello-world demonstrating the API → frontend type pipeline
      parameters:
        - { in: query, name: name, required: false, schema: { type: string, default: world } }
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: { $ref: '#/components/schemas/HelloResponse' }
  /auth/magic-link/request:
    post:
      operationId: requestMagicLink
      tags: [auth]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [email]
              properties:
                email: { type: string, format: email }
      responses:
        '202': { description: Email queued }
        '400': { description: Bad request }
  /auth/magic-link/verify:
    post:
      operationId: verifyMagicLink
      tags: [auth]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [token]
              properties:
                token: { type: string }
      responses:
        '200':
          description: OK + Set-Cookie session
          content:
            application/json:
              schema: { $ref: '#/components/schemas/VerifyResponse' }
        '400': { description: Bad request }
  /auth/logout:
    post:
      operationId: logout
      tags: [auth]
      responses:
        '204': { description: Logged out }
  /me:
    get:
      operationId: me
      tags: [auth]
      security: [{ session: [] }]
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: { $ref: '#/components/schemas/MeResponse' }
        '401': { description: Unauthenticated }
  /organizations:
    post:
      operationId: createOrganization
      tags: [organizations]
      security: [{ session: [] }]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name: { type: string }
                slug: { type: string }
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema: { $ref: '#/components/schemas/OrgResponse' }
        '401': { description: Unauthenticated }
        '409': { description: Slug taken }
  /contracts:
    get:
      operationId: listContracts
      tags: [contracts]
      security: [{ session: [] }]
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ContractsResponse' }
        '400': { description: No organization context }
        '401': { description: Unauthenticated }
components:
  securitySchemes:
    session:
      type: apiKey
      in: cookie
      name: pactline_session
  schemas:
    HealthResponse:
      type: object
      required: [status]
      properties: { status: { type: string } }
    HelloResponse:
      type: object
      required: [message]
      properties: { message: { type: string } }
    VerifyResponse:
      type: object
      required: [session_token, expires_at, user_id, user_email]
      properties:
        session_token: { type: string }
        expires_at: { type: string, format: date-time }
        user_id: { type: string, format: uuid }
        user_email: { type: string, format: email }
    MeResponse:
      type: object
      required: [user_id, user_email, memberships]
      properties:
        user_id: { type: string, format: uuid }
        user_email: { type: string, format: email }
        current_organization_id: { type: string, format: uuid }
        current_organization_name: { type: string }
        memberships:
          type: array
          items: { $ref: '#/components/schemas/MembershipRow' }
    MembershipRow:
      type: object
      required: [organization_id, organization_slug, organization_name, role]
      properties:
        organization_id: { type: string, format: uuid }
        organization_slug: { type: string }
        organization_name: { type: string }
        role: { type: string, enum: [owner, admin, reviewer, approver, viewer] }
    OrgResponse:
      type: object
      required: [id, slug, name]
      properties:
        id: { type: string, format: uuid }
        slug: { type: string }
        name: { type: string }
    ContractsResponse:
      type: object
      required: [contracts]
      properties:
        contracts:
          type: array
          items: { type: object }
```

- [ ] **Step 5: Run type generation**

```bash
make generate-types
```

Expected: `apps/web/src/api/types.ts` updated.

- [ ] **Step 6: Run all backend tests + verify build**

```bash
GOTOOLCHAIN=local go build ./...
GOTOOLCHAIN=local go test ./...
```

Expected: clean build, all tests PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/api/main.go internal/api/ schemas/openapi.yaml apps/web/src/api/types.ts
git commit -m "feat(api): wire all phase-1-story-1 routes; OpenAPI spec + regenerated TS types"
```

---

## Task 17: Multi-tenant isolation integration test

**Files:**
- Create: `internal/api/integration_test.go`

- [ ] **Step 1: Create integration test that proves tenant isolation**

```go
package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/api"
	"github.com/rupivbluegreen/pactline/internal/api/handlers"
	"github.com/rupivbluegreen/pactline/internal/auth"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/email"
	"github.com/rupivbluegreen/pactline/internal/organizations"
)

func TestTenantIsolation_TwoUsersTwoOrgs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		t.Skipf("db unavailable: %v", err)
	}
	if err := database.MigrateUp(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer pool.Close()

	users := database.NewUserRepo(pool)
	orgs := database.NewOrganizationRepo(pool)
	mems := database.NewMembershipRepo(pool)
	sess := database.NewSessionRepo(pool)
	links := database.NewMagicLinkRepo(pool)
	authSvc := auth.NewService(pool, users, links, sess, email.NewSender(), "http://localhost:8000")
	orgSvc := organizations.NewService(pool, orgs, mems)

	router := api.Router(api.Deps{
		Sessions: sess, Memberships: mems,
		Auth:      &handlers.AuthHandlers{Svc: authSvc},
		Me:        &handlers.MeHandler{Users: users, Memberships: mems, Orgs: orgs},
		Orgs:      &handlers.OrgsHandlers{Svc: orgSvc},
		Contracts: &handlers.ContractsHandler{},
	})

	user1 := setupUserWithOrg(t, ctx, pool, sess, mems, users, orgSvc, "iso1-"+t.Name(), "Org One "+t.Name())
	user2 := setupUserWithOrg(t, ctx, pool, sess, mems, users, orgSvc, "iso2-"+t.Name(), "Org Two "+t.Name())

	// User1 fetches /me — sees Org One only.
	me1 := callMe(t, router, user1.token)
	if !strings.Contains(me1, "Org One") || strings.Contains(me1, "Org Two") {
		t.Errorf("user1 /me leaked: %s", me1)
	}
	me2 := callMe(t, router, user2.token)
	if !strings.Contains(me2, "Org Two") || strings.Contains(me2, "Org One") {
		t.Errorf("user2 /me leaked: %s", me2)
	}

	// User1 fetches /contracts — empty list, but should be empty for *their* org only.
	c1 := callContracts(t, router, user1.token)
	if c1 != `{"contracts":[]}` && !strings.Contains(c1, `"contracts":[]`) {
		t.Errorf("user1 contracts unexpected: %s", c1)
	}
}

type seedUser struct {
	token database.SessionToken
}

func setupUserWithOrg(
	t *testing.T, ctx context.Context, pool *database.Pool,
	sess *database.SessionRepo, mems *database.MembershipRepo, users *database.UserRepo,
	orgSvc *organizations.Service, emailSeed, orgName string,
) seedUser {
	t.Helper()
	u, _, err := users.CreateOrGetByEmail(ctx, emailSeed+"@example.com")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	if _, err := orgSvc.CreateForUser(ctx, u.ID, orgName, ""); err != nil {
		t.Fatalf("org: %v", err)
	}
	tok, _, err := sess.Create(ctx, u.ID, time.Hour)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	return seedUser{token: tok}
}

func callMe(t *testing.T, router http.Handler, tok database.SessionToken) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+string(tok))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("/me code: %d body: %s", rec.Code, rec.Body.String())
	}
	var pretty map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&pretty)
	b, _ := json.Marshal(pretty)
	return string(b)
}

func callContracts(t *testing.T, router http.Handler, tok database.SessionToken) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/contracts", nil)
	req.Header.Set("Authorization", "Bearer "+string(tok))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("/contracts code: %d body: %s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}
```

- [ ] **Step 2: Run the integration test**

```bash
GOTOOLCHAIN=local go test ./internal/api/... -run TestTenantIsolation -v
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/api/integration_test.go
git commit -m "test(api): two-user/two-org tenant isolation integration test"
```

---

## Task 18: Frontend — signin flow

**Files:**
- Modify: `apps/web/src/app/page.tsx`
- Create: `apps/web/src/app/signin/page.tsx`
- Create: `apps/web/src/app/signin/sent/page.tsx`
- Create: `apps/web/src/app/auth/verify/page.tsx`
- Create: `apps/web/src/lib/server-session.ts`

- [ ] **Step 1: Create apps/web/src/lib/server-session.ts**

```ts
import { cookies } from "next/headers";
import { client } from "@/api/client";

export async function getMe() {
  const cookieStore = await cookies();
  const session = cookieStore.get("pactline_session")?.value;
  if (!session) return null;

  const { data, error } = await client.GET("/me", {
    headers: { Authorization: `Bearer ${session}` },
  });
  if (error) return null;
  return data;
}
```

- [ ] **Step 2: Create apps/web/src/app/signin/page.tsx**

```tsx
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { client } from "@/api/client";

export default function SignIn() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-6 p-8">
      <h1 className="text-3xl font-semibold">Pactline</h1>
      <form
        className="flex w-80 flex-col gap-3"
        onSubmit={async (e) => {
          e.preventDefault();
          setSubmitting(true);
          setErr(null);
          const { error } = await client.POST("/auth/magic-link/request", {
            body: { email },
          });
          setSubmitting(false);
          if (error) {
            setErr(String(error));
          } else {
            router.push("/signin/sent");
          }
        }}
      >
        <label className="text-sm font-medium">Work email</label>
        <input
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          className="rounded-md border border-neutral-300 px-3 py-2"
          placeholder="you@company.com"
        />
        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-neutral-900 px-3 py-2 text-white disabled:opacity-50"
        >
          {submitting ? "Sending..." : "Send sign-in link"}
        </button>
        {err && <p className="text-sm text-red-600">{err}</p>}
      </form>
    </main>
  );
}
```

- [ ] **Step 3: Create apps/web/src/app/signin/sent/page.tsx**

```tsx
export default function SignInSent() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-4 p-8">
      <h1 className="text-2xl font-semibold">Check your email</h1>
      <p className="text-sm text-neutral-500">
        We sent a sign-in link. Open it on this device to continue.
      </p>
      <p className="text-xs text-neutral-400">
        (Dev: open Mailpit at <a className="underline" href="http://localhost:8025">localhost:8025</a>.)
      </p>
    </main>
  );
}
```

- [ ] **Step 4: Create apps/web/src/app/auth/verify/page.tsx**

```tsx
import { redirect } from "next/navigation";
import { client } from "@/api/client";
import { cookies } from "next/headers";

export const dynamic = "force-dynamic";

export default async function Verify({
  searchParams,
}: {
  searchParams: Promise<{ token?: string }>;
}) {
  const { token } = await searchParams;
  if (!token) {
    redirect("/signin");
  }

  const { data, error } = await client.POST("/auth/magic-link/verify", {
    body: { token: token! },
  });
  if (error || !data) {
    redirect("/signin?error=invalid");
  }

  const cookieStore = await cookies();
  cookieStore.set("pactline_session", data!.session_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    expires: new Date(data!.expires_at),
    path: "/",
  });

  // /me to find out whether user already has an org
  const meRes = await client.GET("/me", {
    headers: { Authorization: `Bearer ${data!.session_token}` },
  });
  if (meRes.data && meRes.data.memberships.length > 0) {
    redirect("/contracts");
  }
  redirect("/onboarding/organization");
}
```

- [ ] **Step 5: Replace apps/web/src/app/page.tsx**

```tsx
import { redirect } from "next/navigation";
import { getMe } from "@/lib/server-session";

export const dynamic = "force-dynamic";

export default async function Home() {
  const me = await getMe();
  if (!me) redirect("/signin");
  if (me.memberships.length === 0) redirect("/onboarding/organization");
  redirect("/contracts");
}
```

- [ ] **Step 6: Run frontend typecheck**

```bash
cd apps/web && pnpm typecheck
```

Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add apps/web/src/app/signin apps/web/src/app/auth apps/web/src/app/page.tsx apps/web/src/lib/server-session.ts
git commit -m "feat(web): magic-link signin flow + verify -> session cookie"
```

---

## Task 19: Frontend — onboarding + contracts list + layout

**Files:**
- Create: `apps/web/src/app/onboarding/organization/page.tsx`
- Create: `apps/web/src/app/contracts/page.tsx`
- Modify: `apps/web/src/app/layout.tsx`

- [ ] **Step 1: Create apps/web/src/app/onboarding/organization/page.tsx**

```tsx
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { client } from "@/api/client";

export default function OnboardingOrganization() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-6 p-8">
      <h1 className="text-2xl font-semibold">Create your organization</h1>
      <form
        className="flex w-80 flex-col gap-3"
        onSubmit={async (e) => {
          e.preventDefault();
          setSubmitting(true);
          setErr(null);
          const { error } = await client.POST("/organizations", {
            body: { name },
          });
          setSubmitting(false);
          if (error) {
            setErr(String(error));
            return;
          }
          router.push("/contracts");
        }}
      >
        <input
          type="text"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="rounded-md border border-neutral-300 px-3 py-2"
          placeholder="Acme, Inc."
        />
        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-neutral-900 px-3 py-2 text-white disabled:opacity-50"
        >
          {submitting ? "Creating..." : "Continue"}
        </button>
        {err && <p className="text-sm text-red-600">{err}</p>}
      </form>
    </main>
  );
}
```

- [ ] **Step 2: Create apps/web/src/app/contracts/page.tsx**

```tsx
import { redirect } from "next/navigation";
import { getMe } from "@/lib/server-session";

export const dynamic = "force-dynamic";

export default async function Contracts() {
  const me = await getMe();
  if (!me) redirect("/signin");
  if (me.memberships.length === 0) redirect("/onboarding/organization");

  return (
    <main className="flex min-h-screen flex-col gap-6 p-8">
      <header className="flex items-center justify-between border-b border-neutral-200 pb-3">
        <div>
          <h1 className="text-xl font-semibold">{me.current_organization_name ?? me.memberships[0].organization_name}</h1>
          <p className="text-xs text-neutral-500">{me.user_email}</p>
        </div>
        <form action="/auth/logout" method="post">
          <button className="text-sm text-neutral-500 underline">Sign out</button>
        </form>
      </header>

      <section>
        <h2 className="text-lg font-medium">Contracts</h2>
        <p className="mt-2 text-sm text-neutral-500">
          No contracts yet. Phase 1 story 2 (intake form) lights this up.
        </p>
      </section>
    </main>
  );
}
```

- [ ] **Step 3: Replace apps/web/src/app/layout.tsx**

```tsx
import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Pactline",
  description: "Contract lifecycle platform built on durable workflows.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="antialiased">{children}</body>
    </html>
  );
}
```

(Layout stays minimal; per-page header lives in `/contracts/page.tsx` for now.)

- [ ] **Step 4: Frontend typecheck + dev verify**

```bash
cd apps/web && pnpm typecheck
```

Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add apps/web/src/app/onboarding apps/web/src/app/contracts apps/web/src/app/layout.tsx
git commit -m "feat(web): onboarding/organization + contracts list with header"
```

---

## Task 20: End-to-end smoke + audit chain SQL verification

**Files:** none new. Uses live stack.

- [ ] **Step 1: Bring up the full stack**

```bash
make up      # docker compose
make dev &   # api + worker + sidecars + web in honcho
```

Wait ~5 seconds for all processes to log "ready."

- [ ] **Step 2: Walk the flow manually**

1. Open http://localhost:3000 → redirected to `/signin`
2. Enter `demo@example.com` → submit
3. Open http://localhost:8025 → click magic link in the latest message
4. Land on `/onboarding/organization` → enter "Acme" → submit
5. Land on `/contracts` (empty list, header shows "Acme" + "demo@example.com")

- [ ] **Step 3: Verify audit chain via SQL**

```bash
PGPASSWORD=pactline psql -h localhost -p 5433 -U pactline -d pactline -c "
  SELECT action, organization_id IS NOT NULL AS scoped, encode(prev_hash, 'hex') AS prev, encode(event_hash, 'hex') AS hash
  FROM audit_events
  ORDER BY created_at;
"
```

Expected (one chain across actions in order — `prev_hash` of row N = `event_hash` of row N-1 within same chain):

```
       action          | scoped |  prev   |  hash
-----------------------+--------+---------+------
 user.created          | f      | (null)  | abc...
 magic_link.requested  | f      | abc...  | def...
 magic_link.consumed   | f      | def...  | 0ab...
 session.created       | f      | 0ab...  | 1cd...
 organization.created  | t      | (null)  | 2ef...   <- new chain (non-null org)
 membership.created    | t      | 2ef...  | 3fa...
```

- [ ] **Step 4: Verify chain integrity in SQL**

```bash
PGPASSWORD=pactline psql -h localhost -p 5433 -U pactline -d pactline -c "
WITH chain AS (
  SELECT id, organization_id, prev_hash, event_hash, created_at,
         lag(event_hash) OVER (PARTITION BY organization_id ORDER BY created_at) AS expected_prev
  FROM audit_events
)
SELECT count(*) AS broken FROM chain
WHERE prev_hash IS DISTINCT FROM expected_prev;
"
```

Expected: `broken = 0`.

- [ ] **Step 5: Stop services**

```bash
kill %1   # the honcho running make dev
make down # docker compose down
```

- [ ] **Step 6: Final commit**

```bash
git commit --allow-empty -m "chore(phase-1-story-1): end-to-end smoke verified, audit chain integrity confirmed"
```

---

## Self-review

Walked the spec section by section against the tasks:

- ✅ User flow: Tasks 11 (auth service), 13 (handlers), 18 (signin web), 19 (onboarding + contracts)
- ✅ Domain model (6 tables): Task 3
- ✅ Audit chain w/ trigger: Task 3 (schema), Task 4 (writer + chain test), Task 20 (e2e SQL verify)
- ✅ API surface: Tasks 13, 14, 15, 16 (router + spec)
- ✅ Auth middleware: Task 12
- ✅ Multi-tenancy: Task 6 (helper), Task 17 (isolation integration test)
- ✅ Frontend pages: Tasks 18, 19
- ✅ Substrate copy-in (audit chain): Task 3 (schema), Task 4 (writer)
- ✅ OpenAPI + drift: Task 16
- ✅ Acceptance criteria: covered (verification by Tasks 17 and 20)

No placeholders found. Type/method names checked for consistency between tasks (e.g., `database.NewUserRepo`, `handlers.AuthHandlers{Svc: …}`, `auth.NewService(...)` all consistent across tasks).

**One known imperfection** — Task 4's audit tests use a fresh org UUID per test to avoid bleed; they don't clean up rows between tests. The audit_events chain therefore grows across runs of the test database. That's fine for dev (we don't reset DB between tests within a session) but means running the suite multiple times causes the chain to keep extending. Acceptable: the acceptance criterion is "valid hash chain," not "exactly N rows."

---

## Plan complete

Plan saved to `docs/superpowers/plans/2026-05-05-phase1-story1-signup-org-contracts.md`.

Two execution options:

1. **Subagent-Driven (recommended)** — Dispatch a fresh subagent per task, review between tasks, fast iteration. Best when you want strong checkpoint review and parallel-friendly chunks.

2. **Inline Execution** — Execute tasks in this session using executing-plans, batch with checkpoints. Best when you want to watch the whole thing happen with tight feedback loops.

Which approach?
