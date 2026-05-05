# Pactline

Open-source contract lifecycle platform built on durable workflows.

**Docs:** [pactline GitHub Pages](https://rupivbluegreen.github.io/pactline) (deployed from `docs/` on push to `main`)

- **Product** — `docs/product.md`
- **Architecture** — `docs/architecture.md`
- **Roadmap** — `docs/roadmap.md`
- **ADRs** — `docs/adr/`

## Phase status

| Phase | Status | Tag |
|---|---|---|
| 0 — Foundation | ✓ shipped 2026-05-05 | `phase-0` |
| 1 — v1 MVP (pre-sign pipeline + AI extraction) | in progress | — |
| 2 — Multi-type, templating, reporting | planned | — |
| 3 — AI redlines | planned | — |
| 4 — Post-sign lifecycle | planned | — |
| 5 — Counterparty collaboration | planned | — |
| 6 — Enterprise (SSO, ABAC, BYOC) | planned | — |

The Pages site reflects this table on every push to `main`. Each phase
boundary is tagged in git for navigation.

## Stack

Polyglot by design (see ADR 0003):

- **Go 1.22+** core (`cmd/api`, `cmd/worker`, `internal/*`)
- **Python 3.12** sidecars for AI (`apps/ai-sidecar`) and document
  parsing / DOCX rendering (`apps/document-sidecar`), reached via gRPC
- **TypeScript / Next.js 15** frontend (`apps/web`)

## Prerequisites

- Go 1.22+ (1.25 is the production target — set `GOTOOLCHAIN=local` if
  using 1.22 locally and the toolchain server isn't reachable)
- Python 3.12 + [uv](https://docs.astral.sh/uv/)
- Node 20+ + [pnpm](https://pnpm.io/) 9
- Docker (for Postgres, Temporal, Redis, MinIO, Mailpit)

## Quickstart

```bash
make setup     # go mod download + sidecar uv sync + pnpm install + docker compose up
make dev       # api + worker + ai-sidecar + document-sidecar + web in parallel
```

Then visit:

- Web: http://localhost:3000
- API: http://localhost:8000/healthz
- Temporal UI: http://localhost:8233
- MinIO console: http://localhost:9001 (user: `pactline`, password: `pactlinepactline`)
- Mailpit: http://localhost:8025

## Default ports

| Service | Port |
|---|---|
| web (Next.js) | 3000 |
| api (Go) | 8000 |
| ai-sidecar (gRPC) | 50051 |
| document-sidecar (gRPC) | 50052 |
| Temporal | 7233 (gRPC), 8233 (UI) |
| Postgres | 5433 (5432 inside the compose network) |
| Redis | 6379 |
| MinIO | 9000 (S3), 9001 (console) |
| Mailpit | 1025 (SMTP), 8025 (UI) |

## Verification before commit

```bash
make lint
make typecheck
make test
```

## License

AGPL-3.0. See `LICENSE`.
