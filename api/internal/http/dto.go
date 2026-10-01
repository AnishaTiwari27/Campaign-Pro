package httpapi

import (
	"time"

	"campaigntrackerpro/internal/domain"
	"campaigntrackerpro/internal/service"
)

const rfc3339 = time.RFC3339

type CampaignDTO struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	SubjectType   string  `json:"subjectType"`
	Role          string  `json:"role,omitempty"`
	Initials      string  `json:"initials"`
	Category      string  `json:"category"`
	Region        string  `json:"region"`
	AdType        string  `json:"adType"`
	Platform      string  `json:"platform"`
	Status        string  `json:"status"`
	DaysRunning   int     `json:"daysRunning"`
	Reach         float64 `json:"reach"`
	Spend         int64   `json:"spend"`
	Budget        int64   `json:"budget"`
	Frequency     float64 `json:"frequency"`
	Approval      string  `json:"approval"`
	CurveShape    string  `json:"curveShape"`
	FlagReason    string  `json:"flagReason,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
	Pace          float64 `json:"pace"`
	PaceClass     string  `json:"paceClass"`
	CPM           float64 `json:"cpm"`
	Index         float64 `json:"index"`
	CategoryIndex float64 `json:"categoryIndex"`
}

func campaignDTO(r service.CampaignRow) CampaignDTO {
	return CampaignDTO{
		ID: r.ID, Name: r.Name, SubjectType: string(r.SubjectType), Role: r.Role,
		Initials: r.Initials, Category: r.Category, Region: r.Region, AdType: string(r.AdType),
		Platform: r.Platform, Status: string(r.Status), DaysRunning: r.DaysRunning,
		Reach: r.Reach, Spend: r.Spend, Budget: r.Budget, Frequency: r.Frequency,
		Approval: string(r.Approval), CurveShape: string(r.CurveShape), FlagReason: r.FlagReason,
		CreatedAt: r.CreatedAt.Format(rfc3339), UpdatedAt: r.UpdatedAt.Format(rfc3339),
		Pace: r.Pace, PaceClass: string(r.PaceClass), CPM: r.CPM, Index: r.Index,
		CategoryIndex: r.CategoryIndex,
	}
}

func campaignDTOs(rows []service.CampaignRow) []CampaignDTO {
	out := make([]CampaignDTO, len(rows))
	for i, r := range rows {
		out[i] = campaignDTO(r)
	}
	return out
}

type ListResponse struct {
	Items []CampaignDTO `json:"items"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Pages int           `json:"pages"`
}

func listResponse(r service.ListResult) ListResponse {
	return ListResponse{Items: campaignDTOs(r.Items), Total: r.Total, Page: r.Page, Pages: r.Pages}
}

type CreativeDTO struct {
	ID            string  `json:"id"`
	Headline      string  `json:"headline"`
	Kind          string  `json:"kind"`
	DurationLabel string  `json:"durationLabel"`
	Reach         float64 `json:"reach"`
	CTR           float64 `json:"ctr"`
}

func creativeDTO(c domain.Creative) CreativeDTO {
	return CreativeDTO{ID: c.ID, Headline: c.Headline, Kind: c.Kind, DurationLabel: c.DurationLabel, Reach: c.Reach, CTR: c.CTR}
}

func creativeDTOs(cs []domain.Creative) []CreativeDTO {
	out := make([]CreativeDTO, len(cs))
	for i, c := range cs {
		out[i] = creativeDTO(c)
	}
	return out
}

type AuditEventDTO struct {
	ID        int64  `json:"id"`
	Actor     string `json:"actor"`
	Action    string `json:"action"`
	Kind      string `json:"kind"`
	CreatedAt string `json:"createdAt"`
}

func auditDTO(e domain.AuditEvent) AuditEventDTO {
	return AuditEventDTO{ID: e.ID, Actor: e.Actor, Action: e.Action, Kind: e.Kind, CreatedAt: e.CreatedAt.Format(rfc3339)}
}

func auditDTOs(es []domain.AuditEvent) []AuditEventDTO {
	out := make([]AuditEventDTO, len(es))
	for i, e := range es {
		out[i] = auditDTO(e)
	}
	return out
}

type CampaignDetailDTO struct {
	CampaignDTO
	Creatives []CreativeDTO   `json:"creatives"`
	Audit     []AuditEventDTO `json:"audit"`
	Similar   []CampaignDTO   `json:"similar"`
	Position  int             `json:"position"`
	Total     int             `json:"total"`
	PrevID    string          `json:"prevId,omitempty"`
	NextID    string          `json:"nextId,omitempty"`
	MedAll    float64         `json:"medAll"`
}

func campaignDetailDTO(d service.CampaignDetail) CampaignDetailDTO {
	return CampaignDetailDTO{
		CampaignDTO: campaignDTO(d.CampaignRow),
		Creatives:   creativeDTOs(d.Creatives),
		Audit:       auditDTOs(d.Audit),
		Similar:     campaignDTOs(d.Similar),
		Position:    d.Position,
		Total:       d.Total,
		PrevID:      d.PrevID,
		NextID:      d.NextID,
		MedAll:      d.MedAll,
	}
}

type SparklineDTO struct {
	Points []domain.SparklinePoint `json:"points"`
	Growth float64                 `json:"growth"`
}

func sparklineDTO(s service.KPISparkline) SparklineDTO {
	return SparklineDTO{Points: s.Points, Growth: s.Growth}
}

type OverviewDTO struct {
	PendingCount     int           `json:"pendingCount"`
	FlaggedCount     int           `json:"flaggedCount"`
	PendingSpend     int64         `json:"pendingSpend"`
	LiveCount        int           `json:"liveCount"`
	LiveSparkline    SparklineDTO  `json:"liveSparkline"`
	PendingSparkline SparklineDTO  `json:"pendingSparkline"`
	ReachLive        float64       `json:"reachLive"`
	ReachSparkline   SparklineDTO  `json:"reachSparkline"`
	SpendWindow      int64         `json:"spendWindow"`
	SpendSparkline   SparklineDTO  `json:"spendSparkline"`
	ApprovedBudget   int64         `json:"approvedBudget"`
	SpendPctBudget   float64       `json:"spendPctBudget"`
	MedAll           float64       `json:"medAll"`
	Spotlight        *CampaignDTO  `json:"spotlight,omitempty"`
	NeedsDecision    []CampaignDTO `json:"needsDecision"`
	Flagged          []CampaignDTO `json:"flagged"`
	Movers           []CampaignDTO `json:"movers"`
	People           []CampaignDTO `json:"people"`
}

func overviewDTO(o service.Overview) OverviewDTO {
	dto := OverviewDTO{
		PendingCount: o.PendingCount, FlaggedCount: o.FlaggedCount, PendingSpend: o.PendingSpend,
		LiveCount: o.LiveCount, LiveSparkline: sparklineDTO(o.LiveSparkline),
		PendingSparkline: sparklineDTO(o.PendingSparkline),
		ReachLive:        o.ReachLive, ReachSparkline: sparklineDTO(o.ReachSparkline),
		SpendWindow: o.SpendWindow, SpendSparkline: sparklineDTO(o.SpendSparkline),
		ApprovedBudget: o.ApprovedBudget, SpendPctBudget: o.SpendPctBudget, MedAll: o.MedAll,
		NeedsDecision: campaignDTOs(o.NeedsDecision), Flagged: campaignDTOs(o.Flagged),
		Movers: campaignDTOs(o.Movers), People: campaignDTOs(o.People),
	}
	if o.Spotlight != nil {
		d := campaignDTO(*o.Spotlight)
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

func regionDTO(r domain.RegionRollup) RegionDTO {
	split := make([]AdTypeCountDTO, len(r.AdTypeSplit))
	for i, a := range r.AdTypeSplit {
		split[i] = AdTypeCountDTO{AdType: string(a.AdType), Count: a.Count}
	}
	return RegionDTO{
		Region: r.Region, Count: r.Count, LiveCount: r.LiveCount,
		TotalReach: r.TotalReach, TotalSpend: r.TotalSpend, AdTypeSplit: split,
	}
}

func regionDTOs(rs []domain.RegionRollup) []RegionDTO {
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
	PendingCount      int                `json:"pendingCount"`
	ShareOfBudget     float64            `json:"shareOfBudget"`
	Campaigns         []CampaignDTO      `json:"campaigns"`
	CategoryBreakdown []CategoryCountDTO `json:"categoryBreakdown"`
}

func regionDetailDTO(d service.RegionDetail) RegionDetailDTO {
	breakdown := make([]CategoryCountDTO, len(d.CategoryBreakdown))
	for i, c := range d.CategoryBreakdown {
		breakdown[i] = CategoryCountDTO{Category: c.Category, Count: c.Count, Reach: c.Reach}
	}
	return RegionDetailDTO{
		RegionDTO: regionDTO(d.RegionRollup), PendingCount: d.PendingCount, ShareOfBudget: d.ShareOfBudget,
		Campaigns: campaignDTOs(d.Campaigns), CategoryBreakdown: breakdown,
	}
}

type ReportDTO struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Enabled      bool           `json:"enabled"`
	Cadence      string         `json:"cadence"`
	CadenceLabel string         `json:"cadenceLabel"`
	Recipients   []string       `json:"recipients"`
	ScopeFilters map[string]any `json:"scopeFilters"`
	ScopeLabel   string         `json:"scopeLabel"`
	Columns      []string       `json:"columns"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
	LastRun      *ReportRunDTO  `json:"lastRun,omitempty"`
}

func reportDTO(r domain.Report) ReportDTO {
	return ReportDTO{
		ID: r.ID, Name: r.Name, Enabled: r.Enabled, Cadence: r.Cadence,
		CadenceLabel: domain.CadenceLabel(domain.Cadence(r.Cadence)),
		Recipients:   r.Recipients, ScopeFilters: r.ScopeFilters, ScopeLabel: r.ScopeLabel,
		Columns: r.Columns, CreatedAt: r.CreatedAt.Format(rfc3339), UpdatedAt: r.UpdatedAt.Format(rfc3339),
	}
}

func reportDTOs(rs []domain.Report) []ReportDTO {
	out := make([]ReportDTO, len(rs))
	for i, r := range rs {
		out[i] = reportDTO(r)
	}
	return out
}

type ReportRunDTO struct {
	RanAt    string `json:"ranAt"`
	Result   string `json:"result"`
	RowCount int    `json:"rowCount"`
}

func reportRunDTO(r domain.ReportRun) ReportRunDTO {
	return ReportRunDTO{RanAt: r.RanAt.Format(rfc3339), Result: r.Result, RowCount: r.RowCount}
}

func reportRunDTOs(rs []domain.ReportRun) []ReportRunDTO {
	out := make([]ReportRunDTO, len(rs))
	for i, r := range rs {
		out[i] = reportRunDTO(r)
	}
	return out
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
