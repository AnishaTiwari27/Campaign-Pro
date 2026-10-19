package service

import (
	"context"
	"log/slog"

	"campaigntrackerpro/services/creators"
)

// ProfileLookup is what the follower sync needs of Instagram.
// An interface so the sync can be tested without a token, and so a
// different source is one implementation rather than a rewrite.
type ProfileLookup interface {
	Configured() bool
	Lookup(ctx context.Context, handle string) (creators.Profile, error)
}

// SyncFollowers refreshes follower counts from Instagram.
//
// The seeded figures are press-article numbers: right once, with no
// way of saying when. This replaces them with what Instagram reports
// and stamps the time, so the app can tell the two apart.
//
// A profile that does not look like the creator it was recorded
// against is counted and skipped, never written. A handle is typed by
// hand, and the wrong one does not merely show a wrong number — it
// attaches another person's audience to a real named individual, and
// that figure then flows into their tier, their index, and every
// comparison the product draws against them.
func (c *Creators) SyncFollowers(ctx context.Context, ig ProfileLookup, logger *slog.Logger) (creators.SyncResult, error) {
	var out creators.SyncResult
	if !ig.Configured() {
		return out, creators.ErrInstagramNotConfigured
	}

	targets, err := c.Store.ForFollowerSync(ctx)
	if err != nil {
		return out, err
	}

	for _, t := range targets {
		out.Checked++
		profile, err := ig.Lookup(ctx, t.InstagramHandle)
		if err != nil {
			// One unreachable account must not end the run: the rest
			// of the roster is still worth refreshing.
			logger.Warn("follower sync: lookup failed",
				"creator", t.ID, "handle", t.InstagramHandle, "err", err)
			out.Failed = append(out.Failed, t.ID)
			continue
		}
		if !creators.SamePerson(t.Name, profile) {
			logger.Error("follower sync: handle does not match the creator — not written",
				"creator", t.ID, "expected", t.Name,
				"handle", profile.Username, "profileName", profile.Name)
			out.Mismatched = append(out.Mismatched, t.ID)
			continue
		}
		if err := c.Store.SetFollowers(ctx, t.ID, profile.Followers); err != nil {
			out.Failed = append(out.Failed, t.ID)
			continue
		}
		out.Updated++
	}
	return out, nil
}
