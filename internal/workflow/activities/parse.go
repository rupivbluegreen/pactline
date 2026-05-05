package activities

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/audit"
	"github.com/rupivbluegreen/pactline/internal/document/documentgrpc"
)

type ParseInput struct {
	OrganizationID uuid.UUID
	ContractID     uuid.UUID
	DocumentID     uuid.UUID
	StorageKey     string
	MimeType       string
}

type ParseOutput struct {
	Text      string
	PageCount int32
	Segments  []*documentgrpc.TextSegment
}

// ParseDocument fetches the bytes from storage, calls the document sidecar,
// and returns the parsed text + segments.
func (a *Activities) ParseDocument(ctx context.Context, in ParseInput) (*ParseOutput, error) {
	if err := audit.Write(ctx, a.Pool, audit.Event{
		OrganizationID: &in.OrganizationID, EntityType: "contract",
		EntityID: &in.ContractID, Action: audit.ActionContractParseStarted,
		After: map[string]string{"document_id": in.DocumentID.String()},
	}); err != nil {
		return nil, fmt.Errorf("audit parse_started: %w", err)
	}

	body, err := a.Storage.Get(ctx, in.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("storage get: %w", err)
	}
	resp, err := a.Document.Parse(ctx, in.DocumentID.String(), body, in.MimeType)
	if err != nil {
		return nil, fmt.Errorf("document parse: %w", err)
	}

	if err := audit.Write(ctx, a.Pool, audit.Event{
		OrganizationID: &in.OrganizationID, EntityType: "contract",
		EntityID: &in.ContractID, Action: audit.ActionContractParseCompleted,
		After: map[string]string{
			"document_id": in.DocumentID.String(),
			"page_count":  fmt.Sprintf("%d", resp.GetPageCount()),
			"text_length": fmt.Sprintf("%d", len(resp.GetText())),
		},
	}); err != nil {
		return nil, fmt.Errorf("audit parse_completed: %w", err)
	}

	return &ParseOutput{
		Text:      resp.GetText(),
		PageCount: resp.GetPageCount(),
		Segments:  resp.GetSegments(),
	}, nil
}
