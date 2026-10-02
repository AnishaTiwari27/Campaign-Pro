package store

import (
	"campaigntrackerpro/platform/database"
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
		CreatedAt:       database.TimeOf(c.CreatedAt),
		UpdatedAt:       database.TimeOf(c.UpdatedAt),
	}
}

func (s *Store) ListCreators(ctx context.Context) ([]creators.Creator, error) {
	cs, err := s.db.Queries.ListCreators(ctx)
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
	c, err := s.db.Queries.GetCreator(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return creators.Creator{}, database.ErrNotFound
		}
		return creators.Creator{}, err
	}
	return toDomainCreator(c), nil
}

func (s *Store) CreateCreator(ctx context.Context, c creators.Creator) (creators.Creator, error) {
	created, err := s.db.Queries.CreateCreator(ctx, gen.CreateCreatorParams{
		ID: c.ID, Name: c.Name, Role: c.Role, Initials: c.Initials,
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
