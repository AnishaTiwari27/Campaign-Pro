package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"campaigntrackerpro/db/gen"
	"campaigntrackerpro/services/creators"

	"github.com/jackc/pgx/v5/pgtype"
)

func newID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return "bp_" + hex.EncodeToString(b)
}

func tsOrNull(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// SaveBrandPosts writes the evidence from one scan.
//
// orgID is passed rather than read from the context because the scan
// runs under a system context that spans tenants: a read may use the
// bypass, but a row has to be written into a real organisation, and
// the only right one is the creator's own.
func (s *Store) SaveBrandPosts(ctx context.Context, orgID, creatorID string, posts []creators.BrandPostEvidence) error {
	for _, p := range posts {
		err := s.db.Queries.UpsertBrandPost(ctx, gen.UpsertBrandPostParams{
			ID:        newID(),
			OrgID:     orgID,
			CreatorID: creatorID,
			Brand:     p.Brand,
			Permalink: p.Permalink,
			PostedAt:  tsOrNull(p.PostedAt),
			MatchedOn: p.MatchedOn,
			Method:    p.Method,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// MarkScanned stamps when this creator's media was last read. It is
// what separates "looked and found nothing" from "never looked", so it
// is written even when the scan found no brand work at all.
func (s *Store) MarkScanned(ctx context.Context, creatorID string, at time.Time) error {
	sys, org := orgParams(ctx)
	return s.db.Queries.MarkMediaScanned(ctx, gen.MarkMediaScannedParams{
		ID: creatorID, ScannedAt: tsOrNull(at), OrgSystem: sys, OrgID: org,
	})
}

// RecordSnapshot writes today's figures for one creator.
//
// engagement_rate is left null where the audit could not compute one.
// A zero would be read as a dead account, which is a different and
// much worse claim than "not enough posts to say".
func (s *Store) RecordSnapshot(ctx context.Context, orgID, creatorID string, followers, mediaCount int64, a creators.Audit, on time.Time) error {
	var rate pgtype.Float8
	if a.HasRate {
		rate = pgtype.Float8{Float64: a.EngagementRate, Valid: true}
	}
	return s.db.Queries.RecordFollowerSnapshot(ctx, gen.RecordFollowerSnapshotParams{
		OrgID:          orgID,
		CreatorID:      creatorID,
		CapturedOn:     pgtype.Date{Time: on, Valid: true},
		Followers:      followers,
		MediaCount:     mediaCount,
		EngagementRate: rate,
		PostsScanned:   int32(a.Posts),
	})
}

// BrandPostCounts is how many disclosed brand posts each creator has,
// for the roster view. Creators with none are absent rather than zero —
// the caller distinguishes those from unscanned ones by the scan stamp.
func (s *Store) BrandPostCounts(ctx context.Context) (map[string]int, error) {
	sys, org := orgParams(ctx)
	rows, err := s.db.Queries.BrandPostCountsByCreator(ctx,
		gen.BrandPostCountsByCreatorParams{OrgSystem: sys, OrgID: org})
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		out[r.CreatorID] = int(r.Posts)
	}
	return out, nil
}

// FollowerHistory is the growth series for one creator.
func (s *Store) FollowerHistory(ctx context.Context, creatorID string) ([]creators.FollowerPoint, error) {
	sys, org := orgParams(ctx)
	rows, err := s.db.Queries.ListFollowerSnapshots(ctx,
		gen.ListFollowerSnapshotsParams{CreatorID: creatorID, OrgSystem: sys, OrgID: org})
	if err != nil {
		return nil, err
	}
	out := make([]creators.FollowerPoint, 0, len(rows))
	for _, r := range rows {
		p := creators.FollowerPoint{
			On:        r.CapturedOn.Time,
			Followers: r.Followers,
			Posts:     int(r.PostsScanned),
		}
		if r.EngagementRate.Valid {
			p.EngagementRate, p.HasRate = r.EngagementRate.Float64, true
		}
		out = append(out, p)
	}
	return out, nil
}

// SignalsByCreator is the platform-derived half of the roster, in one
// pass: the scan stamp from the creator row, the brand-post count, and
// the figures from the most recent snapshot.
//
// A creator absent from the map has never been scanned. A creator
// present with no brand posts has been scanned and had none found —
// the distinction the whole Signals type exists for.
func (s *Store) SignalsByCreator(ctx context.Context, scannedAt map[string]time.Time) (map[string]creators.Signals, error) {
	counts, err := s.BrandPostCounts(ctx)
	if err != nil {
		return nil, err
	}
	sys, org := orgParams(ctx)
	snaps, err := s.db.Queries.LatestSnapshotPerCreator(ctx,
		gen.LatestSnapshotPerCreatorParams{OrgSystem: sys, OrgID: org})
	if err != nil {
		return nil, err
	}

	out := make(map[string]creators.Signals, len(scannedAt))
	for id, at := range scannedAt {
		out[id] = creators.Signals{Scanned: true, ScanAt: at, BrandPosts: counts[id]}
	}
	for _, r := range snaps {
		sig := out[r.CreatorID]
		// A snapshot exists only because a scan wrote it, so it is
		// evidence of a scan in its own right — belt and braces
		// against a creator row whose stamp was lost.
		sig.Scanned = true
		sig.PostsScanned = int(r.PostsScanned)
		if r.EngagementRate.Valid {
			sig.EngagementRate, sig.HasRate = r.EngagementRate.Float64, true
		}
		if sig.BrandPosts == 0 {
			sig.BrandPosts = counts[r.CreatorID]
		}
		out[r.CreatorID] = sig
	}
	return out, nil
}
