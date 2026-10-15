// Package api is the insights transport. It owns /insights.
package api

import (
	"log/slog"
	"net/http"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/insights/internal/service"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	svc    *service.Insights
	logger *slog.Logger
}

func New(svc *service.Insights, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

func (h *Handlers) Routes(r chi.Router) {
	r.Get("/insights/campaigns/{id}", h.Explain)
}

// Explain is scoped by the campaign read underneath it: a campaign the
// caller cannot see is not in All(), so this answers "no such campaign"
// for another tenant's id exactly as every other read does.
func (h *Handlers) Explain(w http.ResponseWriter, r *http.Request) {
	exp, err := h.svc.Explain(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, exp)
}
