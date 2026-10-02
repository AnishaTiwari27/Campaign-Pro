package service

import (
	"campaigntrackerpro/services/campaigns"
	"context"
	"sort"
)

type CategoryCount struct {
	Category string
	Count    int
	Reach    float64
}

type RegionDetail struct {
	campaigns.RegionRollup
	PendingCount      int
	ShareOfBudget     float64
	Campaigns         []campaigns.CampaignRow
	CategoryBreakdown []CategoryCount
}

func (a *Analytics) Regions(ctx context.Context) ([]campaigns.RegionRollup, error) {
	all, err := a.campaigns.All(ctx)
	if err != nil {
		return nil, err
	}
	return campaigns.RegionRollups(all), nil
}

// RegionDetail adds pending count, share of tracked ad spend, the region's
// campaigns sorted by reach, and a category breakdown to that region's rollup.
func (a *Analytics) RegionDetail(ctx context.Context, region string) (RegionDetail, error) {
	all, err := a.campaigns.All(ctx)
	if err != nil {
		return RegionDetail{}, err
	}
	rollups := campaigns.RegionRollups(all)
	var rollup campaigns.RegionRollup
	found := false
	var totalSpendAll int64
	for _, r := range rollups {
		totalSpendAll += r.TotalSpend
		if r.Region == region {
			rollup, found = r, true
		}
	}
	if !found {
		return RegionDetail{}, ErrNotFound
	}

	rowsByID := a.campaigns.Enrich(ctx, all)
	share := 0.0
	if totalSpendAll > 0 {
		share = float64(rollup.TotalSpend) / float64(totalSpendAll) * 100
	}

	var pending int
	catCounts := map[string]*CategoryCount{}
	var order []string
	var regionCampaigns []campaigns.Campaign
	for _, camp := range all {
		if camp.Region != region {
			continue
		}
		regionCampaigns = append(regionCampaigns, camp)
		if camp.Approval == campaigns.ApprovalPending {
			pending++
		}
		cc, ok := catCounts[camp.Category]
		if !ok {
			cc = &CategoryCount{Category: camp.Category}
			catCounts[camp.Category] = cc
			order = append(order, camp.Category)
		}
		cc.Count++
		cc.Reach += camp.Reach
	}

	rows := toRows(rowsByID, regionCampaigns)
	sortRowsDesc(rows, func(r campaigns.CampaignRow) float64 { return r.Reach })

	breakdown := make([]CategoryCount, 0, len(order))
	for _, cat := range order {
		breakdown = append(breakdown, *catCounts[cat])
	}
	sort.Slice(breakdown, func(i, j int) bool { return breakdown[i].Reach > breakdown[j].Reach })

	return RegionDetail{
		RegionRollup:      rollup,
		PendingCount:      pending,
		ShareOfBudget:     share,
		Campaigns:         rows,
		CategoryBreakdown: breakdown,
	}, nil
}
