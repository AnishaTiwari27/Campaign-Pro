package service

import (
	"context"

	"campaigntrackerpro/internal/domain"
	"campaigntrackerpro/internal/store"
)

type Creators struct {
	Store     *store.Store
	Campaigns *Campaigns
}

func NewCreators(s *store.Store, c *Campaigns) *Creators {
	return &Creators{Store: s, Campaigns: c}
}

// tierMedians computes, for each tier, the median campaign reach across
// every creator in that tier. This is the baseline a creator is judged
// against — comparing a nano creator's reach to a mega creator's says
// nothing except that one has a bigger audience.
func tierMedians(creators []domain.Creator, campaignsByCreator map[string][]domain.Campaign) map[domain.Tier]float64 {
	reachByTier := map[domain.Tier][]float64{}
	for _, cr := range creators {
		for _, camp := range campaignsByCreator[cr.ID] {
			if !camp.IsRunning() {
				continue
			}
			reachByTier[cr.Tier] = append(reachByTier[cr.Tier], camp.Reach)
		}
	}
	out := map[domain.Tier]float64{}
	for tier, reaches := range reachByTier {
		out[tier] = domain.Median(reaches)
	}
	return out
}

func performanceFor(cr domain.Creator, camps []domain.Campaign, tierMedian float64) domain.CreatorPerformance {
	perf := domain.CreatorPerformance{Creator: cr, TierMedian: tierMedian}

	reaches := make([]float64, 0, len(camps))
	for _, c := range camps {
		perf.Campaigns++
		if c.Status == domain.StatusLive {
			perf.LiveCampaigns++
		}
		if c.IsFlagged() {
			perf.Flagged++
		}
		perf.TotalReach += c.Reach
		perf.TotalSpend += c.Spend
		if c.IsRunning() {
			reaches = append(reaches, c.Reach)
		}
	}

	if len(reaches) > 0 {
		var sum float64
		for _, r := range reaches {
			sum += r
		}
		perf.AvgReach = sum / float64(len(reaches))
	}

	perf.TierIndex = domain.Index(perf.AvgReach, tierMedian)
	perf.CostPerLakh = domain.CostPerLakhReach(perf.TotalSpend, perf.TotalReach)
	perf.Consistency = domain.Consistency(reaches)
	perf.ConsistencyN = len(reaches)
	perf.AudienceReachPct = domain.AudienceReachPct(perf.AvgReach, cr.Followers)
	return perf
}

// List returns the full roster ranked by how well each creator performs
// for their own tier.
func (c *Creators) List(ctx context.Context) ([]domain.CreatorPerformance, error) {
	creators, err := c.Store.ListCreators(ctx)
	if err != nil {
		return nil, err
	}
	all, err := c.Store.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}

	byCreator := map[string][]domain.Campaign{}
	for _, camp := range all {
		if camp.CreatorID != "" {
			byCreator[camp.CreatorID] = append(byCreator[camp.CreatorID], camp)
		}
	}

	medians := tierMedians(creators, byCreator)
	out := make([]domain.CreatorPerformance, 0, len(creators))
	for _, cr := range creators {
		out = append(out, performanceFor(cr, byCreator[cr.ID], medians[cr.Tier]))
	}
	domain.RankCreators(out)
	return out, nil
}

type CreatorDetail struct {
	domain.CreatorPerformance
	CampaignRows []CampaignRow
	// LanguageBreakdown is reach by creative language across this creator's
	// campaigns — the multi-language view global tools don't offer.
	LanguageBreakdown []LanguageReach
	TierLabel         string
	TierPeers         int
}

type LanguageReach struct {
	Language string
	Reach    float64
	Count    int
}

func (c *Creators) Get(ctx context.Context, id string) (CreatorDetail, error) {
	cr, err := c.Store.GetCreator(ctx, id)
	if err != nil {
		return CreatorDetail{}, err
	}

	creators, err := c.Store.ListCreators(ctx)
	if err != nil {
		return CreatorDetail{}, err
	}
	all, err := c.Store.ListCampaigns(ctx)
	if err != nil {
		return CreatorDetail{}, err
	}

	byCreator := map[string][]domain.Campaign{}
	for _, camp := range all {
		if camp.CreatorID != "" {
			byCreator[camp.CreatorID] = append(byCreator[camp.CreatorID], camp)
		}
	}
	medians := tierMedians(creators, byCreator)

	camps := byCreator[cr.ID]
	perf := performanceFor(cr, camps, medians[cr.Tier])

	rowsByID, _, _, _ := c.Campaigns.enrich(ctx, all)
	rows := make([]CampaignRow, 0, len(camps))
	for _, camp := range camps {
		rows = append(rows, rowsByID[camp.ID])
	}
	sortRowsDesc(rows, func(r CampaignRow) float64 { return r.Reach })

	// Language mix, weighted by the reach of the campaign each creative ran on.
	langReach := map[string]*LanguageReach{}
	var langOrder []string
	for _, camp := range camps {
		creatives, err := c.Store.ListCreativesByCampaign(ctx, camp.ID)
		if err != nil {
			return CreatorDetail{}, err
		}
		for _, cre := range creatives {
			if cre.Language == "" {
				continue
			}
			lr, ok := langReach[cre.Language]
			if !ok {
				lr = &LanguageReach{Language: cre.Language}
				langReach[cre.Language] = lr
				langOrder = append(langOrder, cre.Language)
			}
			lr.Reach += cre.Reach
			lr.Count++
		}
	}
	breakdown := make([]LanguageReach, 0, len(langOrder))
	for _, l := range langOrder {
		breakdown = append(breakdown, *langReach[l])
	}

	peers := 0
	for _, other := range creators {
		if other.Tier == cr.Tier {
			peers++
		}
	}

	return CreatorDetail{
		CreatorPerformance: perf,
		CampaignRows:       rows,
		LanguageBreakdown:  breakdown,
		TierLabel:          domain.TierLabel(cr.Tier),
		TierPeers:          peers,
	}, nil
}
