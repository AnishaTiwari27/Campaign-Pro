// Package module is the campaigns service's factory. It is the only way
// in: everything it constructs lives under the service's internal/, which
// the compiler forbids any sibling service or the composition root from
// importing. Callers get capabilities, never implementations.
package module

import (
	"context"
	"log/slog"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/campaigns"
	"campaigntrackerpro/services/campaigns/internal/api"
	"campaigntrackerpro/services/campaigns/internal/service"
	"campaigntrackerpro/services/campaigns/internal/store"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	svc     *service.Campaigns
	anomaly *service.AnomalyDetector
	handler *api.Handlers
	store   *store.Store
}

// New wires this service's own layers from the shared database.
func New(db *database.DB, logger *slog.Logger) *Module {
	st := store.New(db)
	svc := service.NewCampaigns(st)
	return &Module{
		svc:     svc,
		anomaly: service.NewAnomalyDetector(svc, logger),
		handler: api.New(svc, logger),
		store:   st,
	}
}

// Routes mounts the service's endpoints under the caller's router.
func (m *Module) Routes(r chi.Router) { m.handler.Routes(r) }

// Reader is the read capability sibling services consume. Analytics,
// creators and reports each depend on this narrow interface rather than on
// the concrete service, so none of them can reach past what they need.
type Reader interface {
	List(ctx context.Context, p campaigns.ListParams) (campaigns.ListResult, error)
	Anomalies(ctx context.Context) ([]campaigns.Campaign, error)
	RowOf(ctx context.Context, c campaigns.Campaign) campaigns.CampaignRow
}

// Service hands back the concrete type for siblings that need more than
// Reader. Still unreachable without going through this package.
func (m *Module) Service() *service.Campaigns { return m.svc }

var _ Reader = (*service.Campaigns)(nil)

// Detector re-runs anomaly detection; the reports worker triggers it.
func (m *Module) Detector() *service.AnomalyDetector { return m.anomaly }

// Store exposes campaign-owned writes to the seeder, which loads fixtures
// across every service and is deliberately allowed past the service layer.
func (m *Module) Store() *store.Store { return m.store }

// LoadUser resolves the acting account for platform's auth middleware.
// Campaigns owns the users table, so it supplies the loader.
func (m *Module) LoadUser(ctx context.Context, email string) (any, error) {
	return m.store.GetUserByEmail(ctx, email)
}
