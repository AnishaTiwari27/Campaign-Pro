package store

import (
	"context"

	"campaigntrackerpro/internal/db/gen"
	"campaigntrackerpro/internal/domain"
)

func (s *Store) ListCreativesByCampaign(ctx context.Context, campaignID string) ([]domain.Creative, error) {
	cs, err := s.Queries.ListCreativesByCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	return toDomainCreatives(cs), nil
}

func (s *Store) CountCreativesByCampaign(ctx context.Context, campaignID string) (int64, error) {
	return s.Queries.CountCreativesByCampaign(ctx, campaignID)
}

func (s *Store) CreateCreative(ctx context.Context, c domain.Creative) (domain.Creative, error) {
	created, err := s.Queries.CreateCreative(ctx, gen.CreateCreativeParams{
		CampaignID:    c.CampaignID,
		Headline:      c.Headline,
		Kind:          gen.CreativeKindT(c.Kind),
		DurationLabel: c.DurationLabel,
		Reach:         c.Reach,
		Ctr:           c.CTR,
		Language:      textParam(c.Language),
		HookType:      hookTypeParam(c.HookType),
		Claim:         textParam(c.Claim),
		Festival:      textParam(c.Festival),
		AnalyzedAt:    timestampParam(c.AnalyzedAt),
	})
	if err != nil {
		return domain.Creative{}, err
	}
	return toDomainCreative(created), nil
}
