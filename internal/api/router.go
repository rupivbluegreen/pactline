package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/rupivbluegreen/pactline/internal/api/handlers"
	authmw "github.com/rupivbluegreen/pactline/internal/api/middleware"
	"github.com/rupivbluegreen/pactline/internal/database"
)

type Deps struct {
	Sessions    *database.SessionRepo
	Memberships *database.MembershipRepo

	Auth      *handlers.AuthHandlers
	Me        *handlers.MeHandler
	Orgs      *handlers.OrgsHandlers
	Contracts *handlers.ContractsHandler
}

func Router(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", handlers.Health)
	r.Get("/hello", handlers.Hello)

	if d.Auth != nil {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/magic-link/request", d.Auth.RequestMagicLink)
			r.Post("/magic-link/verify", d.Auth.Verify)
			r.Post("/logout", d.Auth.Logout)
		})
	}

	if d.Sessions != nil && d.Memberships != nil {
		r.Group(func(r chi.Router) {
			r.Use(authmw.Auth(d.Sessions, d.Memberships))
			if d.Me != nil {
				r.Get("/me", d.Me.Me)
			}
			if d.Orgs != nil {
				r.Post("/organizations", d.Orgs.Create)
			}
			if d.Contracts != nil {
				r.Get("/contracts", d.Contracts.List)
			}
		})
	}

	return r
}
