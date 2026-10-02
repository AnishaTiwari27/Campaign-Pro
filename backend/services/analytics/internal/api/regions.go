package api

import (
	"net/http"

	"campaigntrackerpro/platform/httpx"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) ListRegions(w http.ResponseWriter, r *http.Request) {
	regions, err := h.svc.Regions(r.Context())
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": regionDTOs(regions)})
}

func (h *Handlers) GetRegion(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := h.svc.RegionDetail(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, regionDetailDTO(detail))
}
