package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/rupivbluegreen/pactline/internal/api/middleware"
	"github.com/rupivbluegreen/pactline/internal/auth"
	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

type AuthHandlers struct {
	Svc *auth.Service
}

type magicLinkRequestBody struct {
	Email string `json:"email"`
}

func (h *AuthHandlers) RequestMagicLink(w http.ResponseWriter, r *http.Request) {
	var body magicLinkRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := h.Svc.RequestMagicLink(r.Context(), body.Email); err != nil {
		http.Error(w, "request failed", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type verifyRequestBody struct {
	Token string `json:"token"`
}

type verifyResponse struct {
	SessionToken string    `json:"session_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       string    `json:"user_id"`
	UserEmail    string    `json:"user_email"`
}

func (h *AuthHandlers) Verify(w http.ResponseWriter, r *http.Request) {
	var body verifyRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	res, err := h.Svc.VerifyMagicLink(r.Context(), database.MagicLinkToken(body.Token))
	if err != nil {
		switch {
		case errors.Is(err, core.ErrInvalidToken),
			errors.Is(err, core.ErrTokenExpired),
			errors.Is(err, core.ErrTokenConsumed):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    string(res.SessionToken),
		Expires:  res.ExpiresAt,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, verifyResponse{
		SessionToken: string(res.SessionToken),
		ExpiresAt:    res.ExpiresAt,
		UserID:       res.User.ID.String(),
		UserEmail:    res.User.Email,
	})
}

func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	tok, _ := r.Cookie(middleware.SessionCookieName)
	if tok != nil {
		_ = h.Svc.Logout(r.Context(), database.SessionToken(tok.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Expires:  time.Unix(0, 0),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}
