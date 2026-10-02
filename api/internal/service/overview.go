package service

import (
	"context"
	"sort"

	"campaigntrackerpro/internal/domain"
)

type KPISparkline struct {
	Points []domain.SparklinePoint
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
	Spotlight        *CampaignRow
	// SpotlightCandidates are the rest of the campaigns worth surfacing,
	// best first, so the UI can cycle through them rather than fixing on
	// one. Flagged campaigns by index, topped up with movers.
	SpotlightCandidates []CampaignRow
	NeedsDecision       []CampaignRow
	Flagged             []CampaignRow
	Movers              []CampaignRow
	People              []CampaignRow
}

type OverviewService struct {
	Campaigns *Campaigns
}

func NewOverview(c *Campaigns) *OverviewService { return &OverviewService{Campaigns: c} }

func (o *OverviewService) Get(ctx context.Context) (Overview, error) {
	all, err := o.Campaigns.Store.ListCampaigns(ctx)
	if err != nil {
		return Overview{}, err
	}
	rowsByID, medAll, _, _ := o.Campaigns.enrich(ctx, all)

	running := domain.Running(all)
	sparklinePoints := domain.Sparkline(running)

	var liveCount int
	var reachLive float64
	var spendWindow, approvedBudget int64
	for _, c := range all {
		if c.Status == domain.StatusLive {
			liveCount++
			reachLive += c.Reach
		}
		spendWindow += c.Spend
		if c.Approval == domain.ApprovalApproved {
			approvedBudget += c.Budget
		}
	}
	pendingCount, err := o.Campaigns.Store.CountPendingApprovals(ctx)
	if err != nil {
		return Overview{}, err
	}
	pendingSpend, err := o.Campaigns.Store.TotalPendingSpend(ctx)
	if err != nil {
		return Overview{}, err
	}
	flagged, err := o.Campaigns.Store.ListFlaggedCampaigns(ctx)
	if err != nil {
		return Overview{}, err
	}

	spendPct := 0.0
	if approvedBudget > 0 {
		spendPct = float64(spendWindow) / float64(approvedBudget) * 100
	}

	reachSpark := KPISparkline{
		Points: sparklinePoints,
		Growth: domain.GrowthPct(sparklinePoints, func(p domain.SparklinePoint) float64 { return p.Reach }),
	}
	spendSpark := KPISparkline{
		Points: sparklinePoints,
		Growth: domain.GrowthPct(sparklinePoints, func(p domain.SparklinePoint) float64 { return p.Spend }),
	}
	liveSpark := KPISparkline{
		Points: sparklinePoints,
		Growth: domain.GrowthPct(sparklinePoints, func(p domain.SparklinePoint) float64 { return float64(p.LiveCount) }),
	}

	spotlightCamp, ok := domain.Spotlight(all, medAll)
	var spotlight *CampaignRow
	if ok {
		row := rowsByID[spotlightCamp.ID]
		spotlight = &row
	}

	pendingRows := toRows(rowsByID, filterCampaigns(all, func(c domain.Campaign) bool { return c.Approval == domain.ApprovalPending }))
	sortRowsDesc(pendingRows, func(r CampaignRow) float64 { return float64(r.Spend) })

	flaggedRows := toRows(rowsByID, flagged)
	sortRowsDesc(flaggedRows, func(r CampaignRow) float64 { return r.Reach })

	movers := domain.Movers(running, medAll)
	moverRows := make([]CampaignRow, 0, len(movers))
	for _, m := range movers {
		moverRows = append(moverRows, rowsByID[m.Campaign.ID])
	}

	peopleRows := toRows(rowsByID, filterCampaigns(all, func(c domain.Campaign) bool { return c.SubjectType == domain.SubjectPerson }))
	sortRowsDesc(peopleRows, func(r CampaignRow) float64 { return r.Reach })

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
func spotlightCandidates(flagged, movers []CampaignRow, max int) []CampaignRow {
	out := make([]CampaignRow, 0, max)
	seen := map[string]bool{}

	ranked := append([]CampaignRow(nil), flagged...)
	sortRowsDesc(ranked, func(r CampaignRow) float64 { return r.Index })
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

func filterCampaigns(all []domain.Campaign, keep func(domain.Campaign) bool) []domain.Campaign {
	out := make([]domain.Campaign, 0)
	for _, c := range all {
		if keep(c) {
			out = append(out, c)
		}
	}
	return out
}

func toRows(rowsByID map[string]CampaignRow, camps []domain.Campaign) []CampaignRow {
	out := make([]CampaignRow, 0, len(camps))
	for _, c := range camps {
		out = append(out, rowsByID[c.ID])
	}
	return out
}

func sortRowsDesc(rows []CampaignRow, key func(CampaignRow) float64) {
	sort.SliceStable(rows, func(i, j int) bool { return key(rows[i]) > key(rows[j]) })
}
