// Package module is the reports service's factory, including the
// background worker that runs scheduled reports on their cadence.
package module

import (
	"context"
	"log/slog"
	"time"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/mail"
	"campaigntrackerpro/services/reports/internal/api"
	"campaigntrackerpro/services/reports/internal/service"
	"campaigntrackerpro/services/reports/internal/store"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	svc     *service.Reports
	worker  *service.Worker
	handler *api.Handlers
	store   *store.Store
}

// New takes the campaign reader and the anomaly detector as dependencies:
// reports reads campaigns to build a CSV and reacts to new flags, but owns
// neither.
func New(db *database.DB, campaigns service.CampaignReader, detector service.Detector,
	mailer mail.Mailer, logger *slog.Logger, tick time.Duration) *Module {
	st := store.New(db)
	svc := service.NewReports(st, campaigns, mailer)
	return &Module{
		svc:     svc,
		worker:  service.NewWorker(detector, svc, logger, tick),
		handler: api.New(svc, logger),
		store:   st,
	}
}

func (m *Module) Routes(r chi.Router) { m.handler.Routes(r) }

// Run starts the scheduler; it returns when ctx is cancelled.
func (m *Module) Run(ctx context.Context) { m.worker.Run(ctx) }

// Store exposes report-owned writes to the fixture seeder.
func (m *Module) Store() *store.Store { return m.store }
