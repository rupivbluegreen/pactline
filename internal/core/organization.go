package core

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID        uuid.UUID
	Slug      string
	Name      string
	CreatedAt time.Time
}
