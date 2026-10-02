package api

import (
	"campaigntrackerpro/services/analytics/internal/service"
	"campaigntrackerpro/services/campaigns"
)

type SparklineDTO struct {
	Points []campaigns.SparklinePoint `json:"points"`
	Growth float64                    `json:"growth"`
}

func sparklineDTO(s service.KPISparkline) SparklineDTO {
	return SparklineDTO{Points: s.Points, Growth: s.Growth}
}

type OverviewDTO struct {
	PendingCount        int                     `json:"pendingCount"`
	FlaggedCount        int                     `json:"flaggedCount"`
	PendingSpend        int64                   `json:"pendingSpend"`
	LiveCount           int                     `json:"liveCount"`
	LiveSparkline       SparklineDTO            `json:"liveSparkline"`
	PendingSparkline    SparklineDTO            `json:"pendingSparkline"`
	ReachLive           float64                 `json:"reachLive"`
	ReachSparkline      SparklineDTO            `json:"reachSparkline"`
	SpendWindow         int64                   `json:"spendWindow"`
	SpendSparkline      SparklineDTO            `json:"spendSparkline"`
	ApprovedBudget      int64                   `json:"approvedBudget"`
	SpendPctBudget      float64                 `json:"spendPctBudget"`
	MedAll              float64                 `json:"medAll"`
	Spotlight           *campaigns.CampaignDTO  `json:"spotlight,omitempty"`
	SpotlightCandidates []campaigns.CampaignDTO `json:"spotlightCandidates"`
	NeedsDecision       []campaigns.CampaignDTO `json:"needsDecision"`
	Flagged             []campaigns.CampaignDTO `json:"flagged"`
	Movers              []campaigns.CampaignDTO `json:"movers"`
	People              []campaigns.CampaignDTO `json:"people"`
}

func overviewDTO(o service.Overview) OverviewDTO {
	dto := OverviewDTO{
		PendingCount: o.PendingCount, FlaggedCount: o.FlaggedCount, PendingSpend: o.PendingSpend,
		LiveCount: o.LiveCount, LiveSparkline: sparklineDTO(o.LiveSparkline),
		PendingSparkline: sparklineDTO(o.PendingSparkline),
		ReachLive:        o.ReachLive, ReachSparkline: sparklineDTO(o.ReachSparkline),
		SpendWindow: o.SpendWindow, SpendSparkline: sparklineDTO(o.SpendSparkline),
		ApprovedBudget: o.ApprovedBudget, SpendPctBudget: o.SpendPctBudget, MedAll: o.MedAll,
		SpotlightCandidates: campaigns.NewCampaignDTOs(o.SpotlightCandidates),
		NeedsDecision:       campaigns.NewCampaignDTOs(o.NeedsDecision), Flagged: campaigns.NewCampaignDTOs(o.Flagged),
		Movers: campaigns.NewCampaignDTOs(o.Movers), People: campaigns.NewCampaignDTOs(o.People),
	}
	if o.Spotlight != nil {
		d := campaigns.NewCampaignDTO(*o.Spotlight)
		dto.Spotlight = &d
	}
	return dto
}

type CategoryBenchmarkDTO struct {
	Category    string  `json:"category"`
	N           int     `json:"n"`
	MedianReach float64 `json:"medianReach"`
	TopCampaign string  `json:"topCampaign"`
	TopReach    float64 `json:"topReach"`
	TotalSpend  int64   `json:"totalSpend"`
}

func benchmarkDTO(b service.BenchmarkResult) map[string]any {
	cats := make([]CategoryBenchmarkDTO, len(b.Categories))
	for i, c := range b.Categories {
		cats[i] = CategoryBenchmarkDTO{
			Category: c.Category, N: c.N, MedianReach: c.MedianReach,
			TopCampaign: c.TopCampaign, TopReach: c.TopReach, TotalSpend: c.TotalSpend,
		}
	}
	return map[string]any{"medAll": b.MedAll, "categories": cats}
}

type AdTypeCountDTO struct {
	AdType string `json:"adType"`
	Count  int    `json:"count"`
}

type RegionDTO struct {
	Region      string           `json:"region"`
	Count       int              `json:"count"`
	LiveCount   int              `json:"liveCount"`
	TotalReach  float64          `json:"totalReach"`
	TotalSpend  int64            `json:"totalSpend"`
	AdTypeSplit []AdTypeCountDTO `json:"adTypeSplit"`
}

func regionDTO(r campaigns.RegionRollup) RegionDTO {
	split := make([]AdTypeCountDTO, len(r.AdTypeSplit))
	for i, a := range r.AdTypeSplit {
		split[i] = AdTypeCountDTO{AdType: string(a.AdType), Count: a.Count}
	}
	return RegionDTO{
		Region: r.Region, Count: r.Count, LiveCount: r.LiveCount,
		TotalReach: r.TotalReach, TotalSpend: r.TotalSpend, AdTypeSplit: split,
	}
}

func regionDTOs(rs []campaigns.RegionRollup) []RegionDTO {
	out := make([]RegionDTO, len(rs))
	for i, r := range rs {
		out[i] = regionDTO(r)
	}
	return out
}

type CategoryCountDTO struct {
	Category string  `json:"category"`
	Count    int     `json:"count"`
	Reach    float64 `json:"reach"`
}

type RegionDetailDTO struct {
	RegionDTO
	PendingCount      int                     `json:"pendingCount"`
	ShareOfBudget     float64                 `json:"shareOfBudget"`
	Campaigns         []campaigns.CampaignDTO `json:"campaigns"`
	CategoryBreakdown []CategoryCountDTO      `json:"categoryBreakdown"`
}

func regionDetailDTO(d service.RegionDetail) RegionDetailDTO {
	breakdown := make([]CategoryCountDTO, len(d.CategoryBreakdown))
	for i, c := range d.CategoryBreakdown {
		breakdown[i] = CategoryCountDTO{Category: c.Category, Count: c.Count, Reach: c.Reach}
	}
	return RegionDetailDTO{
		RegionDTO: regionDTO(d.RegionRollup), PendingCount: d.PendingCount, ShareOfBudget: d.ShareOfBudget,
		Campaigns: campaigns.NewCampaignDTOs(d.Campaigns), CategoryBreakdown: breakdown,
	}
}

type SearchResultDTO struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
}

func searchDTOs(rs []service.SearchResult) []SearchResultDTO {
	out := make([]SearchResultDTO, len(rs))
	for i, r := range rs {
		out[i] = SearchResultDTO{Kind: r.Kind, ID: r.ID, Title: r.Title, Subtitle: r.Subtitle}
	}
	return out
}
