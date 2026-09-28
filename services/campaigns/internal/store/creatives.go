// Creative-level tracking — see models/creatives.go and
// db/init/02_campaigns.sql's comment on campaigns.creatives.
package store

import (
	"context"

	"campaigntrackerpro/services/campaigns/internal/models"
)

// ListCreatives returns every creative under campaignID, oldest first.
func (s *Store) ListCreatives(ctx context.Context, campaignID int64) ([]models.Creative, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, campaign_id, headline, creative_type, reach, spend_paise, created_at
		 FROM campaigns.creatives WHERE campaign_id = $1 ORDER BY created_at ASC`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Creative
	for rows.Next() {
		var c models.Creative
		var spendPaise int64
		if err := rows.Scan(&c.ID, &c.CampaignID, &c.Headline, &c.CreativeType, &c.Reach, &spendPaise, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.Spend = spendPaise / 100
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateCreative inserts one creative under campaignID. Does not touch
// the parent campaign's own reach/spend — see models/creatives.go's
// package doc comment for why.
func (s *Store) CreateCreative(ctx context.Context, campaignID int64, headline, creativeType string, reach, spend int64) (models.Creative, error) {
	c := models.Creative{CampaignID: campaignID, Headline: headline, CreativeType: creativeType, Reach: reach, Spend: spend}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO campaigns.creatives (campaign_id, headline, creative_type, reach, spend_paise)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		campaignID, headline, creativeType, reach, spend*100,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return models.Creative{}, err
	}
	return c, nil
}
