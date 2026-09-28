package api

import (
	"log/slog"
	"net/http"
	"time"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/analytics/internal/aggregate"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// cached wraps the "fetch pre-aggregated data, reshape, respond" pattern
// shared by every aggregate endpoint: check Redis, and on a miss, call
// compute(), cache the result, and write it. compute() gets the raw
// querystring so it can forward filters to campaigns-service unchanged.
func (s *Server) cached(w http.ResponseWriter, r *http.Request, keyPrefix string, compute func() (any, error)) {
	ctx := r.Context()
	key := keyPrefix + ":" + r.URL.RawQuery

	if s.Cache != nil {
		var cached any
		if s.Cache.Get(ctx, key, &cached) {
			platform.WriteJSON(w, http.StatusOK, cached)
			return
		}
	}

	result, err := compute()
	if err != nil {
		slog.Error("compute aggregate", "endpoint", keyPrefix, "err", err)
		platform.Upstream(w, "failed to compute: "+err.Error())
		return
	}
	if s.Cache != nil {
		s.Cache.Set(ctx, key, result, cacheTTLSeconds*time.Second)
	}
	platform.WriteJSON(w, http.StatusOK, result)
}

func (s *Server) handleKPIs(w http.ResponseWriter, r *http.Request) {
	s.cached(w, r, "analytics:kpis", func() (any, error) {
		raw, err := s.Campaigns.KPIs(r.Context(), r.URL.RawQuery)
		if err != nil {
			return nil, err
		}
		return aggregate.KPIs(raw.Totals, raw.Weekly), nil
	})
}

func (s *Server) handleTrend(w http.ResponseWriter, r *http.Request) {
	s.cached(w, r, "analytics:trend", func() (any, error) {
		buckets, err := s.Campaigns.Trend(r.Context(), r.URL.RawQuery)
		if err != nil {
			return nil, err
		}
		return aggregate.Trend(buckets), nil
	})
}

func (s *Server) handleRegions(w http.ResponseWriter, r *http.Request) {
	s.cached(w, r, "analytics:regions", func() (any, error) {
		counts, err := s.Campaigns.Regions(r.Context(), r.URL.RawQuery)
		if err != nil {
			return nil, err
		}
		return aggregate.Regions(counts), nil
	})
}

func (s *Server) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	s.cached(w, r, "analytics:benchmark", func() (any, error) {
		rows, err := s.Campaigns.Benchmark(r.Context(), r.URL.RawQuery)
		if err != nil {
			return nil, err
		}
		return aggregate.Benchmark(rows), nil
	})
}
