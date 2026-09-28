// Package api is catalog-service's HTTP surface: the public /api/v1/meta
// and /api/v1/subjects endpoints, plus an /internal/subjects/lookup route
// only other services are expected to call.
package api

import (
	"net/http"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/catalog/internal/store"
)

type Server struct {
	Store *store.Store
	Cache *platform.Cache // may be nil — every handler tolerates that
}

func NewRouter(s *Server, allowedOrigin string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/v1/meta", s.handleMeta)
	mux.HandleFunc("GET /api/v1/subjects", s.handleListSubjects)
	mux.HandleFunc("GET /internal/subjects/lookup", s.handleLookupSubject)

	handler := platform.RequestLogger(platform.CORS(allowedOrigin)(mux))
	return platform.Traced("catalog-service", handler)
}
