// Package service polls the sources and settles outcomes.
package service

import (
	"context"
	"log/slog"
	"time"

	"campaigntrackerpro/services/campaigns"
	"campaigntrackerpro/services/radar"
	"campaigntrackerpro/services/radar/internal/store"
)

// CampaignReader is how the radar finds out that a launch has started
// advertising. Narrow on purpose.
type CampaignReader interface {
	All(ctx context.Context) ([]campaigns.Campaign, error)
}

type Radar struct {
	store     *store.Store
	campaigns CampaignReader
	sources   []radar.Source
	logger    *slog.Logger
}

func New(st *store.Store, c CampaignReader, sources []radar.Source, logger *slog.Logger) *Radar {
	return &Radar{store: st, campaigns: c, sources: sources, logger: logger}
}

func (r *Radar) List(ctx context.Context) ([]radar.Launch, error) {
	return r.store.List(ctx)
}

// Poll fetches from every source and stores what it finds.
func (r *Radar) Poll(ctx context.Context, orgID string, since time.Time) (int, error) {
	var stored int
	for _, src := range r.sources {
		launches, err := src.Fetch(since)
		if err != nil {
			// One source being down is not a reason to lose the others.
			r.logger.Error("radar: source failed", "source", src.Name(), "err", err)
			continue
		}
		for _, l := range launches {
			if err := r.store.Upsert(ctx, orgID, l); err != nil {
				r.logger.Error("radar: could not store launch", "source", src.Name(), "id", l.ExternalID, "err", err)
				continue
			}
			stored++
		}
		r.logger.Info("radar: polled", "source", src.Name(), "launches", len(launches))
	}
	return stored, nil
}

// SettleOutcomes looks for launches whose brand now has a campaign,
// and records how long that took.
//
// This is the part that makes the estimate improvable. Nothing here
// reads the outcome back yet — there is no history to fit to — but
// without collecting it there never will be, and the question "is the
// banding any good?" stays unanswerable.
func (r *Radar) SettleOutcomes(ctx context.Context) (int, error) {
	pending, err := r.store.Pending(ctx)
	if err != nil {
		return 0, err
	}
	if len(pending) == 0 {
		return 0, nil
	}
	all, err := r.campaigns.All(ctx)
	if err != nil {
		return 0, err
	}

	var settled int
	for _, p := range pending {
		for _, c := range all {
			if !store.MatchesBrand(c.Name, p.Name) {
				continue
			}
			// CreatedAt is when this system first saw the campaign,
			// which is the closest thing available to when the brand
			// started advertising. Not the same thing — a campaign may
			// have been running before it was ingested — and that
			// caveat belongs with the number when anyone comes to use
			// it.
			first := c.CreatedAt
			days := int32(first.Sub(p.LaunchedAt).Hours() / 24)
			if days < 0 {
				days = 0
			}
			if err := r.store.RecordOutcome(ctx, p.ID, first, days); err != nil {
				r.logger.Error("radar: could not record outcome", "launch", p.ID, "err", err)
				break
			}
			r.logger.Info("radar: launch reached market", "launch", p.Name, "campaign", c.Name, "days", days)
			settled++
			break
		}
	}
	return settled, nil
}
