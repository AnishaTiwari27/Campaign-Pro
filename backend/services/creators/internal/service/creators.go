package service

import (
	"errors"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/campaigns"
	"campaigntrackerpro/services/creators"
	"context"
	"sort"

	"campaigntrackerpro/services/creators/internal/store"
)

// ErrNotFound is this service's not-found, aliased to the one the HTTP
// layer maps to a 404.
var ErrNotFound = httpx.ErrNotFound

// Creators depends on a narrow read interface for campaign data rather
// than on the campaigns service itself — it only ever needs to list them
// and enrich a row, so that is all it asks for.
type CampaignReader interface {
	All(ctx context.Context) ([]campaigns.Campaign, error)
	Enrich(ctx context.Context, all []campaigns.Campaign) map[string]campaigns.CampaignRow
	CreativesFor(ctx context.Context, campaignID string) ([]campaigns.Creative, error)
}

type Creators struct {
	Store     *store.Store
	Campaigns CampaignReader
}

func NewCreators(s *store.Store, c CampaignReader) *Creators {
	return &Creators{Store: s, Campaigns: c}
}

// tierMedians computes, for each tier, the median campaign reach across
// every creator in that tier. This is the baseline a creator is judged
// against — comparing a nano creator's reach to a mega creator's says
// nothing except that one has a bigger audience.
func tierMedians(roster []creators.Creator, campaignsByCreator map[string][]campaigns.Campaign) map[creators.Tier]float64 {
	reachByTier := map[creators.Tier][]float64{}
	for _, cr := range roster {
		for _, camp := range campaignsByCreator[cr.ID] {
			if !camp.IsRunning() {
				continue
			}
			reachByTier[cr.Tier] = append(reachByTier[cr.Tier], camp.Reach)
		}
	}
	out := map[creators.Tier]float64{}
	for tier, reaches := range reachByTier {
		out[tier] = campaigns.Median(reaches)
	}
	return out
}

func performanceFor(cr creators.Creator, camps []campaigns.Campaign, tierMedian float64) creators.CreatorPerformance {
	perf := creators.CreatorPerformance{Creator: cr, TierMedian: tierMedian}

	reaches := make([]float64, 0, len(camps))
	for _, c := range camps {
		perf.Campaigns++
		if c.Status == campaigns.StatusLive {
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

	perf.TierIndex = campaigns.Index(perf.AvgReach, tierMedian)
	perf.CostPerLakh = creators.CostPerLakhReach(perf.TotalSpend, perf.TotalReach)
	perf.Consistency = creators.Consistency(reaches)
	perf.ConsistencyN = len(reaches)
	perf.AudienceReachPct = creators.AudienceReachPct(perf.AvgReach, cr.Followers)
	return perf
}

// List returns the full roster ranked by how well each creator performs
// for their own tier.
func (c *Creators) List(ctx context.Context) ([]creators.CreatorPerformance, error) {
	roster, err := c.Store.ListCreators(ctx)
	if err != nil {
		return nil, err
	}
	all, err := c.Campaigns.All(ctx)
	if err != nil {
		return nil, err
	}

	byCreator := map[string][]campaigns.Campaign{}
	for _, camp := range all {
		if camp.CreatorID != "" {
			byCreator[camp.CreatorID] = append(byCreator[camp.CreatorID], camp)
		}
	}

	medians := tierMedians(roster, byCreator)
	out := make([]creators.CreatorPerformance, 0, len(roster))
	for _, cr := range roster {
		out = append(out, performanceFor(cr, byCreator[cr.ID], medians[cr.Tier]))
	}
	creators.RankCreators(out)
	return out, nil
}

type CreatorDetail struct {
	creators.CreatorPerformance
	CampaignRows []campaigns.CampaignRow
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
		return CreatorDetail{}, notFound(err)
	}

	roster, err := c.Store.ListCreators(ctx)
	if err != nil {
		return CreatorDetail{}, err
	}
	all, err := c.Campaigns.All(ctx)
	if err != nil {
		return CreatorDetail{}, err
	}

	byCreator := map[string][]campaigns.Campaign{}
	for _, camp := range all {
		if camp.CreatorID != "" {
			byCreator[camp.CreatorID] = append(byCreator[camp.CreatorID], camp)
		}
	}
	medians := tierMedians(roster, byCreator)

	camps := byCreator[cr.ID]
	perf := performanceFor(cr, camps, medians[cr.Tier])

	rowsByID := c.Campaigns.Enrich(ctx, all)
	rows := make([]campaigns.CampaignRow, 0, len(camps))
	for _, camp := range camps {
		rows = append(rows, rowsByID[camp.ID])
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Reach > rows[j].Reach })

	// Language mix, weighted by the reach of the campaign each creative ran on.
	langReach := map[string]*LanguageReach{}
	var langOrder []string
	for _, camp := range camps {
		creatives, err := c.Campaigns.CreativesFor(ctx, camp.ID)
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
	for _, other := range roster {
		if other.Tier == cr.Tier {
			peers++
		}
	}

	return CreatorDetail{
		CreatorPerformance: perf,
		CampaignRows:       rows,
		LanguageBreakdown:  breakdown,
		TierLabel:          creators.TierLabel(cr.Tier),
		TierPeers:          peers,
	}, nil
}

// notFound translates the store's not-found sentinel into this service's,
// which is the one httpx.WriteServiceError maps to a 404. Without it a
// missing row leaves here as database.ErrNotFound, which that mapper does
// not recognise — so a mistyped id came back as a 500 and wrote an
// "internal error" line to the log for what is really a client mistake.
func notFound(err error) error {
	if errors.Is(err, database.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
