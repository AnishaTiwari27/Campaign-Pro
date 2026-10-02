package main

import (
	"log/slog"
	"net/http"

	"campaigntrackerpro/platform/httpx"

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

func newRouter(health http.HandlerFunc, loadUser httpx.UserLoader, adminEmail string,
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
		r.Use(httpx.AuthMiddleware(loadUser, adminEmail, logger))
		r.Get("/health", health)
		for _, m := range modules {
			m.Routes(r)
		}
	})

	return r
}
