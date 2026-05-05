# Phase 1 Story 2 — Contract Intake + AI Metadata Extraction

- **Status:** Approved (this session, 2026-05-05)
- **Phase:** 1 (v1 MVP)
- **Story:** Second walking-skeleton vertical slice; lights up document
  storage, document sidecar, AI sidecar, and Temporal workflows.

## Context

Phase 1 Story 1 shipped the multi-tenant chassis (auth, sessions,
hash-chained audit, tenant-scoped queries, empty contracts list).
Story 2 lights up the rest of the polyglot stack: MinIO storage, the
Python document sidecar (PyMuPDF + Docling), the Python AI sidecar
(LangGraph + LiteLLM + Pydantic), and the first Temporal
ContractLifecycleWorkflow. Both proto contracts (`ai.proto`,
`document.proto`) get their first real RPCs.

After Story 2, an authenticated user can upload an NDA PDF/DOCX,
watch it parse, and see five Citation-bearing extracted fields on
the contract detail page. Approval routes and signature land in
Stories 3-4.

## Decisions (locked at story start)

- **One contract type per org for v1.** ContractType row seeded at org
  creation (NDA). Multi-type lands Phase 2.
- **Supported uploads:** PDF and DOCX. Other formats rejected at intake.
- **AI provider:** LiteLLM with Anthropic Claude Sonnet (per
  architecture.md). Configurable per-org, defaults to global setting.
- **Parser:** PyMuPDF for PDF, python-docx for DOCX. Docling reserved
  for Phase 2 layout-aware parsing.
- **Workflow:** ContractLifecycleWorkflow starts on upload, runs
  ParseDocumentActivity → ExtractMetadataActivity → EvaluatePlaybookActivity (no rules in this story; placeholder) → transitions Contract to `ready_for_review` and waits for an approval signal that never arrives in this story.
- **Citation model:** `(document_id, page_or_paragraph, span_start,
  span_end, model_id, prompt_version, timestamp)` per architecture.md.
- **Extracted fields (5):** `parties`, `effective_date`, `term`,
  `value`, `governing_law`. Each Citation-bearing.
- **Storage:** MinIO bucket `pactline-documents`, key path
  `{organization_id}/{contract_id}/{document_id}.{ext}`. Signed URLs
  for download.

## Domain model delta (this story)

Adds to the Story 1 schema (one Goose migration: `00003_contracts.sql`):

```
contract_types        (id uuid, organization_id, slug, name, created_at)

contracts             (id uuid, organization_id, contract_type_id,
                       title, status enum (intake|parsing|ready_for_review|approved|executed|rejected),
                       owner_user_id, created_at, updated_at)

contract_documents    (id uuid, organization_id, contract_id,
                       storage_key text, mime_type, sha256 bytea,
                       byte_size bigint, parsed_text text NULL,
                       page_count int NULL, created_at)

extracted_fields      (id uuid, organization_id, contract_id,
                       document_id, field_name, field_value text,
                       field_value_json jsonb,
                       page_or_paragraph text, span_start int, span_end int,
                       model_id text, prompt_version text,
                       extracted_at timestamptz)
```

`extracted_fields` carries the Citation tuple inline (rather than a
separate `citations` table) for this story; if/when Citations gain
independent lifecycle (e.g. AI redlines bind multiple to one
suggestion), they get extracted in Phase 3.

Audit actions added:
- `contract.created`
- `contract.document_uploaded`
- `contract.parse_started` / `contract.parse_completed`
- `contract.extraction_started` / `contract.extraction_completed`
- `contract.transitioned` (with before/after status)

## Proto contracts

### proto/document.proto

```proto
service Document {
  rpc Health(HealthRequest) returns (HealthResponse);

  // Parse extracts text + layout from a PDF or DOCX.
  rpc Parse(ParseRequest) returns (ParseResponse);
}

message ParseRequest {
  string document_id = 1;
  bytes content = 2;
  string mime_type = 3;  // "application/pdf" or "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
}

message ParseResponse {
  string text = 1;
  int32 page_count = 2;
  // Per-page or per-paragraph offset map for citation resolution.
  repeated TextSegment segments = 3;
}

message TextSegment {
  string locator = 1;   // "page:1" or "para:23"
  int32 char_start = 2;
  int32 char_end = 3;
}
```

### proto/ai.proto

```proto
service AI {
  rpc Health(HealthRequest) returns (HealthResponse);

  // ExtractFields returns Citation-bearing structured output.
  rpc ExtractFields(ExtractRequest) returns (ExtractResponse);
}

message ExtractRequest {
  string document_id = 1;
  string parsed_text = 2;
  repeated TextSegment segments = 3;
  repeated string field_names = 4;  // e.g., ["parties","effective_date",...]
  string model_id = 5;              // "anthropic/claude-3-5-sonnet"
  string prompt_version = 6;        // bumped on prompt change
}

message ExtractResponse {
  repeated ExtractedField fields = 1;
}

message ExtractedField {
  string field_name = 1;
  string value = 2;
  string value_json = 3;        // optional structured form (parties=array)
  Citation citation = 4;
}

message Citation {
  string document_id = 1;
  string locator = 2;           // "page:3" or "para:14"
  int32 span_start = 3;
  int32 span_end = 4;
  string model_id = 5;
  string prompt_version = 6;
  string timestamp = 7;         // RFC3339
}
```

The Go core treats responses without citations on every field as a
protocol violation and returns 502 to the API caller.

## Temporal workflow

`internal/workflow/contract_lifecycle.go`:

```go
ContractLifecycleWorkflow(ctx, contract_id) {
  document = activity.GetContractDocument(ctx, contract_id)

  // Activity wraps the gRPC call to document-sidecar.
  parsed = activity.ParseDocument(ctx, document)
  activity.PersistParsedText(ctx, document.id, parsed)

  // Activity wraps the gRPC call to ai-sidecar.
  extracted = activity.ExtractFields(ctx, parsed, ContractType.NDA.fields)
  activity.PersistExtractedFields(ctx, contract_id, extracted)

  activity.TransitionContract(ctx, contract_id, "ready_for_review")

  // Wait for approval signal — never arrives in Story 2.
  workflow.GetSignalChannel("approval_decision").Receive(ctx)
}
```

Activities are idempotent on `(contract_id, document_id)`. The
workflow ID is `contract:{contract_id}` so a duplicate trigger is a
no-op via Temporal's de-dup.

## API surface (delta on schemas/openapi.yaml)

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | `/contracts` | session+org | Create contract + upload document (multipart). Returns 201 with contract id. Starts ContractLifecycleWorkflow. |
| GET | `/contracts` | session+org | List org's contracts (was empty; now returns rows). |
| GET | `/contracts/{id}` | session+org | Contract detail with extracted fields + status. |
| GET | `/contracts/{id}/document` | session+org | Signed URL to MinIO for the latest ContractDocument. |

`POST /contracts` shape:
- `multipart/form-data`
- `title` (string), `file` (binary)
- Returns `{ id, status, contract_type, created_at }`

## Frontend pages

- `/contracts/new` — upload form (title + file picker), client component, posts multipart
- `/contracts` — table: title, status, created_at, link to detail
- `/contracts/[id]` — server component:
  - Header: title + status badge + "Download document" link
  - Section: "Extracted fields" — 5 rows, each with field name, value, citation locator (clickable to focus the doc viewer in Story 5), confidence-style chip showing model_id + prompt_version
  - Polling section: if status is `parsing` or `intake`, refresh every 2s until `ready_for_review`. (Server-component refresh via `revalidatePath` or a small client-component poller.)

## Substrate copy-in this story

Nothing new from statebound — Story 2 is mostly application code. The
audit chain, signing, OIDC primitives are all in place from Story 1.

## Out of scope (deferred to later Phase 1 stories)

- Approval routes and routing (Story 3)
- Playbook rule evaluation beyond the empty placeholder activity
  (Story 3)
- DocuSign integration (Story 4)
- Repository search and filters (Story 5)
- Multi-contract-type per org (Phase 2)
- Document layout-aware extraction with Docling (Phase 2)
- AI redlines (Phase 3)
- Counterparty access (Phase 5)

## Acceptance criteria

1. Authenticated user uploads an NDA PDF; backend stores it in MinIO,
   starts the workflow, parses, extracts 5 Citation-bearing fields,
   contract transitions to `ready_for_review`.
2. Same flow works for a DOCX upload.
3. AI response without citations on every field returns 502 to the
   caller; the contract stays in `parsing` (not transitioned).
4. Two orgs uploading concurrently can't see each other's contracts
   via any API call (multi-tenant isolation test extended to cover
   contracts).
5. Hash-chained audit log records every state change including the
   workflow's `contract.transitioned` events; chain integrity remains
   0 broken links.
6. `make lint typecheck test` clean. Generated TS types and proto
   stubs pass drift checks in CI.

## Estimated effort

5–8 developer-days. Roughly:

- Day 1: migration + ContractType + Contract + ContractDocument
  schema, repo, audit actions
- Day 2: MinIO client + storage abstraction, multipart upload
  handler, signed URL helper
- Day 3: proto codegen wiring (Go server-stubs into internal/{ai,document}/...grpc/, Python server-stubs into apps/{ai,document}-sidecar/src/...proto/), Health RPCs implemented end-to-end
- Day 4: document-sidecar Parse handler (PyMuPDF + python-docx) +
  fixture-based determinism tests
- Day 5: ai-sidecar ExtractFields handler (LiteLLM + Pydantic +
  recorded-fixture tests; live-call tests behind eval-set flag)
- Day 6: Temporal worker registration + ContractLifecycleWorkflow +
  activities + workflow tests via testsuite
- Day 7: API handlers (POST/GET/GET/{id}/document) + ExtractedField
  repo + tenant isolation extension
- Day 8: Frontend (intake form, list-with-status, detail with
  fields), drift checks, end-to-end smoke + audit chain verify, tag
  `phase-1-story-2`

## Verification

End-to-end smoke: bring up the full stack via `make dev`; sign in;
upload `fixtures/sample-nda.pdf`; watch contract appear in list with
`parsing` status; refresh until `ready_for_review`; click into detail
view; see 5 fields with citation locators; verify audit chain via
psql shows the full state transition trail.

## References

- `docs/architecture.md` (domain model, AI design, data flows)
- `docs/superpowers/specs/2026-05-05-phase1-story1-design.md`
- `docs/superpowers/plans/2026-05-05-phase1-story1-signup-org-contracts.md`
- statebound — Temporal patterns (sidecar add-on)
