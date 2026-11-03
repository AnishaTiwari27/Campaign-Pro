package service

import (
	"context"
	"errors"
	"log/slog"

	"campaigntrackerpro/services/creators"
)

// ChannelSearch is the slice of the YouTube client discovery needs.
type ChannelSearch interface {
	SearchChannels(ctx context.Context, query string, limit int) ([]string, error)
	Channels(ctx context.Context, ids []string) ([]creators.Channel, error)
}

// DiscoverResult is what one discovery run found.
type DiscoverResult struct {
	Queries int
	Found   int
	Saved   int
	// Exhausted says the run stopped on the quota rather than
	// finishing the brief list. Worth surfacing: the briefs that never
	// ran are not briefs that found nothing.
	Exhausted bool
}

// Discover runs each brief through YouTube and records the channels as
// candidates for review.
//
// Searches first, then one batched lookup for every id found. Doing it
// per brief would cost one unit per brief instead of one per fifty
// channels — the same data at a hundred times the quota — and the
// searches are already the expensive half.
func (c *Creators) Discover(ctx context.Context, yt ChannelSearch, orgID string, briefs []string, logger *slog.Logger) (DiscoverResult, error) {
	var res DiscoverResult

	// Which brief found a channel is kept because a reviewer needs to
	// see what it was meant to match. First brief wins: a channel
	// found twice was found for both, and the first is the one whose
	// results they will be reading.
	foundFor := map[string]string{}
	var ids []string

	for _, brief := range briefs {
		res.Queries++
		got, err := yt.SearchChannels(ctx, brief, 50)
		if err != nil {
			if errors.Is(err, creators.ErrQuotaExhausted) {
				res.Exhausted = true
				logger.Warn("stopped on the daily quota; the remaining briefs did not run",
					"ran", res.Queries-1, "of", len(briefs))
				break
			}
			logger.Warn("search failed", "brief", brief, "err", err)
			continue
		}
		for _, id := range got {
			if _, seen := foundFor[id]; !seen {
				foundFor[id] = brief
				ids = append(ids, id)
			}
		}
	}

	if len(ids) == 0 {
		return res, nil
	}

	channels, err := yt.Channels(ctx, ids)
	// Channels already paid for come back alongside a quota error, so
	// they are saved before the error is returned.
	if err != nil && !errors.Is(err, creators.ErrQuotaExhausted) {
		return res, err
	}
	if errors.Is(err, creators.ErrQuotaExhausted) {
		res.Exhausted = true
	}
	res.Found = len(channels)

	cands := make([]creators.Candidate, 0, len(channels))
	for _, ch := range channels {
		cands = append(cands, creators.CandidateFrom(ch, foundFor[ch.ID]))
	}
	saved, err := c.Store.SaveCandidates(ctx, orgID, cands)
	res.Saved = saved
	return res, err
}

// NewCandidates are the channels still waiting on a decision.
func (c *Creators) NewCandidates(ctx context.Context, limit int) ([]creators.Candidate, error) {
	return c.Store.NewCandidates(ctx, limit)
}
