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

func TestContractsCreate_NoOrg(t *testing.T) {
	h := &handlers.ContractsHandler{}
	req := httptest.NewRequest(http.MethodPost, "/contracts", nil)
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestContractsCreate_NoUser(t *testing.T) {
	h := &handlers.ContractsHandler{}
	ctx := database.WithOrgID(context.Background(), uuid.New())
	req := httptest.NewRequest(http.MethodPost, "/contracts", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
