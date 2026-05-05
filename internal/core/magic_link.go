package core

import (
	"time"

	"github.com/google/uuid"
)

type MagicLink struct {
	ID        uuid.UUID
	Email     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
