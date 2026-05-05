package core

import (
	"time"

	"github.com/google/uuid"
)

type ContractType struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Slug           string
	Name           string
	CreatedAt      time.Time
}

const ContractTypeSlugNDA = "nda"
