package activities

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/audit"
	"github.com/rupivbluegreen/pactline/internal/core"
)

type TransitionInput struct {
	OrganizationID uuid.UUID
	ContractID     uuid.UUID
	NextStatus     core.ContractStatus
	ActorUserID    *uuid.UUID
}

func (a *Activities) TransitionContract(ctx context.Context, in TransitionInput) error {
	prev, err := a.Contracts.UpdateStatus(ctx, in.OrganizationID, in.ContractID, in.NextStatus)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return audit.Write(ctx, a.Pool, audit.Event{
		OrganizationID: &in.OrganizationID,
		ActorUserID:    in.ActorUserID,
		Action:         audit.ActionContractTransitioned,
		EntityType:     "contract",
		EntityID:       &in.ContractID,
		Before:         map[string]string{"status": string(prev)},
		After:          map[string]string{"status": string(in.NextStatus)},
	})
}
