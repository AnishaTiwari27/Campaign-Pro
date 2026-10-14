// Package module is the endorsements service's factory. It takes no
// database, because the public record is not the agency's data and must
// not become writable by the agency's machinery.
package module

import (
	"log/slog"

	"campaigntrackerpro/services/endorsements/internal/api"

	"github.com/go-chi/chi/v5"
)

type Module struct{ handler *api.Handlers }

func New(logger *slog.Logger) *Module { return &Module{handler: api.New(logger)} }

func (m *Module) Routes(r chi.Router) { m.handler.Routes(r) }
