# Pactline — Roadmap

This document is the authoritative source for what's in scope per phase.
CLAUDE.md and `docs/architecture.md` reference this file when scope is
ambiguous. Phase boundaries are deliberately conservative — better to
ship a smaller phase and start the next than to widen a phase and slip.

## Phasing principle

Each phase ends with a shippable, dogfoodable artifact. There is no
phase that lands behind a feature flag. Code that anticipates Phase
N+1 in Phase N is rejected on review. ADRs accompany decisions that
span phases (e.g., the v1 storage layer must be Phase-6-BYOC-compatible
even though BYOC ships in Phase 6).

## Phase 0 — Foundation

**Status:** in flight (May 2026).

**Deliverable:** repository scaffold runnable end-to-end with no
features.

- Monorepo skeleton: `apps/web`, `apps/api`, `apps/worker`,
  `packages/pactline_*`, `infra/`, `docs/`.
- Tooling: `uv` for Python, `pnpm` for frontend, `Makefile` for
  orchestration, `ruff` + `pyright` + `pytest` for Python, `tsc` +
  `vitest` for web.
- Local dev: `docker compose up -d` brings up Postgres 16, Redis 7,
  MinIO, Temporal, Mailpit. `make dev` runs api + worker + web in
  parallel via honcho.
- CI: GitHub Actions running `make lint typecheck test` plus container
  build, Trivy scan, syft SBOM, Cosign signing on tagged releases.
- Hello-world endpoint demonstrating the FastAPI → openapi-typescript
  → web type pipeline.
- ADRs: 0001 (record decisions), 0002 (language stack).

**Out of Phase 0:** any domain entity, any UI beyond hello-world, any
authentication, any AI calls.

## Phase 1 — v1 MVP

**Theme:** pre-sign pipeline for one contract type per organization,
with read-only AI extraction.

**In scope:**

- One canonical pipeline: intake → review → approve → sign.
- Single contract type per organization (configured at setup).
- Document upload (PDF, DOCX), parsing (PyMuPDF, Docling).
- Read-only AI extraction with Citations: parties, effective date,
  term, value, governing law.
- Playbook flags: declarative rules surfaced as warnings in the UI.
  Rules do not block workflow.
- ApprovalRoute model: type + threshold → ordered approver chain.
- ApprovalRequest: in-app review, approve / reject / request-changes.
- DocuSign integration for execution.
- Repository view: list, search by metadata, view document + extracted
  fields with citation links.
- RBAC: `owner`, `admin`, `reviewer`, `approver`, `viewer`.
- Self-host runnable with the same `docker compose` from Phase 0 plus
  migrations.

**Out of scope (deferred):** multi-contract-type, templating + clause
library, reporting, AI redlines, post-sign obligations, counterparty
portal, SSO, ABAC.

**Success criteria:**

- Median pre-sign cycle time under 5 business days on a dogfooded
  contract set of 50 contracts.
- Every audit event reconstructable end-to-end from a contract URL.
- Zero AI outputs without citations.
- Self-host smoke test: a developer with Docker + uv + pnpm can land a
  runnable instance in under 30 minutes.

**Key risks:**

- Workflow versioning discipline (every running execution must
  reference its workflow version; `patched()` for in-flight changes).
- AI extraction accuracy on real-world MSAs — managed via the eval
  set, not by tuning prompts in production.
- Signature-provider sandbox / production parity — covered by
  integration tests against the sandbox in CI.

## Phase 2 — Multi-type, templating, reporting

Multiple contract types per organization. `pactline_template` package
goes live: Jinja2 + python-docx for DOCX rendering, LibreOffice
headless for DOCX→PDF. Versioned clause library. Reporting on cycle
time, bottlenecks, contract-value distribution.

## Phase 3 — AI redlines

AI suggests inline redlines with Citations. Humans accept / reject per
suggestion. Playbook rule engine matures: rules can target clauses,
not just metadata. AI never auto-applies redlines (principle 3).

## Phase 4 — Post-sign lifecycle

Obligations extracted at execution time (renewal date, notice period,
SLAs, payment milestones). Alerts via in-app, email, calendar
integration. Renewal-decision workflow.

## Phase 5 — Counterparty collaboration

External counterparty portal. Magic-link redline rounds. Side-by-side
diff. Counterparty AI exposure remains read-only and citation-bound.

## Phase 6 — Enterprise

SAML / OIDC enterprise SSO. Cedar ABAC replaces ad-hoc role checks.
BYOC private-cloud distribution (Helm + customer-cloud installer).
Audit log export to customer SIEM. Vault / cloud-KMS secrets backend.

## Permanently out of roadmap

- **Native e-signature.** Pactline never replaces a signature provider.
  We integrate.
- **Visual workflow builder.** Violates principle 2 (`docs/product.md`).
- **CRM / ERP replacement features.** We connect; we don't subsume.
- **White-label theming.** Considered only as a narrow commercial-
  license feature, not roadmap.
