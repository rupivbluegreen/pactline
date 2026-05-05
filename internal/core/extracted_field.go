package core

import (
	"time"

	"github.com/google/uuid"
)

type ExtractedField struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	ContractID      uuid.UUID
	DocumentID      uuid.UUID
	FieldName       string
	FieldValue      string
	FieldValueJSON  []byte
	PageOrParagraph string
	SpanStart       int
	SpanEnd         int
	ModelID         string
	PromptVersion   string
	ExtractedAt     time.Time
}

// Citation is the canonical evidence tuple. Every ExtractedField projects to
// one Citation; AI responses missing any of these fields are protocol
// violations.
type Citation struct {
	DocumentID      uuid.UUID
	PageOrParagraph string
	SpanStart       int
	SpanEnd         int
	ModelID         string
	PromptVersion   string
	Timestamp       time.Time
}

// NDAFieldNames are the five fields v1 extracts from every NDA.
var NDAFieldNames = []string{
	"parties", "effective_date", "term", "value", "governing_law",
}
