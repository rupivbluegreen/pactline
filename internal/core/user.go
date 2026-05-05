package core

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID
	Email       string
	CreatedAt   time.Time
	LastLoginAt *time.Time
}
