// Package module is the insights service's factory.
package module

import (
	"log/slog"

	"campaigntrackerpro/platform/llm"
	"campaigntrackerpro/services/insights/internal/api"
	"campaigntrackerpro/services/insights/internal/service"

	"github.com/go-chi/chi/v5"
)

type Module struct{ handler *api.Handlers }

// New takes the model client as a dependency rather than building one,
// so a deployment with no API key passes nil and the service answers
// with its assembled explanation instead of failing.
func New(campaigns service.CampaignReader, model llm.Client, logger *slog.Logger) *Module {
	return &Module{handler: api.New(service.New(campaigns, model, logger), logger)}
}

func (m *Module) Routes(r chi.Router) { m.handler.Routes(r) }
