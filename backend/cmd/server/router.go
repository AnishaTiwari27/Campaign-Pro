package main

import (
	"log/slog"
	"net/http"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// mountable is what every service module offers the router: its own URL
// space. The composition root never names an individual path — services
// own their routes.
type mountable interface {
	Routes(r chi.Router)
}

// publicMountable serves routes that must work without a session — login
// being the obvious one, since requiring a session to sign in is circular.
type publicMountable interface {
	PublicRoutes(r chi.Router)
	Authenticator() httpx.SessionAuthenticator
}

func newRouter(health http.HandlerFunc, auth publicMountable,
	logger *slog.Logger, modules ...mountable) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(httpx.RequestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:5174"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}))

	// Unauthenticated, so a readiness probe doesn't need credentials.
	r.Get("/health", health)

	r.Route("/api", func(r chi.Router) {
		// Login sits outside the guard; everything else sits behind it.
		auth.PublicRoutes(r)

		r.Group(func(r chi.Router) {
			r.Use(httpx.RequireSession(auth.Authenticator(), logger))
			r.Get("/health", health)
			for _, m := range modules {
				m.Routes(r)
			}
		})
	})

	// Everything that is not the API is the single-page app, served from the
	// same origin so the SameSite=Lax session cookie is sent with every
	// request. This is registered last and as the fallback, so it can never
	// shadow a route a service owns. /api and /health are passed through as
	// prefixes the SPA must not answer for, keeping their 404s JSON.
	r.NotFound(web.Handler("/api", "/health").ServeHTTP)

	return r
}
