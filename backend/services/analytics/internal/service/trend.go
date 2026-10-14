package service

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/analytics/internal/store"
	"campaigntrackerpro/services/campaigns"
	"campaigntrackerpro/services/creators"
)

// TrendPoint is one category on one day, as the API returns it.
type TrendPoint struct {
	Category          string
	Day               time.Time
	MedianReach       float64
	TopReach          float64
	CampaignCount     int32
	MedianCostPerLakh float64
}

// TrendWindow is how far back the benchmark trend reads. Ninety days is
// about as far as a quarterly conversation ever reaches back, and it
// keeps the series small enough to send whole rather than paginate.
const TrendWindow = 90 * 24 * time.Hour

// Trend returns the stored series for the caller's organisation.
func (a *Analytics) Trend(ctx context.Context) ([]TrendPoint, error) {
	if a.trends == nil {
		return nil, nil
	}
	snaps, err := a.trends.Trend(ctx, time.Now().Add(-TrendWindow))
	if err != nil {
		return nil, err
	}
	out := make([]TrendPoint, 0, len(snaps))
	for _, s := range snaps {
		out = append(out, TrendPoint{
			Category: s.Category, Day: s.CapturedOn,
			MedianReach: s.MedianReach, TopReach: s.TopReach,
			CampaignCount: s.CampaignCount, MedianCostPerLakh: s.MedianCostPerLakh,
		})
	}
	return out, nil
}

// CaptureToday writes the current standing of every category, for every
// organisation, into the history.
//
// Called on the worker's tick. It is idempotent by day — the upsert key
// is (org, category, day) — which is what lets a thirty-second tick
// drive a daily series without either a scheduler or a duplicate check.
// The cost of that is a rewrite per tick, which at one row per category
// is far cheaper than the machinery to avoid it.
func (a *Analytics) CaptureToday(ctx context.Context, logger *slog.Logger) {
	if a.trends == nil {
		return
	}
	orgs, err := a.trends.Organizations(ctx)
	if err != nil {
		logger.Error("benchmark capture: cannot list organisations", "err", err)
		return
	}
	today := time.Now().Truncate(24 * time.Hour)

	for _, org := range orgs {
		// Each organisation is captured as itself rather than as a
		// system caller, so the figures written are exactly the ones
		// that tenant would see on its own benchmark page.
		orgCtx := scope.WithOrg(ctx, scope.Org{ID: org})
		all, err := a.campaigns.All(orgCtx)
		if err != nil {
			logger.Error("benchmark capture: cannot read campaigns", "org", org, "err", err)
			continue
		}
		running := campaigns.Running(all)
		if len(running) == 0 {
			continue
		}
		for _, b := range campaigns.CategoryBenchmarks(running) {
			if err := a.trends.Capture(orgCtx, org, store.Snapshot{
				Category:          b.Category,
				CapturedOn:        today,
				MedianReach:       b.MedianReach,
				TopReach:          b.TopReach,
				CampaignCount:     int32(b.N),
				MedianCostPerLakh: medianCostPerLakh(running, b.Category),
			}); err != nil {
				logger.Error("benchmark capture failed", "org", org, "category", b.Category, "err", err)
			}
		}
	}
}

// medianCostPerLakh is the cost side of a category's standing: the
// median rupees spent per lakh of reach across its running campaigns.
//
// Median rather than total spend over total reach, because one very
// large campaign would otherwise define the category's cost and the
// figure would describe that campaign rather than the category.
// Campaigns with no reach yet are excluded — their cost per lakh is
// undefined, and treating it as zero would drag the median down and
// make a category look cheap precisely when it is not yet delivering.
//
// It goes through creators.CostPerLakhReach rather than dividing here,
// because Campaign.Reach is already denominated in lakhs and writing
// the arithmetic out again invites exactly the mistake it did the first
// time: dividing by a hundred thousand a second time, and reporting a
// cost per lakh in the billions.
func medianCostPerLakh(running []campaigns.Campaign, category string) float64 {
	var vals []float64
	for _, c := range running {
		if c.Category != category || c.Reach <= 0 {
			continue
		}
		vals = append(vals, creators.CostPerLakhReach(c.Spend, c.Reach))
	}
	if len(vals) == 0 {
		return 0
	}
	sort.Float64s(vals)
	mid := len(vals) / 2
	if len(vals)%2 == 1 {
		return vals[mid]
	}
	return (vals[mid-1] + vals[mid]) / 2
}
