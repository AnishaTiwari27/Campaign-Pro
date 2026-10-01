package store

import (
	"context"
	"errors"

	"campaigntrackerpro/internal/db/gen"
	"campaigntrackerpro/internal/domain"

	"github.com/jackc/pgx/v5"
)

func toDomainCreator(c gen.Creator) domain.Creator {
	return domain.Creator{
		ID:              c.ID,
		Name:            c.Name,
		Role:            c.Role,
		Initials:        c.Initials,
		Category:        c.Category,
		Region:          c.Region,
		Tier:            domain.Tier(c.Tier),
		Followers:       c.Followers,
		PrimaryPlatform: c.PrimaryPlatform,
		Languages:       c.Languages,
		CreatedAt:       timeOf(c.CreatedAt),
		UpdatedAt:       timeOf(c.UpdatedAt),
	}
}

func (s *Store) ListCreators(ctx context.Context) ([]domain.Creator, error) {
	cs, err := s.Queries.ListCreators(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Creator, len(cs))
	for i, c := range cs {
		out[i] = toDomainCreator(c)
	}
	return out, nil
}

func (s *Store) GetCreator(ctx context.Context, id string) (domain.Creator, error) {
	c, err := s.Queries.GetCreator(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Creator{}, ErrNotFound
		}
		return domain.Creator{}, err
	}
	return toDomainCreator(c), nil
}

func (s *Store) CreateCreator(ctx context.Context, c domain.Creator) (domain.Creator, error) {
	created, err := s.Queries.CreateCreator(ctx, gen.CreateCreatorParams{
		ID: c.ID, Name: c.Name, Role: c.Role, Initials: c.Initials,
		Category: c.Category, Region: c.Region, Tier: gen.CreatorTierT(c.Tier),
		Followers: c.Followers, PrimaryPlatform: c.PrimaryPlatform, Languages: c.Languages,
	})
	if err != nil {
		return domain.Creator{}, err
	}
	return toDomainCreator(created), nil
}

func (s *Store) ListCampaignsByCreator(ctx context.Context, creatorID string) ([]domain.Campaign, error) {
	cs, err := s.Queries.ListCampaignsByCreator(ctx, textParam(creatorID))
	if err != nil {
		return nil, err
	}
	return toDomainCampaigns(cs), nil
}

func (s *Store) SetCampaignCreator(ctx context.Context, campaignID, creatorID string) error {
	return s.Queries.SetCampaignCreator(ctx, gen.SetCampaignCreatorParams{
		ID: campaignID, CreatorID: textParam(creatorID),
	})
}
