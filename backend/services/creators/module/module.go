// Package module is the creators service's factory. Everything it builds
// lives under internal/, so the only way to reach this service is through
// the capabilities returned here.
package module

import (
	"context"
	"log/slog"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/creators"
	"campaigntrackerpro/services/creators/internal/api"
	"campaigntrackerpro/services/creators/internal/service"
	"campaigntrackerpro/services/creators/internal/store"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	svc     *service.Creators
	handler *api.Handlers
	store   *store.Store
}

// New takes the campaign reader as a dependency rather than constructing
// one: creators needs campaign data but must not own campaign access.
func New(db *database.DB, campaigns service.CampaignReader, logger *slog.Logger) *Module {
	st := store.New(db)
	svc := service.NewCreators(st, campaigns)
	return &Module{svc: svc, handler: api.New(svc, logger), store: st}
}

func (m *Module) Routes(r chi.Router) { m.handler.Routes(r) }

// Store exposes creator-owned writes to the fixture seeder.
func (m *Module) Store() *store.Store { return m.store }

// SyncFollowers refreshes follower counts from Instagram, replacing
// the seeded press-article figures with what the platform reports.
//
// The provider is passed in rather than built here so the composition
// root keeps hold of the credentials, as it does for every other
// outside service.
func (m *Module) SyncFollowers(ctx context.Context, ig service.ProfileLookup, logger *slog.Logger) (creators.SyncResult, error) {
	return m.svc.SyncFollowers(ctx, ig, logger)
}
