-- +goose Up
-- +goose StatementBegin
CREATE TABLE contract_types (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    slug            text NOT NULL,
    name            text NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, slug)
);
CREATE INDEX contract_types_org_idx ON contract_types(organization_id);

CREATE TYPE contract_status AS ENUM (
    'intake', 'parsing', 'ready_for_review', 'approved', 'executed', 'rejected'
);

CREATE TABLE contracts (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id   uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    contract_type_id  uuid NOT NULL REFERENCES contract_types(id) ON DELETE RESTRICT,
    title             text NOT NULL,
    status            contract_status NOT NULL DEFAULT 'intake',
    owner_user_id     uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX contracts_org_created_idx ON contracts(organization_id, created_at DESC);

CREATE TABLE contract_documents (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    contract_id     uuid NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    storage_key     text NOT NULL,
    mime_type       text NOT NULL,
    sha256          bytea NOT NULL,
    byte_size       bigint NOT NULL CHECK (byte_size > 0),
    parsed_text     text,
    page_count      int,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX contract_documents_contract_idx ON contract_documents(contract_id, created_at DESC);

CREATE TABLE extracted_fields (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id    uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    contract_id        uuid NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    document_id        uuid NOT NULL REFERENCES contract_documents(id) ON DELETE CASCADE,
    field_name         text NOT NULL,
    field_value        text NOT NULL,
    field_value_json   jsonb,
    page_or_paragraph  text NOT NULL,
    span_start         int NOT NULL CHECK (span_start >= 0),
    span_end           int NOT NULL CHECK (span_end >= span_start),
    model_id           text NOT NULL,
    prompt_version     text NOT NULL,
    extracted_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (contract_id, document_id, field_name)
);
CREATE INDEX extracted_fields_contract_idx ON extracted_fields(contract_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS extracted_fields;
DROP TABLE IF EXISTS contract_documents;
DROP TABLE IF EXISTS contracts;
DROP TYPE IF EXISTS contract_status;
DROP TABLE IF EXISTS contract_types;
-- +goose StatementEnd
