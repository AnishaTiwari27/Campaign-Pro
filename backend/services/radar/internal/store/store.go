// Package store persists what the radar has seen.
package store

import (
	"context"
	"regexp"
	"strings"
	"time"

	"campaigntrackerpro/db/gen"
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/radar"

	"github.com/jackc/pgx/v5/pgtype"
)

type Store struct{ db *database.DB }

func New(db *database.DB) *Store { return &Store{db: db} }

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func (s *Store) Upsert(ctx context.Context, orgID string, l radar.Launch) error {
	return s.db.Queries.UpsertLaunch(ctx, gen.UpsertLaunchParams{
		// Deterministic, so re-polling keeps the same row and anything
		// recorded against it survives.
		ID:         l.Source + ":" + l.ExternalID,
		OrgID:      orgID,
		Source:     l.Source,
		ExternalID: l.ExternalID,
		Name:       l.Name,
		Tagline:    l.Tagline,
		Url:        l.URL,
		Category:   l.Category,
		LaunchedAt: ts(l.LaunchedAt),
		Traction:   l.Traction,
		HasSite:    l.HasSite,
		Readiness:  string(l.Readiness),
		Basis:      l.Basis,
	})
}

func (s *Store) List(ctx context.Context) ([]radar.Launch, error) {
	o := scope.OrgFrom(ctx)
	rows, err := s.db.Queries.ListLaunches(ctx, gen.ListLaunchesParams{OrgSystem: o.System, OrgID: o.ID})
	if err != nil {
		return nil, err
	}
	out := make([]radar.Launch, 0, len(rows))
	for _, r := range rows {
		l := radar.Launch{
			ID: r.ID, Source: r.Source, ExternalID: r.ExternalID,
			Name: r.Name, Tagline: r.Tagline, URL: r.Url, Category: r.Category,
			LaunchedAt: r.LaunchedAt.Time, FirstSeen: r.FirstSeenAt.Time,
			Traction: r.Traction, HasSite: r.HasSite,
			Readiness: radar.Readiness(r.Readiness), Basis: r.Basis,
		}
		if r.FirstCampaignAt.Valid {
			t := r.FirstCampaignAt.Time
			l.FirstCampaignAt = &t
		}
		if r.DaysToMarket.Valid {
			d := r.DaysToMarket.Int32
			l.DaysToMarket = &d
		}
		out = append(out, l)
	}
	return out, nil
}

// Pending is the launches still waiting on a first campaign — the ones
// whose estimate has not been settled either way yet.
type Pending struct {
	ID         string
	Name       string
	LaunchedAt time.Time
}

func (s *Store) Pending(ctx context.Context) ([]Pending, error) {
	o := scope.OrgFrom(ctx)
	rows, err := s.db.Queries.ListLaunchesAwaitingOutcome(ctx,
		gen.ListLaunchesAwaitingOutcomeParams{OrgSystem: o.System, OrgID: o.ID})
	if err != nil {
		return nil, err
	}
	out := make([]Pending, 0, len(rows))
	for _, r := range rows {
		out = append(out, Pending{ID: r.ID, Name: r.Name, LaunchedAt: r.LaunchedAt.Time})
	}
	return out, nil
}

// RecordOutcome writes what actually happened: when the first campaign
// for this brand appeared, and how long after launch that was. This is
// the measurement the banding has no access to today, which is exactly
// why it is collected.
func (s *Store) RecordOutcome(ctx context.Context, id string, firstCampaignAt time.Time, days int32) error {
	o := scope.OrgFrom(ctx)
	return s.db.Queries.RecordLaunchOutcome(ctx, gen.RecordLaunchOutcomeParams{
		ID:              id,
		FirstCampaignAt: ts(firstCampaignAt),
		DaysToMarket:    pgtype.Int4{Int32: days, Valid: true},
		OrgSystem:       o.System,
		OrgID:           o.ID,
	})
}

// MatchesBrand reports whether a campaign name refers to a launch.
//
// Deliberately strict. The alternative is fuzzy matching, which pairs
// "Theorem" with "Theorem Health" and quietly records an outcome
// against the wrong brand — corrupting the only measurement this
// feature has, invisibly. A missed match costs a row that stays
// pending; a wrong one costs the data.
//
// The match is on whole words, not substrings. Plain containment
// looked strict enough until a test tried it: the launch "Ito" matched
// "Capitol Hill Campaign", because "ito" is inside "capitol". Short
// brand names are common and that failure is silent.
func MatchesBrand(campaignName, launchName string) bool {
	c := strings.ToLower(strings.TrimSpace(campaignName))
	l := strings.ToLower(strings.TrimSpace(launchName))
	if l == "" || len(l) < 3 {
		return false
	}
	re, err := regexp.Compile(`\b` + regexp.QuoteMeta(l) + `\b`)
	if err != nil {
		return false
	}
	return re.MatchString(c)
}
