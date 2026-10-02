// Package module is the analytics service's factory. Analytics owns no
// tables — it reads across campaigns and creators — so it is constructed
// entirely from capabilities its siblings publish.
package module

import (
	"log/slog"
	"net/http"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/analytics/internal/api"
	"campaigntrackerpro/services/analytics/internal/service"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	svc     *service.Analytics
	handler *api.Handlers
}

func New(db *database.DB, campaigns service.CampaignReader, creators service.CreatorReader, logger *slog.Logger) *Module {
	svc := service.New(campaigns, creators)
	return &Module{svc: svc, handler: api.New(svc, db, logger)}
}

func (m *Module) Routes(r chi.Router) { m.handler.Routes(r) }

// Health is mounted at the root as well as under /api, so the readiness
// probe doesn't sit behind auth.
func (m *Module) Health() http.HandlerFunc { return m.handler.HealthHandler() }
