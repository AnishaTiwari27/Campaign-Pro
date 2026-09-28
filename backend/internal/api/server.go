// Package api wires HTTP routes to the store and implements every endpoint
// in docs/API_CONTRACT.md.
package api

import (
	"net/http"
	"time"

	"campaigntrackerpro/internal/store"
)

// Server holds the dependencies every handler needs. Today is injected
// rather than read from time.Now() at request time: the seed dataset is
// generated relative to a fixed anchor date (see internal/data.GenerateCampaigns),
// so "today" has to match that anchor for date-range filters and campaign
// status ("Live"/"Completed") to behave the way the original frontend mock
// did. Point it at time.Now() once campaigns are ingested from real ad
// platforms instead of the seed generator.
type Server struct {
	Store store.Store
	Today time.Time
}

func NewRouter(s *Server, allowedOrigin string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/v1/meta", s.handleMeta)
	mux.HandleFunc("GET /api/v1/campaigns", s.handleListCampaigns)
	mux.HandleFunc("GET /api/v1/campaigns/export", s.handleExportCampaigns)
	mux.HandleFunc("GET /api/v1/campaigns/{id}", s.handleGetCampaign)
	mux.HandleFunc("GET /api/v1/kpis", s.handleKPIs)
	mux.HandleFunc("GET /api/v1/trend", s.handleTrend)
	mux.HandleFunc("GET /api/v1/territories", s.handleTerritories)
	mux.HandleFunc("GET /api/v1/benchmark", s.handleBenchmark)

	return requestLogger(cors(allowedOrigin)(mux))
}
