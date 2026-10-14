// Package module is ingestion's factory.
package module

import (
	"log/slog"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/ingest"
	"campaigntrackerpro/services/ingest/internal/api"
	"campaigntrackerpro/services/ingest/internal/service"
	"campaigntrackerpro/services/ingest/internal/store"

	"github.com/go-chi/chi/v5"
)

type Module struct{ handler *api.Handlers }

// New builds the service with the providers it should offer. Google Ads
// is always listed even when unconfigured, so the UI can show it as
// available-but-not-set-up rather than pretending it does not exist.
func New(db *database.DB, google ingest.GoogleAdsConfig, logger *slog.Logger) *Module {
	svc := service.New(store.New(db), logger)
	providers := map[string]ingest.Provider{
		"google-ads": ingest.GoogleAdsProvider{Config: google},
	}
	return &Module{handler: api.New(svc, providers, logger)}
}

func (m *Module) Routes(r chi.Router) { m.handler.Routes(r) }
