// Aggregate response shapes for /internal/aggregates/* — computed in SQL
// (services/campaigns/internal/store/aggregates.go), not by scanning every
// row. analytics-service decodes these instead of pulling raw campaign
// rows and aggregating in Go memory over HTTP — see docs/ROADMAP.md's
// Phase A for why (the O(n)-over-the-wire path was the real DB
// bottleneck at scale).
package models

// KPITotals is the full-filtered-set totals behind the four KPI tiles —
// one row, not one row per campaign.
type KPITotals struct {
	Count            int   `json:"count"`
	DistinctSubjects int   `json:"distinctSubjects"`
	Reach            int64 `json:"reach"`
	Spend            int64 `json:"spend"` // rupees
}

// WeeklyBucket is one week's totals, WeeksAgo 0 being the current week —
// only weeks 0-7 are ever populated (enough for an 8-point sparkline);
// campaigns older than that still count toward KPITotals, just not here.
type WeeklyBucket struct {
	WeeksAgo int   `json:"weeksAgo"`
	Count    int64 `json:"count"`
	Reach    int64 `json:"reach"`
	Spend    int64 `json:"spend"` // rupees
}

// KPIRawData is the full /internal/aggregates/kpis response — totals plus
// the weekly buckets analytics-service turns into sparklines/deltas.
type KPIRawData struct {
	Totals KPITotals      `json:"totals"`
	Weekly []WeeklyBucket `json:"weekly"`
}

// TrendBucket is one (month, ad type) count — one row per combination
// present in the data, not one row per campaign.
type TrendBucket struct {
	Month      string `json:"month"`      // "Jan".."Dec", for display
	MonthStart string `json:"monthStart"` // YYYY-MM-DD (first of month), for chronological sort
	AdType     string `json:"adType"`
	Count      int    `json:"count"`
}

// RegionCount is one region's campaign count — SQL only returns regions
// that actually have matching campaigns; zero-filling the rest of the
// canonical list happens on the analytics-service side (aggregate.Regions).
type RegionCount struct {
	Region    string `json:"region"`
	Campaigns int    `json:"campaigns"`
}

// BenchmarkAgg is one subject's aggregated stats — already grouped,
// sorted, and capped to 10 by SQL (services/campaigns/internal/store/aggregates.go's
// BenchmarkRows), not recomputed from raw rows downstream.
type BenchmarkAgg struct {
	Subject     string   `json:"subject"`
	SubjectType string   `json:"subjectType"`
	Category    string   `json:"category"`
	Count       int      `json:"count"`
	Reach       int64    `json:"reach"`
	Platforms   []string `json:"platforms"`
	Last        string   `json:"last"` // YYYY-MM-DD
	// Trend is up to 8 recent campaigns.benchmark_snapshots.total_reach
	// points for this subject, oldest -> newest (matching the KPI tiles'
	// sparkline shape) — empty until at least one snapshot has been taken
	// (see the BENCHMARK_SNAPSHOT_ENABLED ticker, cmd/server/main.go), so
	// omitted rather than a misleadingly-flat single-point line.
	Trend []int64 `json:"trend,omitempty"`
}
