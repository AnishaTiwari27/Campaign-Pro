package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"campaigntrackerpro/platform/httpx"
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

// CandidateStatuses a reviewer can set directly.
const (
	StatusRejected = "rejected"
)

// Reject marks a candidate as not worth pursuing. It sticks through
// re-discovery, or next week's run would put them back in the queue.
func (c *Creators) Reject(ctx context.Context, id string) error {
	return c.Store.SetCandidateStatus(ctx, id, StatusRejected)
}

// Promote turns a reviewed candidate into a roster creator.
//
// The roster is what every benchmark is computed from, so this is the
// only door into it and a person is always on the other side. The
// reviewer supplies category, region and languages because none of
// them appear in a search result and none are guessable from a channel
// title — and all three are axes the medians slice on, so a guess here
// would skew figures a buyer reads.
func (c *Creators) Promote(ctx context.Context, id string, p creators.Promotion) (creators.Creator, error) {
	cand, err := c.Store.GetCandidate(ctx, id)
	if err != nil {
		return creators.Creator{}, notFound(err)
	}
	if cand.Status == "accepted" {
		return creators.Creator{}, fmt.Errorf("%w: this candidate has already been promoted", httpx.ErrValidation)
	}
	if err := p.Validate(cand); err != nil {
		return creators.Creator{}, fmt.Errorf("%w: %s", httpx.ErrValidation, err)
	}

	followers := p.FollowerCount(cand)
	role := p.Role
	if strings.TrimSpace(role) == "" {
		role = "Creator"
	}
	created, err := c.Store.CreateCreator(ctx, creators.Creator{
		ID:       creators.SlugOf(cand.Name),
		Name:     cand.Name,
		Role:     role,
		Initials: creators.InitialsOf(cand.Name),
		Category: p.Category,
		Region:   p.Region,
		// Derived, never supplied: the tier bands are the product's
		// own and a reviewer picking one by hand would be able to put
		// a creator in a band their follower count contradicts.
		Tier:            creators.TierFor(followers),
		Followers:       followers,
		PrimaryPlatform: platformLabel(cand.Platform),
		Languages:       p.Languages,
	})
	if err != nil {
		return creators.Creator{}, err
	}
	// Linked after the creator exists, so a failure leaves the
	// candidate in the queue rather than marking it done with nothing
	// to show for it.
	if err := c.Store.LinkPromoted(ctx, id, created.ID); err != nil {
		return created, err
	}
	return created, nil
}

func platformLabel(p string) string {
	if p == "youtube" {
		return "YouTube"
	}
	return p
}
