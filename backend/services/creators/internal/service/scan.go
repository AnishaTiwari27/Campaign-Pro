package service

import (
	"context"
	"log/slog"
	"time"

	"campaigntrackerpro/services/creators"
)

// MediaReader is the narrow slice of the Instagram client a scan needs.
// An interface so the scan is testable without a network, and so a
// second platform can be added behind it later.
type MediaReader interface {
	LookupMedia(ctx context.Context, handle string, limit int) (creators.Profile, []creators.Post, error)
}

// ScanResult is what one run did.
type ScanResult struct {
	Checked int
	Scanned int
	// Mismatched is the guard firing: the account did not look like
	// the creator it is recorded against. Nothing is written for these
	// — attributing another person's posts, engagement and brand deals
	// to a real named individual is the failure worth stopping the run
	// over, not a warning worth logging past.
	Mismatched int
	Failed     int
	BrandPosts int
}

// ScanCreators reads each creator's recent posts and records what they
// establish: the figures for today's snapshot, and the evidence of any
// disclosed brand work.
//
// Every creator with a handle is scanned, including those whose posts
// show nothing. The scan stamp is the point — without it an unscanned
// creator and a creator with no brand work look identical, and the
// product would report our own backlog as a fact about them.
func (c *Creators) ScanCreators(ctx context.Context, ig MediaReader, brandHandles []string, now time.Time, logger *slog.Logger) (ScanResult, error) {
	targets, err := c.Store.ForFollowerSync(ctx)
	if err != nil {
		return ScanResult{}, err
	}

	var res ScanResult
	for _, t := range targets {
		res.Checked++

		profile, posts, err := ig.LookupMedia(ctx, t.InstagramHandle, creators.MediaScanLimit)
		if err != nil {
			res.Failed++
			logger.Warn("media scan failed",
				"creator", t.ID, "handle", t.InstagramHandle, "err", err)
			continue
		}
		if !creators.SamePerson(t.Name, profile) {
			res.Mismatched++
			logger.Warn("handle does not match the creator it is recorded against — nothing written",
				"creator", t.ID, "recorded", t.Name, "handle", t.InstagramHandle, "profile", profile.Name)
			continue
		}

		stats := make([]creators.PostStats, 0, len(posts))
		var evidence []creators.BrandPostEvidence
		for _, p := range posts {
			stats = append(stats, creators.PostStats{
				Permalink: p.Permalink, Caption: p.Caption, PostedAt: p.PostedAt,
				Likes: p.Likes, Comments: p.Comments, Type: p.Type,
			})
			evidence = append(evidence, creators.EvidenceFrom(p, brandHandles)...)
		}
		audit := creators.AuditPosts(profile.Followers, stats, now)

		if len(evidence) > 0 {
			if err := c.Store.SaveBrandPosts(ctx, t.OrgID, t.ID, evidence); err != nil {
				return res, err
			}
			res.BrandPosts += len(evidence)
		}
		if err := c.Store.RecordSnapshot(ctx, t.OrgID, t.ID, profile.Followers, profile.MediaCount, audit, now); err != nil {
			return res, err
		}
		// Last, and only once the rest has landed: the stamp is what
		// says this creator's figures can be trusted as current, so
		// writing it before the snapshot would claim a scan that had
		// not finished.
		if err := c.Store.MarkScanned(ctx, t.ID, now); err != nil {
			return res, err
		}
		res.Scanned++
	}
	return res, nil
}
