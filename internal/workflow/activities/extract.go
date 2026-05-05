package activities

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/audit"
	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/document/documentgrpc"
)

type ExtractInput struct {
	OrganizationID uuid.UUID
	ContractID     uuid.UUID
	DocumentID     uuid.UUID
	ParsedText     string
	Segments       []*documentgrpc.TextSegment
}

type ExtractOutput struct {
	Fields []core.ExtractedField
}

// ExtractFields invokes the AI sidecar and converts the response into core
// ExtractedField rows ready for persistence. Citation completeness is
// enforced server-side; this activity surfaces any gRPC error verbatim.
func (a *Activities) ExtractFields(ctx context.Context, in ExtractInput) (*ExtractOutput, error) {
	if err := audit.Write(ctx, a.Pool, audit.Event{
		OrganizationID: &in.OrganizationID, EntityType: "contract",
		EntityID: &in.ContractID, Action: audit.ActionContractExtractionStarted,
		After: map[string]string{
			"document_id":    in.DocumentID.String(),
			"model_id":       a.ModelID,
			"prompt_version": a.PromptVersion,
		},
	}); err != nil {
		return nil, fmt.Errorf("audit extraction_started: %w", err)
	}

	resp, err := a.AI.ExtractFields(ctx,
		in.DocumentID.String(), in.ParsedText, in.Segments,
		core.NDAFieldNames, a.ModelID, a.PromptVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("ai extract: %w", err)
	}

	out := make([]core.ExtractedField, 0, len(resp.GetFields()))
	for _, f := range resp.GetFields() {
		c := f.GetCitation()
		out = append(out, core.ExtractedField{
			OrganizationID:  in.OrganizationID,
			ContractID:      in.ContractID,
			DocumentID:      in.DocumentID,
			FieldName:       f.GetFieldName(),
			FieldValue:      f.GetValue(),
			FieldValueJSON:  []byte(f.GetValueJson()),
			PageOrParagraph: c.GetLocator(),
			SpanStart:       int(c.GetSpanStart()),
			SpanEnd:         int(c.GetSpanEnd()),
			ModelID:         c.GetModelId(),
			PromptVersion:   c.GetPromptVersion(),
		})
	}

	if err := audit.Write(ctx, a.Pool, audit.Event{
		OrganizationID: &in.OrganizationID, EntityType: "contract",
		EntityID: &in.ContractID, Action: audit.ActionContractExtractionCompleted,
		After: map[string]string{
			"document_id": in.DocumentID.String(),
			"field_count": fmt.Sprintf("%d", len(out)),
		},
	}); err != nil {
		return nil, fmt.Errorf("audit extraction_completed: %w", err)
	}
	return &ExtractOutput{Fields: out}, nil
}
