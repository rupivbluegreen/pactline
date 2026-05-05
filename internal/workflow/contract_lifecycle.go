// Package workflow holds Temporal workflow definitions for pactline.
//
// Workflows must be deterministic. No clocks, randomness, or IO outside
// activities. Activities live in internal/workflow/activities.
package workflow

import (
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/workflow/activities"
)

type ContractLifecycleInput struct {
	OrganizationID uuid.UUID
	ContractID     uuid.UUID
	DocumentID     uuid.UUID
	StorageKey     string
	MimeType       string
	OwnerUserID    uuid.UUID
}

// ContractLifecycleWorkflow drives an uploaded contract through parse →
// extract → ready_for_review, then waits for an approval signal that lands
// in a later story.
func ContractLifecycleWorkflow(ctx workflow.Context, in ContractLifecycleInput) error {
	opts := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, opts)

	var a *activities.Activities // type-only handle for symbol references

	// Transition to parsing.
	if err := workflow.ExecuteActivity(ctx, a.TransitionContract, activities.TransitionInput{
		OrganizationID: in.OrganizationID, ContractID: in.ContractID,
		NextStatus: core.ContractStatusParsing, ActorUserID: &in.OwnerUserID,
	}).Get(ctx, nil); err != nil {
		return err
	}

	// Parse via document sidecar.
	var parsed activities.ParseOutput
	if err := workflow.ExecuteActivity(ctx, a.ParseDocument, activities.ParseInput{
		OrganizationID: in.OrganizationID, ContractID: in.ContractID,
		DocumentID: in.DocumentID, StorageKey: in.StorageKey, MimeType: in.MimeType,
	}).Get(ctx, &parsed); err != nil {
		return err
	}

	if err := workflow.ExecuteActivity(ctx, a.PersistParsedText, activities.PersistParsedInput{
		OrganizationID: in.OrganizationID, DocumentID: in.DocumentID,
		Text: parsed.Text, PageCount: parsed.PageCount,
	}).Get(ctx, nil); err != nil {
		return err
	}

	// Extract via AI sidecar.
	var extracted activities.ExtractOutput
	if err := workflow.ExecuteActivity(ctx, a.ExtractFields, activities.ExtractInput{
		OrganizationID: in.OrganizationID, ContractID: in.ContractID,
		DocumentID: in.DocumentID, ParsedText: parsed.Text, Segments: parsed.Segments,
	}).Get(ctx, &extracted); err != nil {
		return err
	}

	if err := workflow.ExecuteActivity(ctx, a.PersistExtractedFields, activities.PersistExtractedInput{
		OrganizationID: in.OrganizationID, Fields: extracted.Fields,
	}).Get(ctx, nil); err != nil {
		return err
	}

	// Transition to ready_for_review.
	if err := workflow.ExecuteActivity(ctx, a.TransitionContract, activities.TransitionInput{
		OrganizationID: in.OrganizationID, ContractID: in.ContractID,
		NextStatus: core.ContractStatusReadyForReview, ActorUserID: &in.OwnerUserID,
	}).Get(ctx, nil); err != nil {
		return err
	}

	// Wait for approval signal — never arrives in Story 2.
	var decision string
	workflow.GetSignalChannel(ctx, ContractApprovalSignalName).Receive(ctx, &decision)
	return nil
}
