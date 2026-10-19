package store

import (
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/creators"
	"context"
	"errors"

	"campaigntrackerpro/db/gen"

	"github.com/jackc/pgx/v5"
)

func toDomainCreator(c gen.Creator) creators.Creator {
	return creators.Creator{
		ID:              c.ID,
		Name:            c.Name,
		Role:            c.Role,
		Initials:        c.Initials,
		Category:        c.Category,
		Region:          c.Region,
		Tier:            creators.Tier(c.Tier),
		Followers:       c.Followers,
		PrimaryPlatform: c.PrimaryPlatform,
		Languages:       c.Languages,
		ImageURL:        database.TextOrEmpty(c.ImageUrl),
		CreatedAt:       database.TimeOf(c.CreatedAt),
		UpdatedAt:       database.TimeOf(c.UpdatedAt),
	}
}

func (s *Store) ListCreators(ctx context.Context) ([]creators.Creator, error) {
	sys, org := orgParams(ctx)
	cs, err := s.db.Queries.ListCreators(ctx, gen.ListCreatorsParams{OrgSystem: sys, OrgID: org})
	if err != nil {
		return nil, err
	}
	out := make([]creators.Creator, len(cs))
	for i, c := range cs {
		out[i] = toDomainCreator(c)
	}
	return out, nil
}

func (s *Store) GetCreator(ctx context.Context, id string) (creators.Creator, error) {
	sys, org := orgParams(ctx)
	c, err := s.db.Queries.GetCreator(ctx, gen.GetCreatorParams{ID: id, OrgSystem: sys, OrgID: org})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return creators.Creator{}, database.ErrNotFound
		}
		return creators.Creator{}, err
	}
	return toDomainCreator(c), nil
}

func (s *Store) CreateCreator(ctx context.Context, c creators.Creator) (creators.Creator, error) {
	_, org := orgParams(ctx)
	created, err := s.db.Queries.CreateCreator(ctx, gen.CreateCreatorParams{
		OrgID: org,
		ID:    c.ID, Name: c.Name, Role: c.Role, Initials: c.Initials,
		Category: c.Category, Region: c.Region, Tier: gen.CreatorTierT(c.Tier),
		Followers: c.Followers, PrimaryPlatform: c.PrimaryPlatform, Languages: c.Languages,
	})
	if err != nil {
		return creators.Creator{}, err
	}
	return toDomainCreator(created), nil
}

// All satisfies the reader analytics declares so creators are findable in
// the command palette.
func (s *Store) All(ctx context.Context) ([]creators.Creator, error) {
	return s.ListCreators(ctx)
}

// SetPhoto attaches a creator's own photograph, or clears it when the url
// is empty.
// ForFollowerSync is the roster the Instagram sync walks: those with
// a handle established, least recently synced first.
func (s *Store) ForFollowerSync(ctx context.Context) ([]creators.SyncTarget, error) {
	sys, org := orgParams(ctx)
	rows, err := s.db.Queries.ListCreatorsForFollowerSync(ctx,
		gen.ListCreatorsForFollowerSyncParams{OrgSystem: sys, OrgID: org})
	if err != nil {
		return nil, err
	}
	out := make([]creators.SyncTarget, 0, len(rows))
	for _, r := range rows {
		out = append(out, creators.SyncTarget{
			ID: r.ID, Name: r.Name, InstagramHandle: r.InstagramHandle,
		})
	}
	return out, nil
}

// SetFollowers records a fetched follower count and stamps the time,
// which is what separates a figure fetched from Instagram from the
// seeded one it replaces.
func (s *Store) SetFollowers(ctx context.Context, creatorID string, followers int64) error {
	sys, org := orgParams(ctx)
	return s.db.Queries.SetCreatorFollowers(ctx, gen.SetCreatorFollowersParams{
		ID: creatorID, Followers: followers, OrgSystem: sys, OrgID: org,
	})
}

// SetInstagramHandle records which account the follower sync looks up.
func (s *Store) SetInstagramHandle(ctx context.Context, creatorID, handle string) error {
	sys, org := orgParams(ctx)
	return s.db.Queries.SetCreatorInstagramHandle(ctx, gen.SetCreatorInstagramHandleParams{
		ID: creatorID, InstagramHandle: handle, OrgSystem: sys, OrgID: org,
	})
}

func (s *Store) SetPhoto(ctx context.Context, creatorID, imageURL string) error {
	return s.db.Queries.SetCreatorPhoto(ctx, gen.SetCreatorPhotoParams{ID: creatorID, Column2: imageURL})
}

// orgParams turns the request's tenant into query parameters. Every
// statement touching an org-owned table takes these; platform/arch's
// tenancy test fails the build if one is added that does not.
//
// A system context spans organisations, which is right for the worker,
// the seeder and the CLI. Anything else is confined to one, and an
// unresolved organisation is the empty string — which matches no row, so
// forgetting to set it yields an empty result rather than a leak.
func orgParams(ctx context.Context) (bool, string) {
	o := scope.OrgFrom(ctx)
	return o.System, o.ID
}
