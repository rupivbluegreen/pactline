package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/organizations"
)

type OrgsHandlers struct {
	Svc *organizations.Service
}

type createOrgBody struct {
	Name string `json:"name"`
	Slug string `json:"slug,omitempty"`
}

type orgResponse struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func (h *OrgsHandlers) Create(w http.ResponseWriter, r *http.Request) {
	uid, ok := database.UserID(r.Context())
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	var body createOrgBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	o, err := h.Svc.CreateForUser(r.Context(), uid, body.Name, body.Slug)
	if err != nil {
		if errors.Is(err, core.ErrAlreadyExists) {
			http.Error(w, "slug taken", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, orgResponse{ID: o.ID.String(), Slug: o.Slug, Name: o.Name})
}
