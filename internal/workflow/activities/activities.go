// Package activities implements the side-effecting steps invoked from
// ContractLifecycleWorkflow. Each activity is idempotent on its inputs
// (Temporal may retry).
package activities

import (
	"github.com/rupivbluegreen/pactline/internal/ai"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/document"
	"github.com/rupivbluegreen/pactline/internal/storage"
)

// Activities holds the dependencies registered with the Temporal worker.
// Methods on this struct are registered as activities by name.
type Activities struct {
	Pool              *database.Pool
	Storage           *storage.Client
	AI                *ai.Client
	Document          *document.Client
	Contracts         *database.ContractRepo
	ContractDocuments *database.ContractDocumentRepo
	ExtractedFields   *database.ExtractedFieldRepo
	ModelID           string
	PromptVersion     string
}

func New(pool *database.Pool, st *storage.Client, ac *ai.Client, dc *document.Client, modelID, promptVersion string) *Activities {
	return &Activities{
		Pool: pool, Storage: st, AI: ac, Document: dc,
		Contracts:         database.NewContractRepo(pool),
		ContractDocuments: database.NewContractDocumentRepo(pool),
		ExtractedFields:   database.NewExtractedFieldRepo(pool),
		ModelID:           modelID,
		PromptVersion:     promptVersion,
	}
}
