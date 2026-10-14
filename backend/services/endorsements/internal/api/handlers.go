// Package api is the endorsements transport. It owns the /endorsements
// URL space and is read-only: the record is a published fact, and there
// is no writer for it in the product.
package api

import (
	"log/slog"
	"net/http"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/endorsements"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Handlers { return &Handlers{logger: logger} }

func (h *Handlers) Routes(r chi.Router) {
	r.Route("/endorsements", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
	})
}

// No client scoping here, and that is deliberate rather than an
// omission: this is published information about public figures, not the
// agency's book, so there is nothing in it that one account may see and
// another may not.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": endorsements.All()})
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	p, ok := endorsements.Find(chi.URLParam(r, "id"))
	if !ok {
		httpx.WriteNotFound(w, "no such person in the public record")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}
