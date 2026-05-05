// Package audit is the hash-chained immutable audit log.
//
// Every state change writes one Event via Write. The Postgres trigger
// audit_events_hash_trigger sets prev_hash and event_hash on insert.
package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// Action is the canonical name of an audited operation.
type Action string

const (
	ActionUserCreated         Action = "user.created"
	ActionMagicLinkRequested  Action = "magic_link.requested"
	ActionMagicLinkConsumed   Action = "magic_link.consumed"
	ActionSessionCreated      Action = "session.created"
	ActionSessionRevoked      Action = "session.revoked"
	ActionOrganizationCreated Action = "organization.created"
	ActionMembershipCreated   Action = "membership.created"
)

// Event is one row in audit_events. prev_hash and event_hash are computed
// by the database trigger; do not set them in app code.
type Event struct {
	OrganizationID *uuid.UUID
	ActorUserID    *uuid.UUID
	Action         Action
	EntityType     string
	EntityID       *uuid.UUID
	Before         any
	After          any
}

// Querier is the subset of pgx that audit needs. Both *pgxpool.Pool and
// pgx.Tx satisfy it.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func Write(ctx context.Context, q Querier, e Event) error {
	beforeJSON, err := jsonValue(e.Before)
	if err != nil {
		return fmt.Errorf("encode before: %w", err)
	}
	afterJSON, err := jsonValue(e.After)
	if err != nil {
		return fmt.Errorf("encode after: %w", err)
	}

	_, err = q.Exec(ctx, `
		INSERT INTO audit_events (
			organization_id, actor_user_id, action, entity_type, entity_id, before, after
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, e.OrganizationID, e.ActorUserID, string(e.Action), e.EntityType, e.EntityID, beforeJSON, afterJSON)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func jsonValue(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}
