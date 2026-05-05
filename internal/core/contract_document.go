package core

import (
	"time"

	"github.com/google/uuid"
)

const (
	MimePDF  = "application/pdf"
	MimeDOCX = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
)

type ContractDocument struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ContractID     uuid.UUID
	StorageKey     string
	MimeType       string
	SHA256         []byte
	ByteSize       int64
	ParsedText     *string
	PageCount      *int
	CreatedAt      time.Time
}
