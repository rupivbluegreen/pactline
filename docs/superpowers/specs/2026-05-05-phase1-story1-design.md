# Phase 1 Story 1 — Signup → Organization → Empty Contracts List

- **Status:** Approved (this session, 2026-05-05)
- **Phase:** 1 (v1 MVP)
- **Story:** First walking-skeleton vertical slice

## Context

Phase 0 shipped a runnable polyglot scaffold (commit 8b5487d) but no
domain entities, no auth, no migrations, no audit chain. Phase 1's
roadmap deliverable is "pre-sign pipeline for one contract type per
organization" — too large for a single design.

This story is the smallest concrete user-visible slice: a new user
signs in, creates an organization, and lands on an empty contracts
list. It exercises the multi-tenant chassis end-to-end (auth →
session → membership → tenant-scoped query → empty contract list →
hash-chained audit events) without touching AI, document parsing,
workflow, or external integrations.

Subsequent Phase 1 stories build on this chassis (intake form,
document upload, AI extraction, approval routing, signature, etc.).

## Decisions (locked this session)

- **Auth issuer:** magic-link (passwordless) via Mailpit in dev. The
  session token doubles as the bearer the auth middleware checks.
- **Session model:** server-side session, sha256(token) stored in
  `sessions.token_hash`. Cookie + `Authorization: Bearer` both
  accepted.
- **Audit:** first use of the hash-chain pattern. Per-org chain;
  pre-org events ride a system chain with `organization_id = NULL`.
- **Migrations:** Goose, embedded.
- **Multi-tenancy:** `database.TenantScoped(ctx)` helper introduced
  this story.

## User flow

1. `/signin` — email field → submit
2. POST `/auth/magic-link/request` — server creates / finds user,
   inserts `magic_links` row, emails the link (Mailpit dev)
3. Click link → `/auth/verify?token=…` → POST
   `/auth/magic-link/verify` → server creates session, sets
   `pactline_session` cookie
4. First-time user: redirect to `/onboarding/organization` →
   POST `/organizations` → owner membership created
5. Returning or post-onboarding user: redirect to `/contracts`
   (empty list, multi-tenant scoped)

## Domain model (this story)

```
users               (id uuid, email citext unique, created_at,
                     last_login_at nullable)

organizations       (id uuid, slug citext unique, name, created_at)

memberships         (id uuid, user_id fk, organization_id fk, role
                     enum('owner','admin','reviewer','approver',
                          'viewer'), created_at,
                     unique(user_id, organization_id))

sessions            (id uuid, user_id fk, token_hash bytea unique,
                     expires_at, created_at, revoked_at nullable)

magic_links         (id uuid, email citext, token_hash bytea unique,
                     expires_at, used_at nullable, created_at)

audit_events        (id uuid, organization_id fk nullable,
                     actor_user_id fk nullable, action text,
                     entity_type text, entity_id uuid nullable,
                     before jsonb, after jsonb,
                     prev_hash bytea, event_hash bytea,
                     created_at)
```

## Audit chain

- Postgres function `audit_event_hash(prev_hash, event_data)`
  returns `sha256(coalesce(prev_hash, '') || canonical_json(event_data))`.
- BEFORE INSERT trigger on `audit_events`: looks up the previous
  event's hash for the same chain key (organization_id), computes
  `event_hash`, sets it on the row.
- App role gets `INSERT` only — no `UPDATE` / `DELETE` on
  `audit_events`.
- Events written this story:
  - `user.created`
  - `magic_link.requested`
  - `magic_link.consumed` (login)
  - `session.created`, `session.revoked`
  - `organization.created`
  - `membership.created`

## API surface (delta on schemas/openapi.yaml)

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/auth/magic-link/request` | none | `{ email }` → 202 |
| POST | `/auth/magic-link/verify` | none | `{ token }` → 200 + Set-Cookie session |
| POST | `/auth/logout` | session | revoke session, clear cookie → 204 |
| GET  | `/me` | session | `{ user, memberships, current_organization }` |
| POST | `/organizations` | session | `{ name, slug }` → 201, creates org + owner membership |
| GET  | `/contracts` | session+org | `{ contracts: [] }` |

All routes documented in `schemas/openapi.yaml`. CI fails if generated
TS types drift.

## Auth middleware

`internal/api/middleware/auth.go`:

1. Read `Cookie: pactline_session` or `Authorization: Bearer <tok>`.
2. `sha256(tok)` → look up `sessions.token_hash`. Verify
   `revoked_at IS NULL` and `expires_at > now()`.
3. Set `ctx.user_id` and (when present) `ctx.organization_id` from
   the user's active membership (single-org case for v1).
4. `/auth/*` paths bypass the middleware. `GET /me` requires
   `ctx.user_id`. `POST /organizations` requires `ctx.user_id` only
   (no org yet). `GET /contracts` requires both.

## Multi-tenancy

`internal/database/tenant.go`:

```go
func TenantScoped(ctx context.Context) (orgID uuid.UUID, ok bool)
```

Returns the `organization_id` set by the auth middleware. Repository
methods that touch tenant-scoped tables call this and inject the
filter into every query. There is no admin bypass.

## Frontend

`apps/web/src/app/`:

- `/signin/page.tsx` — email form
- `/signin/sent/page.tsx` — "check your email"
- `/auth/verify/page.tsx` — server component, takes `token` query
  param, calls verify, redirects
- `/onboarding/organization/page.tsx` — org name form
- `/contracts/page.tsx` — empty list
- `layout.tsx` — header (user email + org name + logout)

All data via openapi-fetch from `apps/web/src/api/client.ts`. Types
regenerated by `make generate-types`.

## Substrate copy-in this story

Only what this story exercises:

- `internal/audit/` — chain schema (in migration), `Write(ctx,
  Event)` writer function, event-type constants. Lifted from the
  statebound pattern (per ADR 0003). Implementation done from the
  statebound README's described primitives — actual file copy from
  the statebound repo can be done in a follow-up if cleanly
  separable.
- `internal/database/` — pgxpool wiring, `TenantScoped(ctx)`,
  embedded Goose migration runner.
- Auth middleware shape — bearer-acceptor mirrors the statebound
  middleware contract; the issuer is bespoke pactline.

Deferred (lift when first needed): Ed25519 signing, OPA wiring,
Cedar (Phase 6 only), OpenTelemetry beyond `slog`.

## Out of scope (deferred to later Phase 1 stories)

- Multi-org switcher UI (placeholder when only one org)
- Rate limiting on magic-link request (needs Redis wiring)
- Session refresh / rotation
- 2FA / TOTP
- CSRF tokens (added when first non-bearer form arrives)
- Production SMTP config (Mailpit dev only)
- `audit verify` CLI (write the chain now; verify CLI later)
- Substrate file-copy from statebound repo (re-implementing the
  pattern this round)
- Cedar / OPA / ABAC (Phase 6)
- Email verification beyond magic-link (the link IS the verification)

## Acceptance criteria

1. New user signs in via magic link, creates org, lands on empty
   `/contracts`.
2. Every state-changing action emits an `audit_events` row with a
   valid hash chain (computable + verifiable in a SQL query).
3. Second user creating their own org cannot see the first user's
   data via any authenticated API call (multi-tenant isolation
   test asserts this).
4. `make lint typecheck test` is clean.
5. `apps/web/src/api/types.ts` is regenerated from
   `schemas/openapi.yaml`; CI drift check passes.

## Estimated effort

3–5 developer-days. Roughly:

- Day 1: migration + audit chain + database wiring + tests
- Day 2: magic-link flow + sessions + auth middleware + tests
- Day 3: org + memberships + tenant-scoped query + tests
- Day 4: frontend wiring + e2e
- Day 5: polish, drift checks, demo

## Verification (end-to-end smoke)

1. `make up && make dev`
2. Open http://localhost:3000 → /signin → enter email → submit
3. Open http://localhost:8025 → click magic link in Mailpit
4. Land on /onboarding/organization → enter "Acme" → submit
5. Land on /contracts (empty)
6. `psql` → `SELECT action, organization_id, encode(event_hash,'hex')
   FROM audit_events ORDER BY created_at;` shows the chain
7. Sign in as a second user with a different org → can't see Acme

## References

- `CLAUDE.md` — operational handoff
- `docs/architecture.md` — domain model, security & multi-tenancy
- `docs/roadmap.md` — Phase 1 v1 MVP scope
- `docs/adr/0003-polyglot-stack.md` — substrate-from-statebound
  rationale
- statebound v1.0 README — hash-chain audit pattern, RBAC role
  primitives
