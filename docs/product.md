# Pactline — Product

## Vision

Pactline is a contract lifecycle platform built on durable workflows:
every contract is a versioned, observable, replay-able state machine,
not a row in a database with hopes attached.

The category problem pactline addresses: contract lifecycle tooling has
historically been email and spreadsheets dressed up as software, with
reliability that breaks the moment someone goes on vacation, an approval
is missed, or a counterparty redline arrives during a release window.
Pactline treats every state change as code — durable, idempotent,
auditable, replay-able — and gives the legal-ops engineer the same
operational primitives a backend engineer already has.

## Target user

The first ten customers are **legal ops leaders at tech-mature
mid-market companies** (200–2000 people, eng-led culture, cloud-native
ops, compliance-real). They already own a CLM tool and have outgrown
its workflow primitives. They want:

- **Durable, observable workflows.** They already use Temporal / SQS /
  Airflow elsewhere and expect the same of their CLM.
- **Audit-grade everything.** Every state change, every AI suggestion,
  every policy decision must be reconstructable months later.
- **Self-host or BYOC.** Sending unredacted MSAs to a vendor's
  multi-tenant cloud is a hard no.
- **Open code.** They want to read the engine, fork the edge cases,
  and not be held hostage by vendor roadmaps.
- **Python.** Their internal tooling team can read it, and Python is
  where contract AI, document parsing, and workflow ecosystems are
  materially better than the alternatives.

## Principles

These six opinions drive every later decision. Anything that violates
one needs an ADR.

1. **Workflow is the product.** Every state change runs in a versioned,
   durable workflow. There are no "save and hope" code paths.
2. **Opinionated, not configurable.** One canonical pipeline (intake →
   review → approve → sign). Pactline is not a visual workflow builder.
   Configuration knobs replace pipeline rewriting.
3. **AI assists, never decides.** AI is read-only at v1: it extracts,
   it flags, it cites. Humans drive every state change.
4. **Cite or it didn't happen.** Every AI output is bound to a document
   span, model identifier, prompt version, and timestamp. AI without a
   citation is not consumable.
5. **Multi-tenant by construction.** Every database query goes through
   `TenantScopedSession`. There is no admin bypass.
6. **Open core, not open bait.** AGPL covers everything actually useful.
   Commercial extensions are narrow and explicit (SSO, support, advanced
   analytics) and never load-bearing for the core workflow.

## Deployment models

**Self-hosted (AGPL-3.0).** The full product is open source under
AGPL-3.0. Run it anywhere — laptop, VM, Kubernetes. AGPL means
modifications exposed over a network must be shared back. This is the
deployment mode the codebase optimizes for first.

**Hosted SaaS.** Operated multi-tenant cloud for SMEs and mid-market
customers who don't want to operate the infrastructure. Same code as
self-host, plus operational hardening (managed upgrades, backups, SLA
support).

**Private cloud / BYOC.** The hosted product deployed inside the
customer's cloud (AWS / Azure / GCP) with managed updates. Phase 6+.
Same code, nothing forked.

## Non-goals

- **Not a workflow builder.** No drag-and-drop flow editor. Customers
  configure pipeline stages, not invent them.
- **Not a contract drafter.** Templating exists from Phase 2, but
  pactline is not a "describe what you want and get a contract"
  generator. AI suggests edits inside humans' redlines, not whole
  drafts.
- **Not an e-signature provider.** We integrate signature providers; we
  never replace them.
- **Not a CRM/ERP replacement.** We connect to those systems; we never
  subsume their data ownership.
- **Not procurement-first.** Buy-side workflows work via the same
  primitives, but legal ops is the audience the v1 product is designed
  for.
- **Not a "configure your CLM" platform.** The endless-config CLM
  dashboard is the experience pactline is escaping.

See `docs/roadmap.md` for what each phase ships and `docs/architecture.md`
for how the principles above translate into code.
