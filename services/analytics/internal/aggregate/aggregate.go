// Package aggregate holds the pure computation behind every analytics
// endpoint — no HTTP, no caching, no network. Each function here takes the
// small, pre-aggregated shape campaigns-service's SQL already computed
// (services/campaigns/internal/store/aggregates.go) and does only the
// output-sized work left: zero-filling, pivoting, spark-line/delta math,
// reshaping — never a scan over every campaign row. See
// docs/ROADMAP.md's Phase A for why (this package used to take
// []models.Campaign and do all of that in Go over the full filtered set;
// that was the real DB bottleneck at scale).
package aggregate

import (
	"fmt"
	"sort"

	"campaigntrackerpro/services/analytics/internal/models"
)

// canonicalRegions is a small, rarely-changing vocabulary duplicated here
// deliberately rather than fetched from catalog-service — it's just
// enough to zero-fill the region breakdown chart, not owned data. See
// docs/ARCHITECTURE.md for why analytics-service calls campaigns-service
// only, not catalog-service.
var canonicalRegions = []string{"Mumbai", "Delhi NCR", "Bengaluru", "South Zone", "West Zone", "Pan-India"}

// KPIs turns campaigns-service's totals + weekly buckets into the four KPI
// tiles. totals is the full-filtered-set count/sum (not week-scoped);
// weekly only covers weeks 0-7 ago, enough for an 8-point sparkline.
func KPIs(totals models.KPITotals, weekly []models.WeeklyBucket) models.KPISet {
	countByWeek := make(map[int]int64, len(weekly))
	reachByWeek := make(map[int]int64, len(weekly))
	spendByWeek := make(map[int]int64, len(weekly))
	for _, w := range weekly {
		countByWeek[w.WeeksAgo] = w.Count
		reachByWeek[w.WeeksAgo] = w.Reach
		spendByWeek[w.WeeksAgo] = w.Spend
	}
	countSpark := sparkFrom(countByWeek)
	reachSpark := sparkFrom(reachByWeek)
	spendSpark := sparkFrom(spendByWeek)

	countDelta, countUp := deltaFromSpark(countSpark)
	reachDelta, reachUp := deltaFromSpark(reachSpark)
	spendDelta, spendUp := deltaFromSpark(spendSpark)
	subjectDelta, subjectUp := deltaFromSpark(countSpark) // subject cadence tracks campaign cadence closely enough to reuse this spark

	return models.KPISet{
		ActiveCampaigns: models.KPI{Value: int64(totals.Count), Delta: countDelta, Up: countUp, Spark: countSpark},
		SubjectsTracked: models.KPI{Value: int64(totals.DistinctSubjects), Delta: subjectDelta, Up: subjectUp, Spark: countSpark},
		EstimatedReach:  models.KPI{Value: totals.Reach, Delta: reachDelta, Up: reachUp, Spark: reachSpark},
		EstimatedSpend:  models.KPI{Value: totals.Spend, Delta: spendDelta, Up: spendUp, Spark: spendSpark},
	}
}

// sparkFrom turns a weeksAgo->value map into an 8-point spark array,
// oldest to newest (spark[7] is the current week, weeksAgo 0).
func sparkFrom(byWeek map[int]int64) []int64 {
	spark := make([]int64, 8)
	for i := 0; i < 8; i++ {
		spark[i] = byWeek[7-i]
	}
	return spark
}

func deltaFromSpark(spark []int64) (string, bool) {
	prev, last := spark[len(spark)-2], spark[len(spark)-1]
	if prev == 0 {
		if last == 0 {
			return "0%", true
		}
		return "+100%", true
	}
	pct := float64(last-prev) / float64(prev) * 100
	up := pct >= 0
	sign := "+"
	if pct < 0 {
		sign = "-"
		pct = -pct
	}
	return fmt.Sprintf("%s%.0f%%", sign, pct), up
}

// Trend pivots campaigns-service's (month, adType, count) buckets into one
// row per month, one field per ad type — chronological order (by each
// month's actual start date, not alphabetically), same contract as before.
func Trend(buckets []models.TrendBucket) []map[string]any {
	type month struct {
		counts map[string]int
		start  string // YYYY-MM-DD, sorts correctly as a plain string
	}
	months := map[string]*month{}
	var order []string
	for _, b := range buckets {
		m, ok := months[b.Month]
		if !ok {
			m = &month{counts: map[string]int{}, start: b.MonthStart}
			months[b.Month] = m
			order = append(order, b.Month)
		}
		m.counts[b.AdType] = b.Count
	}
	sort.Slice(order, func(i, j int) bool { return months[order[i]].start < months[order[j]].start })

	out := make([]map[string]any, 0, len(order))
	for _, name := range order {
		row := map[string]any{"month": name}
		for adType, count := range months[name].counts {
			row[adType] = count
		}
		out = append(out, row)
	}
	return out
}

// Regions zero-fills every canonical region SQL didn't return a count for
// (no matching campaigns), then sorts by count descending — same output
// contract as before, now fed a handful of (region, count) pairs instead
// of deriving them by scanning every campaign row.
func Regions(counts []models.RegionCount) []models.RegionCount {
	byRegion := make(map[string]int, len(counts))
	for _, c := range counts {
		byRegion[c.Region] = c.Campaigns
	}
	out := make([]models.RegionCount, 0, len(canonicalRegions))
	for _, r := range canonicalRegions {
		out = append(out, models.RegionCount{Region: r, Campaigns: byRegion[r]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Campaigns > out[j].Campaigns })
	return out
}

// Benchmark reshapes campaigns-service's already-grouped/sorted/capped
// rows into the frontend's response shape. The group/sort/top-10 work
// (what this function used to do in Go, over every campaign row) now
// happens in SQL — see BenchmarkRows in
// services/campaigns/internal/store/aggregates.go.
func Benchmark(rows []models.BenchmarkAgg) []models.BenchmarkRow {
	out := make([]models.BenchmarkRow, len(rows))
	for i, a := range rows {
		out[i] = models.BenchmarkRow{
			Subject: a.Subject, SubjectType: a.SubjectType, Category: a.Category,
			Count: a.Count, Reach: a.Reach, Platforms: a.Platforms, Last: a.Last, Trend: a.Trend,
		}
	}
	return out
}
