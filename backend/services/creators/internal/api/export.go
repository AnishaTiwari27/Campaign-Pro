package api

import (
	"fmt"
	"net/http"
	"strings"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/creators/internal/service"
	"campaigntrackerpro/services/identity"
)

// ExportCreators serves the roster as a CSV and records who took it.
func (h *Handlers) ExportCreators(w http.ResponseWriter, r *http.Request) {
	var columns []string
	if raw := strings.TrimSpace(r.URL.Query().Get("columns")); raw != "" {
		for _, c := range strings.Split(raw, ",") {
			if c = strings.TrimSpace(c); c != "" {
				columns = append(columns, c)
			}
		}
	}

	v, _ := httpx.UserFromContext(r.Context())
	u, _ := v.(identity.User)

	body, name, err := h.svc.ExportCSV(r.Context(), columns, service.Actor{ID: u.ID, Name: u.Name})
	if err != nil {
		httpx.WriteServiceError(w, h.logger, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	// The roster changes as campaigns land, and a cached export is a
	// stale one someone will act on.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
