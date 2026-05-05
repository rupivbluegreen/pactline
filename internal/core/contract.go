package core

import (
	"time"

	"github.com/google/uuid"
)

type ContractStatus string

const (
	ContractStatusIntake         ContractStatus = "intake"
	ContractStatusParsing        ContractStatus = "parsing"
	ContractStatusReadyForReview ContractStatus = "ready_for_review"
	ContractStatusApproved       ContractStatus = "approved"
	ContractStatusExecuted       ContractStatus = "executed"
	ContractStatusRejected       ContractStatus = "rejected"
)

type Contract struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ContractTypeID uuid.UUID
	Title          string
	Status         ContractStatus
	OwnerUserID    uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
