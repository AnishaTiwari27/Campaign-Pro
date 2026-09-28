// Creative-level tracking — a campaign can run multiple creatives (real
// ad platforms do). Deliberately additive, not decomposed: a campaign's
// own Reach/Spend (models.go) are never recalculated from its creatives'
// sum — see db/init/02_campaigns.sql's comment on campaigns.creatives for
// why reconciling the two isn't worth the added risk this pass.
package models

import "time"

// CreativeTypes is the fixed vocabulary campaigns.creatives.creative_type
// is CHECK-constrained to — validated here too so a bad value is a clean
// 400, not a raw DB constraint violation surfacing as a 500.
var creativeTypes = map[string]bool{"image": true, "video": true, "carousel": true, "text": true}

func CreativeTypeValid(t string) bool {
	return creativeTypes[t]
}

type Creative struct {
	ID           int64
	CampaignID   int64
	Headline     string
	CreativeType string
	Reach        int64
	Spend        int64 // rupees
	CreatedAt    time.Time
}

type CreativeJSON struct {
	ID           int64  `json:"id"`
	CampaignID   int64  `json:"campaignId"`
	Headline     string `json:"headline"`
	CreativeType string `json:"creativeType"`
	Reach        int64  `json:"reach"`
	Spend        int64  `json:"spend"`
	CreatedAt    string `json:"createdAt"`
}

func (c Creative) AsJSON() CreativeJSON {
	return CreativeJSON{
		ID: c.ID, CampaignID: c.CampaignID, Headline: c.Headline, CreativeType: c.CreativeType,
		Reach: c.Reach, Spend: c.Spend, CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}
}

// CreateCreativeRequest is the POST /api/v1/campaigns/{id}/creatives body.
type CreateCreativeRequest struct {
	Headline     string `json:"headline"`
	CreativeType string `json:"creativeType"`
	Reach        int64  `json:"reach"`
	Spend        int64  `json:"spend"`
}
