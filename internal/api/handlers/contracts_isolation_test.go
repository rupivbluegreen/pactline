package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/mocks"

	"github.com/rupivbluegreen/pactline/internal/api/handlers"
	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/storage"
)

func skipIfDown(t *testing.T, err error) {
	t.Helper()
	var ne *net.OpError
	if errors.As(err, &ne) {
		t.Skipf("dependency unavailable: %v", err)
	}
	msg := err.Error()
	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "dial tcp") {
		t.Skipf("dependency unavailable: %v", err)
	}
}

func TestContracts_MultiTenantIsolation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		skipIfDown(t, err)
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	if err := database.MigrateUp(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := storage.New(ctx, storage.ConfigFromEnv())
	if err != nil {
		skipIfDown(t, err)
		t.Fatalf("storage: %v", err)
	}

	mkOrg := func(seed string) (orgID, userID uuid.UUID) {
		orgID = uuid.New()
		if _, err := pool.Exec(ctx,
			`INSERT INTO organizations (id, slug, name) VALUES ($1, $2, $3)`,
			orgID, seed+"-"+orgID.String()[:6], seed); err != nil {
			t.Fatalf("org: %v", err)
		}
		userID = uuid.New()
		if _, err := pool.Exec(ctx,
			`INSERT INTO users (id, email) VALUES ($1, $2)`,
			userID, seed+"-"+userID.String()[:6]+"@test.example.com"); err != nil {
			t.Fatalf("user: %v", err)
		}
		ctID := uuid.New()
		if _, err := pool.Exec(ctx,
			`INSERT INTO contract_types (id, organization_id, slug, name) VALUES ($1, $2, $3, $4)`,
			ctID, orgID, core.ContractTypeSlugNDA, "NDA"); err != nil {
			t.Fatalf("ct: %v", err)
		}
		return
	}
	orgA, userA := mkOrg("orgA")
	orgB, userB := mkOrg("orgB")

	// Mock Temporal client — record ExecuteWorkflow + return a stub run.
	tcli := &mocks.Client{}
	wfRun := &mocks.WorkflowRun{}
	wfRun.On("GetID").Return("test-id").Maybe()
	wfRun.On("GetRunID").Return("test-run-id").Maybe()
	tcli.On("ExecuteWorkflow",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(wfRun, nil).Maybe()

	h := &handlers.ContractsHandler{
		Pool: pool, Storage: st,
		Contracts:         database.NewContractRepo(pool),
		ContractTypes:     database.NewContractTypeRepo(pool),
		ContractDocuments: database.NewContractDocumentRepo(pool),
		ExtractedFields:   database.NewExtractedFieldRepo(pool),
		Temporal:          tcli, TaskQueue: "pactline",
	}

	upload := func(orgID, userID uuid.UUID, title string) string {
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		_ = w.WriteField("title", title)
		fw, _ := w.CreateFormFile("file", "x.pdf")
		_, _ = fw.Write([]byte("%PDF-1.4\n%minimal\n"))
		_ = w.Close()

		req := httptest.NewRequest(http.MethodPost, "/contracts", body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		ctx := database.WithUserID(context.Background(), userID)
		ctx = database.WithOrgID(ctx, orgID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		h.Create(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("upload %s: status %d body %s", title, rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}

	upload(orgA, userA, "A1")
	upload(orgB, userB, "B1")

	// Org A list — must contain only A1.
	listReq := httptest.NewRequest(http.MethodGet, "/contracts", nil).
		WithContext(database.WithOrgID(context.Background(), orgA))
	listRec := httptest.NewRecorder()
	h.List(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list A: %d", listRec.Code)
	}

	var listA struct {
		Contracts []map[string]any `json:"contracts"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listA); err != nil {
		t.Fatalf("decode list A: %v", err)
	}
	titles := []string{}
	for _, c := range listA.Contracts {
		titles = append(titles, c["title"].(string))
	}
	for _, title := range titles {
		if title == "B1" {
			t.Errorf("isolation leak: orgA sees B1: %v", titles)
		}
	}
	hasA1 := false
	for _, title := range titles {
		if title == "A1" {
			hasA1 = true
		}
	}
	if !hasA1 {
		t.Errorf("orgA should see A1, got: %v", titles)
	}
}
