// Package module is the subscription service's factory.
package module

import (
	"context"
	"log/slog"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/billing"
	"campaigntrackerpro/services/billing/internal/api"
	"campaigntrackerpro/services/billing/internal/service"
	"campaigntrackerpro/services/billing/internal/store"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	svc     *service.Billing
	handler *api.Handlers
}

func New(db *database.DB, cfg billing.RazorpayConfig, logger *slog.Logger) *Module {
	st := store.New(db)
	svc := service.New(st, billing.NewRazorpay(cfg), logger)
	return &Module{svc: svc, handler: api.New(svc, logger)}
}

// Routes mount behind the session guard. They are deliberately not
// behind the subscription gate: an organisation that cannot use the
// app is precisely the one that needs to reach these.
func (m *Module) Routes(r chi.Router) { m.handler.Routes(r) }

// WebhookRoutes mount outside the session guard, since a provider has
// no session. The signature authenticates them instead.
func (m *Module) WebhookRoutes(r chi.Router) { m.handler.WebhookRoutes(r) }

// Gate is what the composition root mounts as middleware: one question,
// asked once per request, after the tenant is known.
func (m *Module) Gate() func(ctx context.Context, orgID string) (billing.Subscription, bool, error) {
	return m.svc.Allows
}
