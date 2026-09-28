package aggregate

import (
	"testing"

	"campaigntrackerpro/services/analytics/internal/models"
)

func TestKPIs_Empty(t *testing.T) {
	got := KPIs(models.KPITotals{}, nil)
	if got.ActiveCampaigns.Value != 0 || got.SubjectsTracked.Value != 0 ||
		got.EstimatedReach.Value != 0 || got.EstimatedSpend.Value != 0 {
		t.Fatalf("expected all-zero counts on empty input, got %+v", got)
	}
	// deltaFromSpark's prev==0 && last==0 case: flat, not a decline.
	if got.ActiveCampaigns.Delta != "0%" || !got.ActiveCampaigns.Up {
		t.Fatalf("expected 0%% (up) delta on empty input, got %q up=%v", got.ActiveCampaigns.Delta, got.ActiveCampaigns.Up)
	}
}

func TestKPIs_TotalsAndDeltaFromZero(t *testing.T) {
	totals := models.KPITotals{Count: 3, DistinctSubjects: 2, Reach: 350000, Spend: 15000}
	// All 3 campaigns land in the current week (weeksAgo 0); the previous
	// week (weeksAgo 1) has nothing, so this is a rise from zero, not a
	// decline — same edge case the old per-row version covered.
	weekly := []models.WeeklyBucket{{WeeksAgo: 0, Count: 3, Reach: 350000, Spend: 15000}}

	got := KPIs(totals, weekly)

	if got.ActiveCampaigns.Value != 3 {
		t.Errorf("ActiveCampaigns = %d, want 3", got.ActiveCampaigns.Value)
	}
	if got.SubjectsTracked.Value != 2 {
		t.Errorf("SubjectsTracked = %d, want 2", got.SubjectsTracked.Value)
	}
	if got.EstimatedReach.Value != 350000 {
		t.Errorf("EstimatedReach = %d, want 350000", got.EstimatedReach.Value)
	}
	if got.EstimatedSpend.Value != 15000 {
		t.Errorf("EstimatedSpend = %d, want 15000", got.EstimatedSpend.Value)
	}
	if !got.ActiveCampaigns.Up || got.ActiveCampaigns.Delta != "+100%" {
		t.Errorf("ActiveCampaigns delta = %q up=%v, want +100%% up=true", got.ActiveCampaigns.Delta, got.ActiveCampaigns.Up)
	}
}

func TestKPIs_WeeksOutsideSparkWindowDontBreakIt(t *testing.T) {
	// totals cover the whole filtered set, but a bucket more than 7 weeks
	// back must simply not appear in the spark — not panic, not skew it.
	totals := models.KPITotals{Count: 5, DistinctSubjects: 3, Reach: 100, Spend: 10}
	weekly := []models.WeeklyBucket{
		{WeeksAgo: 0, Count: 2, Reach: 60, Spend: 6},
		{WeeksAgo: 40, Count: 3, Reach: 40, Spend: 4}, // old campaigns, outside the 8-point window
	}
	got := KPIs(totals, weekly)
	if got.ActiveCampaigns.Value != 5 {
		t.Errorf("ActiveCampaigns = %d, want 5 (totals aren't week-scoped)", got.ActiveCampaigns.Value)
	}
	if got.ActiveCampaigns.Spark[7] != 2 {
		t.Errorf("current-week spark = %d, want 2", got.ActiveCampaigns.Spark[7])
	}
}

func TestTrend_ChronologicalNotAlphabetical(t *testing.T) {
	// December sorts after January alphabetically but must come first here —
	// Trend orders by each bucket's month-start date, not the label.
	buckets := []models.TrendBucket{
		{Month: "Jan", MonthStart: "2026-01-01", AdType: "Performance", Count: 2},
		{Month: "Dec", MonthStart: "2025-12-01", AdType: "Performance", Count: 3},
	}

	rows := Trend(buckets)
	if len(rows) != 2 || rows[0]["month"] != "Dec" || rows[1]["month"] != "Jan" {
		t.Fatalf("expected [Dec, Jan] in chronological order, got %+v", rows)
	}
}

func TestTrend_PivotsAdTypesIntoColumns(t *testing.T) {
	buckets := []models.TrendBucket{
		{Month: "Aug", MonthStart: "2026-08-01", AdType: "Performance", Count: 4},
		{Month: "Aug", MonthStart: "2026-08-01", AdType: "Video", Count: 2},
	}
	rows := Trend(buckets)
	if len(rows) != 1 {
		t.Fatalf("expected one row for one month, got %d", len(rows))
	}
	if rows[0]["Performance"] != 4 || rows[0]["Video"] != 2 {
		t.Errorf("expected both ad types as columns on the same row, got %+v", rows[0])
	}
}

func TestRegions_ZeroFilledAndSortedDesc(t *testing.T) {
	counts := []models.RegionCount{
		{Region: "Mumbai", Campaigns: 2},
		{Region: "Bengaluru", Campaigns: 1},
	}

	rows := Regions(counts)
	if len(rows) != len(canonicalRegions) {
		t.Fatalf("expected all %d canonical regions present, got %d", len(canonicalRegions), len(rows))
	}
	if rows[0].Region != "Mumbai" || rows[0].Campaigns != 2 {
		t.Errorf("expected Mumbai first with count 2, got %+v", rows[0])
	}
	var sawZero bool
	for _, r := range rows {
		if r.Campaigns == 0 {
			sawZero = true
		}
	}
	if !sawZero {
		t.Error("expected at least one untouched region zero-filled, not omitted")
	}
}

// Benchmark's grouping, sorting, and top-10 cap now happen in SQL
// (services/campaigns/internal/store/aggregates.go's BenchmarkRows) — not
// unit-tested at this layer for the same reason KPITotals/WeeklyBuckets/
// TrendBuckets/RegionCounts' SQL isn't: verified against the live stack
// instead (docs/ROADMAP.md's Phase A verification section). What's left
// here is a pure reshape, so that's what's tested.
func TestBenchmark_ReshapesFieldsExactly(t *testing.T) {
	in := []models.BenchmarkAgg{
		{Subject: "CRED", SubjectType: "brand", Category: "Fintech", Count: 6, Reach: 2140000, Platforms: []string{"Meta Ads", "Google Search"}, Last: "2026-08-20", Trend: []int64{100, 200, 300}},
	}
	out := Benchmark(in)
	if len(out) != 1 {
		t.Fatalf("expected 1 row, got %d", len(out))
	}
	got := out[0]
	if got.Subject != "CRED" || got.SubjectType != "brand" || got.Category != "Fintech" ||
		got.Count != 6 || got.Reach != 2140000 || got.Last != "2026-08-20" ||
		len(got.Platforms) != 2 || got.Platforms[0] != "Meta Ads" {
		t.Errorf("reshape dropped or altered a field: got %+v", got)
	}
	if len(got.Trend) != 3 || got.Trend[2] != 300 {
		t.Errorf("Trend not carried through the reshape: got %+v", got.Trend)
	}
}

func TestBenchmark_EmptyTrendStaysEmpty(t *testing.T) {
	in := []models.BenchmarkAgg{{Subject: "Zepto", SubjectType: "brand"}} // no Trend — no snapshots taken yet
	out := Benchmark(in)
	if len(out[0].Trend) != 0 {
		t.Errorf("expected empty Trend when the source has none, got %+v", out[0].Trend)
	}
}
