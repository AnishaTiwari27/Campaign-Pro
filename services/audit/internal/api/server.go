// Package api is audit-service's HTTP surface: the admin-only read path
// over rows written by the NATS subscriber in cmd/server/main.go.
package api

import (
	"context"
	"net/http"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/audit/internal/models"
)

// AuditStore is exactly what the handlers below call — sized this way
// (Interface Segregation) so handler logic is unit-testable against a
// fake, no real Postgres; same pattern every other service's Store
// interface already follows.
type AuditStore interface {
	Ping(ctx context.Context) error
	List(ctx context.Context, page, limit int, campaignID *int64) ([]models.Entry, int, error)
}

type Server struct {
	Store AuditStore
}

func NewRouter(s *Server, allowedOrigin string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/v1/audit-log", s.handleListAuditLog)

	handler := platform.RequestLogger(platform.CORS(allowedOrigin)(mux))
	return platform.Traced("audit-service", handler)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.Ping(r.Context()); err != nil {
		platform.Upstream(w, "database unreachable")
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
