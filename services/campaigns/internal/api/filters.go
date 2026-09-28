package api

import (
	"net/http"
	"strconv"
	"time"

	"campaigntrackerpro/services/campaigns/internal/store"
)

func parseFilter(r *http.Request, today time.Time) store.Filter {
	q := r.URL.Query()
	days := 0
	if raw := q.Get("days"); raw != "" {
		if d, err := strconv.Atoi(raw); err == nil && d > 0 {
			days = d
		}
	}
	return store.Filter{
		Category:    q.Get("category"),
		Region:      q.Get("region"),
		AdType:      q.Get("adType"),
		SubjectType: q.Get("subjectType"),
		Subject:     q.Get("subject"),
		Query:       q.Get("q"),
		Days:        days,
		Today:       today,
	}
}

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
