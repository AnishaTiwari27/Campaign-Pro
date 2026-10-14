// Package module is the analytics service's factory. Analytics reads
// across campaigns and creators through the capabilities its siblings
// publish, and owns exactly one table of its own: the benchmark history,
// which has to be written down as it happens because reach and spend are
// overwritten on the campaign row as it runs.
package module

import (
	"context"
	"log/slog"
	"net/http"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/analytics/internal/api"
	"campaigntrackerpro/services/analytics/internal/service"
	"campaigntrackerpro/services/analytics/internal/store"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	svc     *service.Analytics
	handler *api.Handlers
	logger  *slog.Logger
}

func New(db *database.DB, campaigns service.CampaignReader, creators service.CreatorReader, logger *slog.Logger) *Module {
	svc := service.New(campaigns, creators).WithTrends(store.New(db))
	return &Module{svc: svc, handler: api.New(svc, db, logger), logger: logger}
}

// CaptureBenchmarks records today's standing for every organisation.
// Driven by the report worker's tick rather than a scheduler of its own:
// the write is idempotent per day, so the only cost of a frequent tick
// is a rewrite of a handful of rows.
func (m *Module) CaptureBenchmarks(ctx context.Context) {
	m.svc.CaptureToday(ctx, m.logger)
}

func (m *Module) Routes(r chi.Router) { m.handler.Routes(r) }

// Health is mounted at the root as well as under /api, so the readiness
// probe doesn't sit behind auth.
func (m *Module) Health() http.HandlerFunc { return m.handler.HealthHandler() }
