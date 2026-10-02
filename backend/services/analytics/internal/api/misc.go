package api

import (
	"net/http"

	"campaigntrackerpro/platform/httpx"
)

func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results, err := h.svc.Search(r.Context(), q)
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": searchDTOs(results)})
}

// Health reports each data source's status for the Settings screen's "Data
// sources" panel. There's one real dependency (Postgres); every logical
// source reflects that same ping so the panel never claims a health signal
// this backend doesn't actually have.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	if err := h.db.Pool.Ping(r.Context()); err != nil {
		status = "unhealthy"
	}
	sources := []map[string]string{
		{"name": "Database", "status": status},
		{"name": "Campaigns API", "status": status},
		{"name": "Reports API", "status": status},
		{"name": "Search", "status": status},
	}
	code := http.StatusOK
	if status != "healthy" {
		code = http.StatusServiceUnavailable
	}
	httpx.WriteJSON(w, code, map[string]any{"status": status, "sources": sources})
}
