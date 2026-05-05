package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/rupivbluegreen/pactline/internal/api/handlers"
	"github.com/rupivbluegreen/pactline/internal/database"
)

func TestContractsList_NoOrg(t *testing.T) {
	h := &handlers.ContractsHandler{}
	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/contracts", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestContractsList_Empty(t *testing.T) {
	h := &handlers.ContractsHandler{}
	ctx := database.WithOrgID(context.Background(), uuid.New())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contracts", nil).WithContext(ctx)
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Error("empty body")
	}
}
