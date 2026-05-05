# ADR 0001 — Record architecture decisions

- **Status:** Accepted
- **Date:** 2026-05-05
- **Deciders:** Founding team

## Context

Pactline is open source under AGPL-3.0 and intended to outlive its
original authors. Over time, contributors, customers, and self-hosters
will need to understand why the codebase looks the way it does — not
just what it currently is.

Code answers "what." Tests answer "does it work." Neither answers "why
this and not the obvious alternative."

Architecture Decision Records (ADRs) are a low-overhead format for
recording the *why* behind non-trivial decisions, alongside the
context they were made in. They follow Michael Nygard's pattern:
short, focused, immutable once accepted.

## Decision

Pactline records architecture decisions as ADRs in `docs/adr/`. Each
ADR is a markdown file numbered sequentially, named
`NNNN-short-slug.md`, and follows `docs/adr/template.md`.

ADRs are required for:

- Stack changes (workflow engine, database, ORM, frontend framework).
- New external dependencies on the critical path (dependencies the
  core workflow can't function without).
- Security model changes (auth, authz, multi-tenancy boundary, audit
  schema).
- Schema changes that affect more than one package.
- AI provider behavior changes (default model, prompt versioning
  rules, citation schema).
- License posture changes (AGPL exception scope, commercial-license
  module additions).

ADRs are not required for routine code changes, refactors, bug fixes,
or single-package changes that don't cross the criteria above.

ADR status transitions: an ADR starts as Proposed (PR open), becomes
Accepted (PR merged), and may later be Superseded (a newer ADR
replaces it) or Deprecated (decision no longer applies; no
replacement). **Accepted ADRs are not edited.** If a decision changes,
write a new ADR that supersedes the prior one.

## Consequences

- Reviewers can demand an ADR for changes that meet the criteria
  above. This is the load-bearing enforcement mechanism — without
  review-time enforcement the practice atrophies.
- New contributors and customers can understand the *why* of any major
  decision without archaeology through git history or chat logs.
- The ADR set is part of the product's open-source posture: forking
  pactline means inheriting its decision context, not just its code.

## Alternatives considered

**Comments in code.** Insufficient — comments are tied to specific
files and rot when code moves. Decisions span files.

**Wiki / Notion.** Doesn't version with the code. A reader checking
out an old commit sees a different codebase than the latest wiki.

**Commit messages only.** Inadequate for cross-cutting decisions and
unsearchable as a body of work.

## References

- Michael Nygard, "Documenting Architecture Decisions"
  (cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
- `docs/adr/template.md`
