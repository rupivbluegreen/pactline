package workflow_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/workflow"
	"github.com/rupivbluegreen/pactline/internal/workflow/activities"
)

func TestContractLifecycle_HappyPath(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	a := &activities.Activities{}
	env.RegisterActivity(a.TransitionContract)
	env.RegisterActivity(a.ParseDocument)
	env.RegisterActivity(a.PersistParsedText)
	env.RegisterActivity(a.ExtractFields)
	env.RegisterActivity(a.PersistExtractedFields)

	contractID := uuid.New()
	docID := uuid.New()
	orgID := uuid.New()
	userID := uuid.New()

	env.OnActivity(a.TransitionContract, mock.Anything, mock.MatchedBy(func(in activities.TransitionInput) bool {
		return in.NextStatus == core.ContractStatusParsing
	})).Return(nil).Once()

	env.OnActivity(a.ParseDocument, mock.Anything, mock.Anything).
		Return(&activities.ParseOutput{Text: "hello", PageCount: 1}, nil).Once()

	env.OnActivity(a.PersistParsedText, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.OnActivity(a.ExtractFields, mock.Anything, mock.Anything).
		Return(&activities.ExtractOutput{Fields: []core.ExtractedField{{
			OrganizationID: orgID, ContractID: contractID, DocumentID: docID,
			FieldName: "parties", FieldValue: "x",
			PageOrParagraph: "page:1", SpanStart: 0, SpanEnd: 1,
			ModelID: "m", PromptVersion: "v1",
		}}}, nil).Once()

	env.OnActivity(a.PersistExtractedFields, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.OnActivity(a.TransitionContract, mock.Anything, mock.MatchedBy(func(in activities.TransitionInput) bool {
		return in.NextStatus == core.ContractStatusReadyForReview
	})).Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(workflow.ContractApprovalSignalName, "approved")
	}, 0)

	env.ExecuteWorkflow(workflow.ContractLifecycleWorkflow, workflow.ContractLifecycleInput{
		OrganizationID: orgID, ContractID: contractID, DocumentID: docID,
		StorageKey: "k", MimeType: core.MimePDF, OwnerUserID: userID,
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	env.AssertExpectations(t)
}

func TestContractLifecycle_ParseFails_NoTransitionToReady(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	a := &activities.Activities{}
	env.RegisterActivity(a.TransitionContract)
	env.RegisterActivity(a.ParseDocument)
	env.RegisterActivity(a.PersistParsedText)
	env.RegisterActivity(a.ExtractFields)
	env.RegisterActivity(a.PersistExtractedFields)

	env.OnActivity(a.TransitionContract, mock.Anything, mock.MatchedBy(func(in activities.TransitionInput) bool {
		return in.NextStatus == core.ContractStatusParsing
	})).Return(nil).Once()

	env.OnActivity(a.ParseDocument, mock.Anything, mock.Anything).
		Return(nil, errors.New("sidecar boom"))

	env.ExecuteWorkflow(workflow.ContractLifecycleWorkflow, workflow.ContractLifecycleInput{
		OrganizationID: uuid.New(), ContractID: uuid.New(), DocumentID: uuid.New(),
		StorageKey: "k", MimeType: core.MimePDF, OwnerUserID: uuid.New(),
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow not completed")
	}
	if err := env.GetWorkflowError(); err == nil {
		t.Fatal("expected workflow error after Parse failures")
	}
}
