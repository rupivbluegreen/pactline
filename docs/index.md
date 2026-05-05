---
layout: default
title: Home
---

# Pactline

> Open-source contract lifecycle platform built on durable workflows.
> Every contract is a versioned, observable, replay-able state machine —
> not a row in a database with hopes attached.

## Current state

**Phase 0 — Foundation** ✓ shipped 2026-05-05 ([tag `phase-0`](https://github.com/rupivbluegreen/pactline/releases/tag/phase-0))

The repository scaffold runs end-to-end with no features.
Polyglot stack: Go core (`cmd/api`, `cmd/worker`, `internal/*`),
Python sidecars (`apps/{ai,document}-sidecar`), Next.js 15 frontend
(`apps/web`). Hello-world API → OpenAPI → TypeScript types pipeline
is live. CI runs lint, typecheck, test, and a generated-types drift
check on every push.

**Phase 1 — v1 MVP** in progress

Pre-sign pipeline (intake → approval → signature) for one contract
type per organization, with read-only AI extraction. Story 1 (signup
→ organization → empty contracts list) is currently being implemented.

[See the full phased plan →](roadmap.html)

## Reading order

| Doc | What it covers |
|---|---|
| [Product](product.html) | Vision, target user, principles, deployment models |
| [Architecture](architecture.html) | Domain model, services, data flows, AI design |
| [Roadmap](roadmap.html) | Phase plan with in-scope / out-of-scope per phase |
| [ADR 0001](adr/0001-record-architecture-decisions.html) | We use ADRs and this is how |
| [ADR 0002](adr/0002-language-stack.html) | Python-everywhere — *superseded* |
| [ADR 0003](adr/0003-polyglot-stack.html) | Polyglot stack: Go core + Python AI/doc sidecars + TypeScript frontend |

## Wedge

Pactline's pitch is **workflow-native CLM**: contracts as state
machines. Every state change runs in a versioned, durable workflow
(Temporal). There are no "save and hope" code paths. The target
audience is legal-ops at tech-mature mid-market companies (200–2000
people, eng-led culture) who already own a CLM tool and have
outgrown its workflow primitives.

## Source and license

[github.com/rupivbluegreen/pactline](https://github.com/rupivbluegreen/pactline)
— AGPL-3.0.
