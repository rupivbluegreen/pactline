package handlers

import (
	"net/http"

	"github.com/rupivbluegreen/pactline/internal/database"
)

type MeHandler struct {
	Users       *database.UserRepo
	Memberships *database.MembershipRepo
	Orgs        *database.OrganizationRepo
}

type meResponse struct {
	UserID         string                  `json:"user_id"`
	UserEmail      string                  `json:"user_email"`
	CurrentOrgID   *string                 `json:"current_organization_id,omitempty"`
	CurrentOrgName *string                 `json:"current_organization_name,omitempty"`
	Memberships    []membershipResponseRow `json:"memberships"`
}

type membershipResponseRow struct {
	OrganizationID   string `json:"organization_id"`
	OrganizationSlug string `json:"organization_slug"`
	OrganizationName string `json:"organization_name"`
	Role             string `json:"role"`
}

func (h *MeHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid, ok := database.UserID(r.Context())
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	u, err := h.Users.GetByID(r.Context(), uid)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ms, err := h.Memberships.ListForUser(r.Context(), uid)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	out := meResponse{UserID: u.ID.String(), UserEmail: u.Email, Memberships: []membershipResponseRow{}}
	for _, m := range ms {
		o, err := h.Orgs.GetByID(r.Context(), m.OrganizationID)
		if err != nil {
			continue
		}
		out.Memberships = append(out.Memberships, membershipResponseRow{
			OrganizationID: o.ID.String(), OrganizationSlug: o.Slug,
			OrganizationName: o.Name, Role: string(m.Role),
		})
	}
	if orgID, ok := database.TenantScoped(r.Context()); ok {
		if o, err := h.Orgs.GetByID(r.Context(), orgID); err == nil {
			id := o.ID.String()
			out.CurrentOrgID = &id
			out.CurrentOrgName = &o.Name
		}
	}
	writeJSON(w, http.StatusOK, out)
}
