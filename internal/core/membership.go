package core

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner    Role = "owner"
	RoleAdmin    Role = "admin"
	RoleReviewer Role = "reviewer"
	RoleApprover Role = "approver"
	RoleViewer   Role = "viewer"
)

type Membership struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Role           Role
	CreatedAt      time.Time
}
