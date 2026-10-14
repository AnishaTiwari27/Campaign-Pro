// Package store is analytics' own persistence: the benchmark history it
// writes and reads. Everything else analytics needs it borrows from
// campaigns through a public contract.
package store

import (
	"context"
	"time"

	"campaigntrackerpro/db/gen"
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/scope"

	"github.com/jackc/pgx/v5/pgtype"
)

type Store struct{ db *database.DB }

func New(db *database.DB) *Store { return &Store{db: db} }

// Snapshot is one category's standing on one day.
type Snapshot struct {
	Category          string
	CapturedOn        time.Time
	MedianReach       float64
	TopReach          float64
	CampaignCount     int32
	MedianCostPerLakh float64
}

func dateParam(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

// Capture writes today's standing for one category in one organisation.
// Re-running it the same day overwrites rather than appending, so the
// worker ticking every thirty seconds leaves one row per day.
func (s *Store) Capture(ctx context.Context, orgID string, snap Snapshot) error {
	return s.db.Queries.UpsertBenchmarkSnapshot(ctx, gen.UpsertBenchmarkSnapshotParams{
		OrgID:             orgID,
		Category:          snap.Category,
		CapturedOn:        dateParam(snap.CapturedOn),
		MedianReach:       snap.MedianReach,
		TopReach:          snap.TopReach,
		CampaignCount:     snap.CampaignCount,
		MedianCostPerLakh: snap.MedianCostPerLakh,
	})
}

// Trend returns every category's series since a date, oldest first.
func (s *Store) Trend(ctx context.Context, since time.Time) ([]Snapshot, error) {
	o := scope.OrgFrom(ctx)
	rows, err := s.db.Queries.ListBenchmarkTrend(ctx, gen.ListBenchmarkTrendParams{
		OrgSystem: o.System,
		OrgID:     o.ID,
		Since:     dateParam(since),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Snapshot, 0, len(rows))
	for _, r := range rows {
		out = append(out, Snapshot{
			Category:          r.Category,
			CapturedOn:        r.CapturedOn.Time,
			MedianReach:       r.MedianReach,
			TopReach:          r.TopReach,
			CampaignCount:     r.CampaignCount,
			MedianCostPerLakh: r.MedianCostPerLakh,
		})
	}
	return out, nil
}

// Organizations lists every tenant, so the capture job can walk them.
// The worker is a system caller; this is the one place that needs to
// know the set of organisations rather than being handed one.
func (s *Store) Organizations(ctx context.Context) ([]string, error) {
	return s.db.Queries.ListOrganizationIDs(ctx)
}
