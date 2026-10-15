// Package api is the radar's transport. It owns /radar.
package api

import (
	"log/slog"
	"net/http"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/radar/internal/service"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	svc    *service.Radar
	logger *slog.Logger
}

func New(svc *service.Radar, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

func (h *Handlers) Routes(r chi.Router) {
	r.Get("/radar/launches", h.List)
}

// List returns what the radar has seen, newest first.
//
// Agency-only: this is prospecting data — who to approach before they
// start spending — and a client account has no business seeing which
// brands their agency is about to pitch.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	v, _ := httpx.UserFromContext(r.Context())
	if c, ok := v.(interface{ IsClient() bool }); ok && c.IsClient() {
		httpx.WriteForbidden(w, "the launch radar is agency prospecting data")
		return
	}
	launches, err := h.svc.List(r.Context())
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": launches})
}
