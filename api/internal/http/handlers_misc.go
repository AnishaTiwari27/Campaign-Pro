package httpapi

import "net/http"

func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results, err := h.Campaigns.Search(r.Context(), q)
	if err != nil {
		writeServiceError(w, h.Logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": searchDTOs(results)})
}

type meDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	CanApprove bool   `json:"canApprove"`
}

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "no user in context"})
		return
	}
	writeJSON(w, http.StatusOK, meDTO{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, CanApprove: user.CanApprove})
}

// Health reports each data source's status for the Settings screen's "Data
// sources" panel. There's one real dependency (Postgres); every logical
// source reflects that same ping so the panel never claims a health signal
// this backend doesn't actually have.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	if err := h.Campaigns.Store.Pool.Ping(r.Context()); err != nil {
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
	writeJSON(w, code, map[string]any{"status": status, "sources": sources})
}
