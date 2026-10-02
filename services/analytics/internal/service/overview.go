package service

import (
	"campaigntrackerpro/services/campaigns"
	"context"
	"sort"
)

type KPISparkline struct {
	Points []campaigns.SparklinePoint
	Growth float64
}

type Overview struct {
	PendingCount     int
	FlaggedCount     int
	PendingSpend     int64
	LiveCount        int
	LiveSparkline    KPISparkline
	PendingSparkline KPISparkline
	ReachLive        float64
	ReachSparkline   KPISparkline
	SpendWindow      int64
	SpendSparkline   KPISparkline
	ApprovedBudget   int64
	SpendPctBudget   float64
	MedAll           float64
	Spotlight        *campaigns.CampaignRow
	// SpotlightCandidates are the rest of the campaigns worth surfacing,
	// best first, so the UI can cycle through them rather than fixing on
	// one. Flagged campaigns by index, topped up with movers.
	SpotlightCandidates []campaigns.CampaignRow
	NeedsDecision       []campaigns.CampaignRow
	Flagged             []campaigns.CampaignRow
	Movers              []campaigns.CampaignRow
	People              []campaigns.CampaignRow
}

func (a *Analytics) Get(ctx context.Context) (Overview, error) {
	all, err := a.campaigns.All(ctx)
	if err != nil {
		return Overview{}, err
	}
	rowsByID := a.campaigns.Enrich(ctx, all)
	medAll := campaigns.MedianReach(campaigns.Running(all))

	running := campaigns.Running(all)
	sparklinePoints := campaigns.Sparkline(running)

	var liveCount int
	var reachLive float64
	var spendWindow, approvedBudget int64
	for _, c := range all {
		if c.Status == campaigns.StatusLive {
			liveCount++
			reachLive += c.Reach
		}
		spendWindow += c.Spend
		if c.Approval == campaigns.ApprovalApproved {
			approvedBudget += c.Budget
		}
	}
	pendingCount, err := a.campaigns.PendingCount(ctx)
	if err != nil {
		return Overview{}, err
	}
	pendingSpend, err := a.campaigns.PendingSpend(ctx)
	if err != nil {
		return Overview{}, err
	}
	flagged, err := a.campaigns.Flagged(ctx)
	if err != nil {
		return Overview{}, err
	}

	spendPct := 0.0
	if approvedBudget > 0 {
		spendPct = float64(spendWindow) / float64(approvedBudget) * 100
	}

	reachSpark := KPISparkline{
		Points: sparklinePoints,
		Growth: campaigns.GrowthPct(sparklinePoints, func(p campaigns.SparklinePoint) float64 { return p.Reach }),
	}
	spendSpark := KPISparkline{
		Points: sparklinePoints,
		Growth: campaigns.GrowthPct(sparklinePoints, func(p campaigns.SparklinePoint) float64 { return p.Spend }),
	}
	liveSpark := KPISparkline{
		Points: sparklinePoints,
		Growth: campaigns.GrowthPct(sparklinePoints, func(p campaigns.SparklinePoint) float64 { return float64(p.LiveCount) }),
	}

	spotlightCamp, ok := campaigns.Spotlight(all, medAll)
	var spotlight *campaigns.CampaignRow
	if ok {
		row := rowsByID[spotlightCamp.ID]
		spotlight = &row
	}

	pendingRows := toRows(rowsByID, filterCampaigns(all, func(c campaigns.Campaign) bool { return c.Approval == campaigns.ApprovalPending }))
	sortRowsDesc(pendingRows, func(r campaigns.CampaignRow) float64 { return float64(r.Spend) })

	flaggedRows := toRows(rowsByID, flagged)
	sortRowsDesc(flaggedRows, func(r campaigns.CampaignRow) float64 { return r.Reach })

	movers := campaigns.Movers(running, medAll)
	moverRows := make([]campaigns.CampaignRow, 0, len(movers))
	for _, m := range movers {
		moverRows = append(moverRows, rowsByID[m.Campaign.ID])
	}

	peopleRows := toRows(rowsByID, filterCampaigns(all, func(c campaigns.Campaign) bool { return c.SubjectType == campaigns.SubjectPerson }))
	sortRowsDesc(peopleRows, func(r campaigns.CampaignRow) float64 { return r.Reach })

	candidates := spotlightCandidates(flaggedRows, moverRows, spotlightMax)

	return Overview{
		PendingCount:        int(pendingCount),
		FlaggedCount:        len(flagged),
		PendingSpend:        pendingSpend,
		LiveCount:           liveCount,
		LiveSparkline:       liveSpark,
		PendingSparkline:    liveSpark,
		ReachLive:           reachLive,
		ReachSparkline:      reachSpark,
		SpendWindow:         spendWindow,
		SpendSparkline:      spendSpark,
		ApprovedBudget:      approvedBudget,
		SpendPctBudget:      spendPct,
		MedAll:              medAll,
		Spotlight:           spotlight,
		SpotlightCandidates: candidates,
		NeedsDecision:       pendingRows,
		Flagged:             flaggedRows,
		Movers:              moverRows,
		People:              peopleRows,
	}, nil
}

// spotlightMax caps how many campaigns the overview will cycle through —
// enough to be worth rotating, few enough that each one gets real screen
// time before it comes round again.
const spotlightMax = 6

// spotlightCandidates ranks what deserves attention: anything anomaly
// detection flagged, best index first, topped up with the strongest
// over-performers when there aren't enough flags to fill the row.
func spotlightCandidates(flagged, movers []campaigns.CampaignRow, max int) []campaigns.CampaignRow {
	out := make([]campaigns.CampaignRow, 0, max)
	seen := map[string]bool{}

	ranked := append([]campaigns.CampaignRow(nil), flagged...)
	sortRowsDesc(ranked, func(r campaigns.CampaignRow) float64 { return r.Index })
	for _, r := range ranked {
		if len(out) == max {
			return out
		}
		if !seen[r.ID] {
			seen[r.ID] = true
			out = append(out, r)
		}
	}
	for _, r := range movers {
		if len(out) == max {
			return out
		}
		if !seen[r.ID] {
			seen[r.ID] = true
			out = append(out, r)
		}
	}
	return out
}

func filterCampaigns(all []campaigns.Campaign, keep func(campaigns.Campaign) bool) []campaigns.Campaign {
	out := make([]campaigns.Campaign, 0)
	for _, c := range all {
		if keep(c) {
			out = append(out, c)
		}
	}
	return out
}

func toRows(rowsByID map[string]campaigns.CampaignRow, camps []campaigns.Campaign) []campaigns.CampaignRow {
	out := make([]campaigns.CampaignRow, 0, len(camps))
	for _, c := range camps {
		out = append(out, rowsByID[c.ID])
	}
	return out
}

func sortRowsDesc(rows []campaigns.CampaignRow, key func(campaigns.CampaignRow) float64) {
	sort.SliceStable(rows, func(i, j int) bool { return key(rows[i]) > key(rows[j]) })
}
