-- +goose Up
-- +goose StatementBegin
-- Serialize all writes to a chain via a per-(org_id) advisory transaction
-- lock. The original FOR UPDATE on the previous row covers the case where
-- a previous event exists, but for genesis inserts (no rows yet) FOR UPDATE
-- has nothing to lock — two concurrent genesis writers can both observe an
-- empty chain and both insert with prev_hash=NULL, forking the chain.
-- pg_advisory_xact_lock blocks the second writer until the first commits.
CREATE OR REPLACE FUNCTION audit_event_set_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_prev bytea;
BEGIN
    PERFORM pg_advisory_xact_lock(
        hashtextextended(coalesce(NEW.organization_id::text, 'system'), 0)
    );

    SELECT event_hash INTO v_prev
    FROM audit_events
    WHERE organization_id IS NOT DISTINCT FROM NEW.organization_id
    ORDER BY created_at DESC, id DESC
    LIMIT 1
    FOR UPDATE;

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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION audit_event_set_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_prev bytea;
BEGIN
    SELECT event_hash INTO v_prev
    FROM audit_events
    WHERE organization_id IS NOT DISTINCT FROM NEW.organization_id
    ORDER BY created_at DESC, id DESC
    LIMIT 1
    FOR UPDATE;

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
-- +goose StatementEnd
