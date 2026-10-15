// Package module is the radar's factory.
package module

import (
	"context"
	"log/slog"
	"time"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/radar"
	"campaigntrackerpro/services/radar/internal/api"
	"campaigntrackerpro/services/radar/internal/service"
	"campaigntrackerpro/services/radar/internal/store"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	svc     *service.Radar
	handler *api.Handlers
	logger  *slog.Logger
}

// New wires the sources the radar reads. Hacker News needs no
// credential and so is always on; sources that fit this market better
// want keys and are added here as they are obtained.
func New(db *database.DB, campaigns service.CampaignReader, logger *slog.Logger) *Module {
	sources := []radar.Source{radar.HackerNews{}}
	svc := service.New(store.New(db), campaigns, sources, logger)
	return &Module{svc: svc, handler: api.New(svc, logger), logger: logger}
}

func (m *Module) Routes(r chi.Router) { m.handler.Routes(r) }

// pollEvery is how often sources are re-read. Launches are a daily
// rhythm, not a live feed, and the sources rate-limit.
const pollEvery = 6 * time.Hour

// Run polls the sources and settles outcomes on a slow loop.
func (m *Module) Run(ctx context.Context) {
	ticker := time.NewTicker(pollEvery)
	defer ticker.Stop()
	m.once(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.once(ctx)
		}
	}
}

func (m *Module) once(ctx context.Context) {
	// Launches are written into the default organisation: they are
	// public information about the market rather than one tenant's
	// data, and every agency sees the same launch list.
	orgCtx := scope.SystemOrgIn(ctx, scope.DefaultOrg)
	if _, err := m.svc.Poll(orgCtx, scope.DefaultOrg, time.Now().AddDate(0, 0, -30)); err != nil {
		m.logger.Error("radar poll failed", "err", err)
	}
	if n, err := m.svc.SettleOutcomes(orgCtx); err != nil {
		m.logger.Error("radar outcome sweep failed", "err", err)
	} else if n > 0 {
		m.logger.Info("radar: outcomes recorded", "count", n)
	}
}
