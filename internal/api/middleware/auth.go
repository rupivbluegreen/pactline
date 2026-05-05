// Package middleware holds chi-compatible HTTP middleware for the API.
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rupivbluegreen/pactline/internal/core"
	"github.com/rupivbluegreen/pactline/internal/database"
)

const SessionCookieName = "pactline_session"

// Auth attaches the user (and current organization, if any) to the request
// context. Returns 401 if no valid session.
func Auth(sessions *database.SessionRepo, memberships *database.MembershipRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := tokenFromRequest(r)
			if tok == "" {
				http.Error(w, "unauthenticated", http.StatusUnauthorized)
				return
			}
			sess, err := sessions.GetByToken(r.Context(), database.SessionToken(tok))
			if err != nil {
				switch {
				case errors.Is(err, core.ErrNotFound),
					errors.Is(err, core.ErrSessionRevoked),
					errors.Is(err, core.ErrTokenExpired):
					http.Error(w, "unauthenticated", http.StatusUnauthorized)
				default:
					http.Error(w, "internal error", http.StatusInternalServerError)
				}
				return
			}

			ctx := database.WithUserID(r.Context(), sess.UserID)

			ms, err := memberships.ListForUser(ctx, sess.UserID)
			if err == nil && len(ms) > 0 {
				ctx = database.WithOrgID(ctx, ms[0].OrganizationID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func tokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(SessionCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
