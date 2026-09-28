// Package models holds analytics-service's JSON-facing response types —
// both what it sends to the frontend (KPISet, TrendBucket-derived rows,
// RegionCount, BenchmarkRow) and what it decodes from campaigns-service's
// pre-aggregated /internal/aggregates/* endpoints (KPIRawData,
// WeeklyBucket, TrendBucket, BenchmarkAgg). This service owns no data of
// its own — everything here is either a decode target or a reshape of
// what campaigns-service already computed in SQL; see
// campaignsclient/client.go and docs/ROADMAP.md's Phase A.
package models

type KPI struct {
	Value int64   `json:"value"`
	Delta string  `json:"delta"`
	Up    bool    `json:"up"`
	Spark []int64 `json:"spark"`
}

type KPISet struct {
	ActiveCampaigns KPI `json:"activeCampaigns"`
	SubjectsTracked KPI `json:"subjectsTracked"`
	EstimatedReach  KPI `json:"estimatedReach"`
	EstimatedSpend  KPI `json:"estimatedSpend"`
}

// KPITotals and WeeklyBucket mirror campaigns-service's
// /internal/aggregates/kpis response shape (services/campaigns/internal/models/aggregates.go) —
// aggregate.KPIs turns these into the KPISet above.
type KPITotals struct {
	Count            int   `json:"count"`
	DistinctSubjects int   `json:"distinctSubjects"`
	Reach            int64 `json:"reach"`
	Spend            int64 `json:"spend"`
}

type WeeklyBucket struct {
	WeeksAgo int   `json:"weeksAgo"`
	Count    int64 `json:"count"`
	Reach    int64 `json:"reach"`
	Spend    int64 `json:"spend"`
}

type KPIRawData struct {
	Totals KPITotals      `json:"totals"`
	Weekly []WeeklyBucket `json:"weekly"`
}

// TrendBucket mirrors /internal/aggregates/trend's response shape —
// aggregate.Trend pivots these into the frontend's month-keyed rows.
type TrendBucket struct {
	Month      string `json:"month"`
	MonthStart string `json:"monthStart"`
	AdType     string `json:"adType"`
	Count      int    `json:"count"`
}

// RegionCount is both the decode target for /internal/aggregates/regions
// and (after aggregate.Regions zero-fills the canonical list) the
// frontend-facing shape — identical either way, no reshape needed.
type RegionCount struct {
	Region    string `json:"region"`
	Campaigns int    `json:"campaigns"`
}

// BenchmarkAgg mirrors /internal/aggregates/benchmark's response shape —
// already grouped/sorted/capped by SQL; aggregate.Benchmark just reshapes
// it into BenchmarkRow.
type BenchmarkAgg struct {
	Subject     string   `json:"subject"`
	SubjectType string   `json:"subjectType"`
	Category    string   `json:"category"`
	Count       int      `json:"count"`
	Reach       int64    `json:"reach"`
	Platforms   []string `json:"platforms"`
	Last        string   `json:"last"`
	// Trend mirrors campaigns-service's own BenchmarkAgg.Trend — up to 8
	// recent benchmark_snapshots reach points, oldest -> newest, empty
	// until at least one snapshot exists (see docs/ROADMAP.md's Phase D).
	Trend []int64 `json:"trend,omitempty"`
}

type BenchmarkRow struct {
	Subject     string   `json:"subject"`
	SubjectType string   `json:"subjectType"`
	Category    string   `json:"category"`
	Count       int      `json:"count"`
	Reach       int64    `json:"reach"`
	Platforms   []string `json:"platforms"`
	Last        string   `json:"last"`
	Trend       []int64  `json:"trend,omitempty"`
}
