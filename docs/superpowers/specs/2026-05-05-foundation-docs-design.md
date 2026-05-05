# Phase 0 Foundation Docs — Design Spec

- **Status:** Approved (this session, 2026-05-05)
- **Topic:** Initial documentation set for the pactline repository

## Context

Pactline is in Phase 0 (Foundation). The repository was just initialized
and is otherwise empty. CLAUDE.md (operational handoff) is in place but
references several authoritative docs that don't yet exist:

- `docs/product.md` — vision, users, principles, deployment models
- `docs/architecture.md` — domain model, services, data flow, AI design
- `docs/roadmap.md` — phased build plan and current state
- `docs/adr/` — ADR template and the first two ADRs (0001 meta,
  0002 language stack, both referenced by CLAUDE.md)

This spec records the design we agreed for those docs. After this spec
is written, the docs themselves are written in the same session.

## Design decisions (locked this session)

- **Wedge:** workflow-native, contracts as state machines.
- **Target user:** legal ops at tech-mature mid-market companies
  (200–2000 person, eng-led culture, compliance-real).
- **v1 lifecycle bullseye:** pre-sign pipeline (intake → approval →
  signature).
- **v1 cut:** workflow-first, AI assists (read-only AI extraction with
  citations + playbook flags). AI redlines deferred to Phase 3.

## Doc set

```
docs/
├── product.md
├── architecture.md
├── roadmap.md
└── adr/
    ├── template.md
    ├── 0001-record-architecture-decisions.md
    └── 0002-language-stack.md
```

## product.md outline

1. Vision (1 paragraph + wedge sentence)
2. Target user (1 paragraph + 5 bullets on the buyer's reality)
3. Principles (6 short opinions)
4. Deployment models (3 short paragraphs)
5. Non-goals (6 bullets)

The six principles:

1. Workflow is the product
2. Opinionated, not configurable
3. AI assists, never decides
4. Cite or it didn't happen
5. Multi-tenant by construction
6. Open core, not open bait

## architecture.md outline

1. Domain model — Phase 1 entities (Organization, User, ContractType,
   Contract, ContractDocument, ApprovalRoute, ApprovalRequest,
   PlaybookRule, ExtractedField, Citation, AuditEvent) with an ASCII
   ER sketch.
2. Service topology — web / api / worker / Postgres / Redis / S3 /
   Temporal / AI provider, with ports and ASCII diagram.
3. Data flows — intake, approval, signature (sequence-style bullets).
4. AI design — provider abstraction, citation primitives, prompt
   versioning, untrusted-input handling, deterministic-test rule.
5. Security and multi-tenancy — TenantScopedSession, RBAC at v1, audit
   schema, secrets boundary.
6. Deferred sections — bullet pointers to the relevant Phase
   (templating, reporting, AI redlines, post-sign, counterparty,
   Cedar).

## roadmap.md outline

1. Phasing principle (each phase ships a dogfoodable artifact, no
   flagged-off features).
2. Phase 0 — Foundation (current state, in flight).
3. Phase 1 — v1 MVP (detailed: in-scope, out-of-scope, success
   criteria, key risks).
4. Phases 2–6 (one short paragraph each).
5. Permanently out of roadmap (4 items: native e-sign, visual
   workflow builder, ERP/CRM replacement, white-label theming).

## ADR plan

- `template.md` — Nygard format (Title, Status, Context, Decision,
  Consequences, Alternatives, References).
- `0001-record-architecture-decisions.md` — meta. We use ADRs; here
  are the situations CLAUDE.md says require one; here is the workflow
  for proposing / accepting / superseding.
- `0002-language-stack.md` — Status: Accepted. Decision: Python 3.12+
  everywhere except the Next.js / TypeScript frontend.

## Verification

- All six files exist at the specified paths.
- `docs/product.md` references `docs/roadmap.md` for phase-scope
  questions.
- `docs/architecture.md` references `docs/roadmap.md` and CLAUDE.md.
- `docs/adr/template.md` is structurally complete.
- ADR 0001 and 0002 follow `template.md`.
- CLAUDE.md is unchanged (it already references the doc set; the docs
  now exist).

## Out of scope for this spec

- Code scaffolding (monorepo structure, packages, apps, CI). Those
  land in Phase 0's next step, after these docs are written.
- ADRs beyond 0001 and 0002. Future ADRs (Temporal-from-day-0,
  multi-tenancy model, RBAC schema) are written when the decisions are
  actually being made, per ADR 0001.

## References

- `CLAUDE.md` — operational handoff
- Brainstorming dialogue 2026-05-05 (this session) — wedge, target,
  v1 cut decisions
