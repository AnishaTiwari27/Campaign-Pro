// Package api is analytics-service's HTTP surface: KPIs, trend, region
// breakdown, and competitor benchmarking — every endpoint stateless,
// computed on demand from campaigns-service's data, cached briefly in Redis.
package api

import (
	"net/http"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/analytics/internal/campaignsclient"
)

const cacheTTLSeconds = 30 // safety-net TTL; the real invalidation is campaign.created — see events.go

type Server struct {
	Campaigns *campaignsclient.Client
	Cache     *platform.Cache // nil-safe
}

func NewRouter(s *Server, allowedOrigin string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/v1/kpis", s.handleKPIs)
	mux.HandleFunc("GET /api/v1/trend", s.handleTrend)
	mux.HandleFunc("GET /api/v1/regions", s.handleRegions)
	mux.HandleFunc("GET /api/v1/benchmark", s.handleBenchmark)

	handler := platform.RequestLogger(platform.CORS(allowedOrigin)(mux))
	return platform.Traced("analytics-service", handler)
}
