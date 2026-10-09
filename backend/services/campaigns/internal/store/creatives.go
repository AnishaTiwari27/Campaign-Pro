package store

import (
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/campaigns"
	"context"

	"campaigntrackerpro/db/gen"
)

func (s *Store) ListCreativesByCampaign(ctx context.Context, campaignID string) ([]campaigns.Creative, error) {
	cs, err := s.db.Queries.ListCreativesByCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	return toDomainCreatives(cs), nil
}

func (s *Store) CountCreativesByCampaign(ctx context.Context, campaignID string) (int64, error) {
	return s.db.Queries.CountCreativesByCampaign(ctx, campaignID)
}

func (s *Store) CreateCreative(ctx context.Context, c campaigns.Creative) (campaigns.Creative, error) {
	created, err := s.db.Queries.CreateCreative(ctx, gen.CreateCreativeParams{
		CampaignID:    c.CampaignID,
		Headline:      c.Headline,
		Kind:          gen.CreativeKindT(c.Kind),
		DurationLabel: c.DurationLabel,
		Reach:         c.Reach,
		Ctr:           c.CTR,
		Language:      database.TextParam(c.Language),
		HookType:      database.HookTypeParam(c.HookType),
		Claim:         database.TextParam(c.Claim),
		Festival:      database.TextParam(c.Festival),
		AnalyzedAt:    database.TimestampParam(c.AnalyzedAt),
	})
	if err != nil {
		return campaigns.Creative{}, err
	}
	return toDomainCreative(created), nil
}

// FestivalBreakdown groups analysed creatives by festival.
func (s *Store) FestivalBreakdown(ctx context.Context) ([]campaigns.FestivalStat, error) {
	rows, err := s.db.Queries.FestivalReach(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]campaigns.FestivalStat, 0, len(rows))
	for _, r := range rows {
		out = append(out, campaigns.FestivalStat{
			Festival:  r.Festival,
			Creatives: int(r.Creatives),
			AvgReach:  r.AvgReach,
			AvgCTR:    r.AvgCtr,
		})
	}
	return out, nil
}
