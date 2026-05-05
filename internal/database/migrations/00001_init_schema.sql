-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         citext NOT NULL UNIQUE,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_login_at timestamptz
);

CREATE TABLE organizations (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       citext NOT NULL UNIQUE,
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE memberships (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role            text NOT NULL CHECK (role IN ('owner','admin','reviewer','approver','viewer')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, organization_id)
);
CREATE INDEX memberships_user_idx ON memberships(user_id);
CREATE INDEX memberships_org_idx  ON memberships(organization_id);

CREATE TABLE sessions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  bytea NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    revoked_at  timestamptz
);
CREATE INDEX sessions_user_idx ON sessions(user_id);

CREATE TABLE magic_links (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email       citext NOT NULL,
    token_hash  bytea NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    used_at     timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX magic_links_email_idx ON magic_links(email);

CREATE TABLE audit_events (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid REFERENCES organizations(id) ON DELETE RESTRICT,
    actor_user_id   uuid REFERENCES users(id) ON DELETE RESTRICT,
    action          text NOT NULL,
    entity_type     text NOT NULL,
    entity_id       uuid,
    before          jsonb,
    after           jsonb,
    prev_hash       bytea,
    event_hash      bytea NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_events_org_created_idx ON audit_events(organization_id, created_at);
CREATE INDEX audit_events_chain_idx ON audit_events(organization_id, created_at DESC);

-- canonical_event_payload returns the bytes that get hashed for a given row.
-- Stable across re-imports: same logical event -> same bytes -> same hash.
CREATE OR REPLACE FUNCTION canonical_event_payload(
    p_org uuid, p_actor uuid, p_action text, p_entity_type text,
    p_entity_id uuid, p_before jsonb, p_after jsonb, p_created_at timestamptz
) RETURNS bytea LANGUAGE sql IMMUTABLE AS $$
    SELECT convert_to(
        json_build_object(
            'organization_id', p_org,
            'actor_user_id', p_actor,
            'action', p_action,
            'entity_type', p_entity_type,
            'entity_id', p_entity_id,
            'before', p_before,
            'after', p_after,
            'created_at', to_char(p_created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        )::text,
        'UTF8'
    );
$$;

-- BEFORE INSERT trigger: looks up prev event in same chain (per organization_id),
-- sets prev_hash and event_hash on the new row.
CREATE OR REPLACE FUNCTION audit_event_set_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_prev bytea;
BEGIN
    SELECT event_hash INTO v_prev
    FROM audit_events
    WHERE organization_id IS NOT DISTINCT FROM NEW.organization_id
    ORDER BY created_at DESC, id DESC
    LIMIT 1;

    NEW.prev_hash := v_prev;
    NEW.event_hash := digest(
        coalesce(v_prev, ''::bytea) ||
        canonical_event_payload(
            NEW.organization_id, NEW.actor_user_id, NEW.action,
            NEW.entity_type, NEW.entity_id, NEW.before, NEW.after, NEW.created_at
        ),
        'sha256'
    );
    RETURN NEW;
END;
$$;

CREATE TRIGGER audit_events_hash_trigger
BEFORE INSERT ON audit_events
FOR EACH ROW EXECUTE FUNCTION audit_event_set_hash();

-- Block UPDATE / DELETE on audit_events at the table level.
CREATE OR REPLACE FUNCTION audit_events_immutable() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_events is append-only';
END;
$$;
CREATE TRIGGER audit_events_no_update BEFORE UPDATE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_immutable();
CREATE TRIGGER audit_events_no_delete BEFORE DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_immutable();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS audit_events_no_delete ON audit_events;
DROP TRIGGER IF EXISTS audit_events_no_update ON audit_events;
DROP TRIGGER IF EXISTS audit_events_hash_trigger ON audit_events;
DROP FUNCTION IF EXISTS audit_events_immutable();
DROP FUNCTION IF EXISTS audit_event_set_hash();
DROP FUNCTION IF EXISTS canonical_event_payload(uuid, uuid, text, text, uuid, jsonb, jsonb, timestamptz);
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS magic_links;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
