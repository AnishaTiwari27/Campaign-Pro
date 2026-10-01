package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) ListRegions(w http.ResponseWriter, r *http.Request) {
	regions, err := h.Campaigns.Regions(r.Context())
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": regionDTOs(regions)})
}

func (h *Handlers) GetRegion(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := h.Campaigns.RegionDetail(r.Context(), id)
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, regionDetailDTO(detail))
}
