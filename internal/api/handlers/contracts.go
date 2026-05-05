package handlers

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"

	"github.com/rupivbluegreen/pactline/internal/audit"
	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/storage"
	pactlineworkflow "github.com/rupivbluegreen/pactline/internal/workflow"
)

const maxUploadBytes = 25 * 1024 * 1024 // 25 MiB

type ContractsHandler struct {
	Pool              *database.Pool
	Storage           *storage.Client
	Contracts         *database.ContractRepo
	ContractTypes     *database.ContractTypeRepo
	ContractDocuments *database.ContractDocumentRepo
	ExtractedFields   *database.ExtractedFieldRepo
	Temporal          client.Client
	TaskQueue         string
}

type contractRow struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type listContractsResponse struct {
	Contracts []contractRow `json:"contracts"`
}

type createContractResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	ContractType string `json:"contract_type"`
	CreatedAt    string `json:"created_at"`
}

type extractedFieldRow struct {
	FieldName       string `json:"field_name"`
	Value           string `json:"value"`
	ValueJSON       string `json:"value_json,omitempty"`
	PageOrParagraph string `json:"page_or_paragraph"`
	SpanStart       int    `json:"span_start"`
	SpanEnd         int    `json:"span_end"`
	ModelID         string `json:"model_id"`
	PromptVersion   string `json:"prompt_version"`
	ExtractedAt     string `json:"extracted_at"`
}

type contractDetailResponse struct {
	ID              string              `json:"id"`
	Title           string              `json:"title"`
	Status          string              `json:"status"`
	ContractType    string              `json:"contract_type"`
	CreatedAt       string              `json:"created_at"`
	UpdatedAt       string              `json:"updated_at"`
	ExtractedFields []extractedFieldRow `json:"extracted_fields"`
}

type signedURLResponse struct {
	URL       string `json:"url"`
	ExpiresIn int    `json:"expires_in_seconds"`
}

func (h *ContractsHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := database.TenantScoped(r.Context())
	if !ok {
		http.Error(w, "no organization", http.StatusBadRequest)
		return
	}
	rows, err := h.Contracts.ListByOrg(r.Context(), orgID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	out := listContractsResponse{Contracts: []contractRow{}}
	for _, c := range rows {
		out.Contracts = append(out.Contracts, contractRow{
			ID: c.ID.String(), Title: c.Title, Status: string(c.Status),
			CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: c.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *ContractsHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, ok := database.TenantScoped(r.Context())
	if !ok {
		http.Error(w, "no organization", http.StatusBadRequest)
		return
	}
	uid, ok := database.UserID(r.Context())
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		http.Error(w, "bad multipart", http.StatusBadRequest)
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	mime, ext, ok := acceptedMime(header)
	if !ok {
		http.Error(w, "unsupported file type (PDF or DOCX only)", http.StatusUnsupportedMediaType)
		return
	}
	body, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		http.Error(w, "read file", http.StatusBadRequest)
		return
	}
	if int64(len(body)) > maxUploadBytes {
		http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
		return
	}

	ct, err := h.ContractTypes.GetBySlug(r.Context(), orgID, core.ContractTypeSlugNDA)
	if err != nil {
		http.Error(w, "contract type missing", http.StatusInternalServerError)
		return
	}

	contractID := uuid.New()
	docID := uuid.New()
	storageKey := orgID.String() + "/" + contractID.String() + "/" + docID.String() + ext

	if err := h.Storage.Put(r.Context(), storageKey, mime, body); err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	sum := sha256.Sum256(body)

	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	created, err := h.Contracts.CreateTx(r.Context(), tx, &core.Contract{
		ID: contractID, OrganizationID: orgID, ContractTypeID: ct.ID,
		Title: title, Status: core.ContractStatusIntake, OwnerUserID: uid,
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	doc, err := h.ContractDocuments.CreateTx(r.Context(), tx, &core.ContractDocument{
		ID: docID, OrganizationID: orgID, ContractID: created.ID,
		StorageKey: storageKey, MimeType: mime, SHA256: sum[:], ByteSize: int64(len(body)),
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := audit.Write(r.Context(), tx, audit.Event{
		OrganizationID: &orgID, ActorUserID: &uid,
		Action: audit.ActionContractCreated, EntityType: "contract",
		EntityID: &created.ID,
		After:    map[string]string{"title": title, "contract_type": ct.Slug},
	}); err != nil {
		http.Error(w, "audit error", http.StatusInternalServerError)
		return
	}
	if err := audit.Write(r.Context(), tx, audit.Event{
		OrganizationID: &orgID, ActorUserID: &uid,
		Action: audit.ActionContractDocumentUploaded, EntityType: "contract_document",
		EntityID: &doc.ID,
		After: map[string]string{
			"contract_id": created.ID.String(),
			"storage_key": storageKey,
			"mime_type":   mime,
		},
	}); err != nil {
		http.Error(w, "audit error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "commit error", http.StatusInternalServerError)
		return
	}

	if err := h.startWorkflow(r.Context(), created, doc); err != nil {
		http.Error(w, "workflow start failed", http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusCreated, createContractResponse{
		ID: created.ID.String(), Status: string(created.Status),
		ContractType: ct.Slug,
		CreatedAt:    created.CreatedAt.UTC().Format(time.RFC3339),
	})
}

func (h *ContractsHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID, ok := database.TenantScoped(r.Context())
	if !ok {
		http.Error(w, "no organization", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	c, err := h.Contracts.GetByID(r.Context(), orgID, id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	ct, err := h.ContractTypes.GetBySlug(r.Context(), orgID, core.ContractTypeSlugNDA)
	if err != nil {
		http.Error(w, "contract type missing", http.StatusInternalServerError)
		return
	}
	fields, err := h.ExtractedFields.ListForContract(r.Context(), orgID, c.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	out := contractDetailResponse{
		ID: c.ID.String(), Title: c.Title, Status: string(c.Status),
		ContractType:    ct.Slug,
		CreatedAt:       c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       c.UpdatedAt.UTC().Format(time.RFC3339),
		ExtractedFields: []extractedFieldRow{},
	}
	for _, f := range fields {
		out.ExtractedFields = append(out.ExtractedFields, extractedFieldRow{
			FieldName:       f.FieldName,
			Value:           f.FieldValue,
			ValueJSON:       string(f.FieldValueJSON),
			PageOrParagraph: f.PageOrParagraph,
			SpanStart:       f.SpanStart, SpanEnd: f.SpanEnd,
			ModelID:       f.ModelID,
			PromptVersion: f.PromptVersion,
			ExtractedAt:   f.ExtractedAt.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *ContractsHandler) DocumentURL(w http.ResponseWriter, r *http.Request) {
	orgID, ok := database.TenantScoped(r.Context())
	if !ok {
		http.Error(w, "no organization", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	doc, err := h.ContractDocuments.GetLatestForContract(r.Context(), orgID, id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	ttl := 5 * time.Minute
	u, err := h.Storage.SignedURL(r.Context(), doc.StorageKey, ttl)
	if err != nil {
		http.Error(w, "sign error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, signedURLResponse{URL: u.String(), ExpiresIn: int(ttl.Seconds())})
}

func (h *ContractsHandler) startWorkflow(ctx context.Context, c *core.Contract, d *core.ContractDocument) error {
	_, err := h.Temporal.ExecuteWorkflow(ctx,
		client.StartWorkflowOptions{
			ID:                       pactlineworkflow.WorkflowID(c.ID.String()),
			TaskQueue:                h.TaskQueue,
			WorkflowExecutionTimeout: 30 * time.Minute,
		},
		pactlineworkflow.ContractLifecycleWorkflow,
		pactlineworkflow.ContractLifecycleInput{
			OrganizationID: c.OrganizationID,
			ContractID:     c.ID,
			DocumentID:     d.ID,
			StorageKey:     d.StorageKey,
			MimeType:       d.MimeType,
			OwnerUserID:    c.OwnerUserID,
		})
	return err
}

func acceptedMime(h *multipart.FileHeader) (string, string, bool) {
	name := strings.ToLower(filepath.Ext(h.Filename))
	switch name {
	case ".pdf":
		return core.MimePDF, ".pdf", true
	case ".docx":
		return core.MimeDOCX, ".docx", true
	}
	return "", "", false
}
