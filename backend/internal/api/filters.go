package api

import (
	"net/http"
	"strconv"

	"campaigntrackerpro/internal/store"
)

// parseFilter reads the shared list/aggregate query params documented in
// docs/API_CONTRACT.md. today is injected by the server (see server.go)
// rather than read from the clock here, so handlers stay testable.
func (s *Server) parseFilter(r *http.Request) store.Filter {
	q := r.URL.Query()
	days := 0
	if raw := q.Get("days"); raw != "" {
		if d, err := strconv.Atoi(raw); err == nil && d > 0 {
			days = d
		}
	}
	return store.Filter{
		Category:  q.Get("category"),
		Territory: q.Get("territory"),
		AdType:    q.Get("adType"),
		Brand:     q.Get("brand"),
		Query:     q.Get("q"),
		Days:      days,
		Today:     s.Today,
	}
}

// parsePagination reads page/limit, clamping limit to [1, 100] and page to >= 1.
func parsePagination(r *http.Request) (page, limit int) {
	page, limit = 1, 25
	q := r.URL.Query()
	if raw := q.Get("page"); raw != "" {
		if p, err := strconv.Atoi(raw); err == nil && p > 0 {
			page = p
		}
	}
	if raw := q.Get("limit"); raw != "" {
		if l, err := strconv.Atoi(raw); err == nil && l > 0 {
			limit = l
			if limit > 100 {
				limit = 100
			}
		}
	}
	return page, limit
}
