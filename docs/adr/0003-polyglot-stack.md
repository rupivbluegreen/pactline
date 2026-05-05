# ADR 0003 — Polyglot stack: Go core + Python AI/document sidecars + TypeScript frontend

- **Status:** Accepted
- **Date:** 2026-05-05
- **Deciders:** Founding team
- **Supersedes:** [ADR 0002](0002-language-stack.md)

## Context

ADR 0002 picked Python everywhere except the frontend, on the strength of
Python's AI-orchestration and document-parsing ecosystems. That reasoning
still holds for those two domains. What ADR 0002 did not weigh: pactline
is not the only product in the portfolio. Statebound — a sister
authorization-governance product — is in Go, ships v1.0, and contains
substrate that pactline genuinely needs:

- Hash-chained audit log (`audit_event_hash`, `audit verify` chain walk)
- Ed25519-signed plan/state bundles
- OIDC bearer-auth middleware
- OPA + Rego runtime engine (used in statebound for governance rules;
  not the same as pactline's per-request authz, which uses Cedar in
  Phase 6 — see ADR 0003-A "Cedar for in-product authz, OPA-via-
  statebound for meta-policy" if/when written)
- OpenTelemetry conventions (span / attribute schema)
- Helm chart shape with NetworkPolicy, runAsNonRoot, secret-backed
  signing key
- Distroless Docker pattern
- Goose migrations
- RBAC role primitives (5 roles, bootstrap-once gate)

Two consequences of keeping pactline in Python (per ADR 0002):

1. Substrate cannot be shared as Go modules; patterns have to be
   reimplemented in Python — duplicating work in the most security-
   sensitive modules.
2. Hiring and operational consistency across the portfolio is forfeit.
   The team needs Go for statebound regardless, so making pactline
   Python adds Python to the skills ledger without a corresponding
   language reduction elsewhere.

Both costs are now dominant. The Python case for pactline is therefore
narrower than ADR 0002 framed it: Python is required for AI
orchestration (LangGraph, LiteLLM, Pydantic-typed structured output)
and for document parsing (PyMuPDF, Docling, layout-aware PDF/DOCX). It
is **not** required for workflow, authz, audit, storage, integrations,
or the API surface — Go has first-class support for all of those
(Temporal Go SDK is the reference SDK; pgx for Postgres; Cedar Go
bindings for ABAC; hand-rolled audit chain following statebound's
pattern; standard chi router or net/http for the API).

## Decision

Pactline adopts a polyglot stack:

- **Go 1.25** for the core. Single Go module rooted at
  `github.com/rupivbluegreen/pactline`. Layout follows statebound's
  convention: `cmd/<binary>/main.go` + `internal/<area>/`. Binaries:
  `cmd/api` (HTTP API), `cmd/worker` (Temporal worker).
- **Python 3.12+** for two sidecar services that talk to the Go core
  via gRPC:
  - `apps/ai-sidecar/` — wraps LangGraph + LiteLLM + Pydantic.
    Returns Citation-bearing structured output to the Go core.
  - `apps/document-sidecar/` — wraps PyMuPDF + Docling + python-docx +
    LibreOffice headless. Parses uploaded documents, renders DOCX
    templates, converts DOCX→PDF.
- **TypeScript / Next.js 15 / React 19** for the frontend
  (`apps/web/`), unchanged from ADR 0002.

Three languages in the codebase. Adding a fourth requires an ADR.

The Go ↔ Python boundary is **gRPC**, with proto schemas defined in
`proto/`. Sidecar services are stateless, run as separate processes
(separate containers in production, separate honcho processes in dev),
and are replaceable behind their gRPC contracts. The Go core must
never import a Python module; the Python sidecars must never call back
into the Go core (one-way request/response only).

Substrate shared with statebound (audit chain, signing, OIDC, OPA
where applicable, Goose migrations, Helm chart shape, OpenTelemetry
conventions) is **vendor-copied** into pactline's `internal/` for
now. When the boundary stabilizes — likely mid-Phase-1 — those
packages get extracted into shared repositories
(`github.com/rupivbluegreen/audit-chain`, `…/signing`, etc.) imported
by both products. Premature extraction risks pinning module APIs
before pactline has exercised them.

ABAC engine for Phase 6 remains **Cedar**, not OPA. Cedar fits
pactline's per-request app-authz shape better than Rego, gives
analyzability (which is legible to legal-ops buyers as an audit-grade
claim), and runs in-process at microsecond latency. Statebound's OPA
stays in statebound's lane (governance over change-sets, a different
problem).

## Consequences

- **Substrate reuse with statebound becomes feasible.** Audit chain,
  signing, OIDC, telemetry, Helm shape, distroless Docker — all
  shared as Go code (vendor-copied now, modules later).
- **Python is contained to where it earns its weight.** AI
  orchestration and document parsing get Python's ecosystem;
  everything else gets Go's deployment story (single static binary,
  distroless image, strong concurrency, tooling parity with
  statebound).
- **gRPC boundary is the new architectural contract.** Adding an AI
  capability or a document operation means defining a proto method,
  not adding a Python module. This is a healthy boundary that allows
  replacing either sidecar (e.g., swapping LiteLLM for a hosted
  vendor) without touching the Go core.
- **Hiring becomes Go + Python + TS.** Same as if pactline stayed
  Python (Python + TS) and statebound stayed Go (Go + TS) — no new
  languages added, just a redistribution.
- **Operational complexity rises slightly.** Two extra services to
  deploy. Mitigated by both being stateless and by sharing the same
  observability conventions as the Go core.
- **AI and document calls run out-of-process.** A ~5-50 ms gRPC hop
  replaces an in-process function call. Acceptable for AI extraction
  (LLM latency dominates); fine for document parsing (parses are
  seconds-to-minutes).

## Alternatives considered

**Stay Python (ADR 0002 unchanged).** Reimplement statebound's
substrate in Python. No code reuse with statebound. Two
implementations of audit chain, signing, OIDC wiring, evidence
export — duplicating the most security-sensitive modules.

**Full Go rewrite, including AI and document parsing.** Maximum
substrate reuse but loses the Python AI ecosystem (LangGraph,
LiteLLM, Pydantic structured output). Document parsing in Go is a
fight — UniPDF is commercial/AGPL, pdfium-Go bindings are CGO,
DOCX layout extraction is materially harder. Net effect: pactline
ships a worse extraction story to gain stack consistency.

**Polyglot via in-process FFI / WebAssembly.** Python or Rust
compiled to wasm, called from Go. Tooling not mature enough;
debugging story is poor. Reconsider in 2027+.

**OPA instead of Cedar for Phase 6 ABAC.** Strengthens building-
blocks story (one engine across portfolio) but weakens technical
fit (Cedar is purpose-built for app authz; OPA is general-purpose).
Decision documented in this ADR's Decision section: Cedar stays.

## References

- ADR 0001 (record architecture decisions)
- ADR 0002 (Python-everywhere — superseded by this ADR)
- statebound v1.0 README — substrate this ADR borrows from
- `docs/architecture.md` — service topology this ADR produces
