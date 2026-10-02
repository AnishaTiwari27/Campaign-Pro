package api

import (
	"log/slog"
	"net/http"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/analytics/internal/service"

	"github.com/go-chi/chi/v5"
)

// Handlers is the analytics transport. It owns the cross-cutting read
// endpoints: the overview, benchmarks, regions, search, health and me.
type Handlers struct {
	svc    *service.Analytics
	db     *database.DB
	logger *slog.Logger
}

func New(svc *service.Analytics, db *database.DB, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, db: db, logger: logger}
}

func (h *Handlers) Routes(r chi.Router) {
	r.Get("/search", h.Search)
	r.Get("/overview", h.GetOverview)
	r.Get("/benchmark", h.GetBenchmark)
	r.Route("/regions", func(r chi.Router) {
		r.Get("/", h.ListRegions)
		r.Get("/{id}", h.GetRegion)
	})
}

// Health is mounted outside /api too, so it stays a plain handler.
func (h *Handlers) HealthHandler() http.HandlerFunc { return h.Health }
