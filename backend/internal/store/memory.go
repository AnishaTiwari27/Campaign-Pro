// Package store holds campaign data behind a small interface so the MVP's
// in-memory slice can be swapped for a Postgres-backed implementation
// (Phase 3, second cut) without touching the API layer.
package store

import (
	"strings"
	"time"

	"campaigntrackerpro/internal/models"
)

// Store is what the API handlers depend on. Filter/BrandOf/etc. are enough
// to serve every route in docs/API_CONTRACT.md.
type Store interface {
	All() []models.Campaign
	ByID(id int) (models.Campaign, bool)
	Filter(f Filter) []models.Campaign
}

// Filter mirrors the shared query params in docs/API_CONTRACT.md.
type Filter struct {
	Category  string
	Territory string
	AdType    string
	Brand     string
	Query     string
	Days      int // 0 means "no range limit"
	Today     time.Time
}

type memoryStore struct {
	campaigns []models.Campaign // sorted by Start desc, set once at construction
}

func NewMemoryStore(campaigns []models.Campaign) Store {
	return &memoryStore{campaigns: campaigns}
}

func (s *memoryStore) All() []models.Campaign {
	out := make([]models.Campaign, len(s.campaigns))
	copy(out, s.campaigns)
	return out
}

func (s *memoryStore) ByID(id int) (models.Campaign, bool) {
	for _, c := range s.campaigns {
		if c.ID == id {
			return c, true
		}
	}
	return models.Campaign{}, false
}

// Filter applies every non-empty field of f, preserving the store's
// start-date-descending order. Zero-value fields ("" / 0) are "no filter",
// matching the frontend's "All" sentinel.
func (s *memoryStore) Filter(f Filter) []models.Campaign {
	var out []models.Campaign
	q := strings.ToLower(strings.TrimSpace(f.Query))
	for _, c := range s.campaigns {
		if f.Category != "" && f.Category != "All" && c.Category != f.Category {
			continue
		}
		if f.Territory != "" && f.Territory != "All" && c.Territory != f.Territory {
			continue
		}
		if f.AdType != "" && f.AdType != "All" && c.AdType != f.AdType {
			continue
		}
		if f.Brand != "" && c.Brand != f.Brand {
			continue
		}
		if f.Days > 0 {
			today := f.Today
			if today.IsZero() {
				today = time.Now()
			}
			daysSinceStart := int(today.Sub(c.Start).Hours() / 24)
			if daysSinceStart > f.Days {
				continue
			}
		}
		if q != "" && !strings.Contains(strings.ToLower(c.Brand), q) {
			continue
		}
		out = append(out, c)
	}
	return out
}
