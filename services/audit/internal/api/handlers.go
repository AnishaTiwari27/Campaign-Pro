package api

import (
	"log/slog"
	"net/http"
	"strconv"

	"campaigntrackerpro/platform"
)

// handleListAuditLog backs GET /api/v1/audit-log — admin-only, paginated,
// optionally filtered to one campaign via ?campaignId=.
func (s *Server) handleListAuditLog(w http.ResponseWriter, r *http.Request) {
	if !platform.RequireRole(w, r, "only admins can view the audit log", platform.RoleAdmin) {
		return
	}

	page, limit := parsePagination(r)
	var campaignID *int64
	if v := r.URL.Query().Get("campaignId"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			platform.BadRequest(w, "campaignId must be an integer")
			return
		}
		campaignID = &id
	}

	entries, total, err := s.Store.List(r.Context(), page, limit, campaignID)
	if err != nil {
		slog.Error("list audit log", "err", err)
		platform.Internal(w, "failed to list audit log")
		return
	}

	out := make([]any, len(entries))
	for i, e := range entries {
		out[i] = e.AsJSON()
	}
	platform.WriteJSON(w, http.StatusOK, map[string]any{"data": out, "page": page, "limit": limit, "total": total})
}

// parsePagination mirrors campaigns-service's own page/limit parsing
// (default page 1, limit 25, capped at 100) — same defaults, same shared
// convention, kept local since it's a two-line helper not worth a shared
// package for one call site per service.
func parsePagination(r *http.Request) (page, limit int) {
	page, limit = 1, 25
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	return
}
