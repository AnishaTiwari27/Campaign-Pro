package store

import (
	"context"

	"campaigntrackerpro/db/gen"
	"campaigntrackerpro/services/creators"

	"github.com/jackc/pgx/v5/pgtype"
)

// SaveCandidates records what discovery turned up.
//
// orgID is explicit for the same reason the scan's writes are: this
// runs from the CLI under a system context, and a row has to land in a
// real organisation.
func (s *Store) SaveCandidates(ctx context.Context, orgID string, cs []creators.Candidate) (int, error) {
	var n int
	for _, c := range cs {
		var followers pgtype.Int8
		if c.HasFollowers {
			followers = pgtype.Int8{Int64: c.Followers, Valid: true}
		}
		err := s.db.Queries.UpsertCandidate(ctx, gen.UpsertCandidateParams{
			ID:          newCandidateID(),
			OrgID:       orgID,
			Platform:    c.Platform,
			ExternalID:  c.ExternalID,
			Handle:      c.Handle,
			Name:        c.Name,
			Description: c.Description,
			Followers:   followers,
			Videos:      c.Videos,
			Views:       c.Views,
			FoundFor:    c.FoundFor,
		})
		if err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func newCandidateID() string { return "cand_" + randomHex(12) }

// NewCandidates are those still waiting on a review.
func (s *Store) NewCandidates(ctx context.Context, limit int) ([]creators.Candidate, error) {
	if limit <= 0 {
		limit = 200
	}
	sys, org := orgParams(ctx)
	rows, err := s.db.Queries.ListNewCandidates(ctx, gen.ListNewCandidatesParams{
		OrgSystem: sys, OrgID: org, RowLimit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]creators.Candidate, 0, len(rows))
	for _, r := range rows {
		c := creators.Candidate{
			ID: r.ID, Platform: r.Platform, ExternalID: r.ExternalID,
			Handle: r.Handle, Name: r.Name, Description: r.Description,
			Videos: r.Videos, Views: r.Views, FoundFor: r.FoundFor,
			Status: r.Status,
		}
		if r.Followers.Valid {
			c.Followers, c.HasFollowers = r.Followers.Int64, true
		}
		out = append(out, c)
	}
	return out, nil
}

// SetCandidateStatus records a reviewer's decision. A rejection sticks
// through re-discovery, or the same search would ask the same question
// every week.
func (s *Store) SetCandidateStatus(ctx context.Context, id, status string) error {
	sys, org := orgParams(ctx)
	return s.db.Queries.SetCandidateStatus(ctx, gen.SetCandidateStatusParams{
		ID: id, Status: status, OrgSystem: sys, OrgID: org,
	})
}
