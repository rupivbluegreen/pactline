package handlers

import (
	"net/http"

	"github.com/rupivbluegreen/pactline/internal/database"
)

type ContractsHandler struct{}

type contractsResponse struct {
	Contracts []any `json:"contracts"`
}

func (h *ContractsHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := database.TenantScoped(r.Context()); !ok {
		http.Error(w, "no organization", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, contractsResponse{Contracts: []any{}})
}
