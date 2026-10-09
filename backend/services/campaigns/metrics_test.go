package campaigns

import (
	"math"
	"strings"
	"testing"
)

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestReachCurve(t *testing.T) {
	curve := ReachCurve(100, CurveFast)
	want := [8]float64{10, 34, 55, 70, 81, 89, 95, 100}
	for i := range want {
		if !almostEqual(curve[i], want[i]) {
			t.Fatalf("fast[%d] = %v, want %v", i, curve[i], want[i])
		}
	}
	if got := ReachCurve(50, CurveSteady)[7]; !almostEqual(got, 50) {
		t.Fatalf("steady last point = %v, want 50 (full reach)", got)
	}
}

func TestReachCurveUnknownShapeFallsBackToSteady(t *testing.T) {
	got := ReachCurve(100, "bogus")
	want := ReachCurve(100, CurveSteady)
	if got != want {
		t.Fatalf("unknown shape = %v, want steady fallback %v", got, want)
	}
}

func TestValueAtAge(t *testing.T) {
	// daysRunning=14, steady shape: age 0 -> 0, age>=14 -> total, midpoint interpolates.
	if got := ValueAtAge(100, CurveSteady, 14, 0); got != 0 {
		t.Fatalf("age 0 = %v, want 0", got)
	}
	if got := ValueAtAge(100, CurveSteady, 14, 14); got != 100 {
		t.Fatalf("age == daysRunning = %v, want 100", got)
	}
	if got := ValueAtAge(100, CurveSteady, 14, 20); got != 100 {
		t.Fatalf("age > daysRunning = %v, want clamped 100", got)
	}
	if got := ValueAtAge(100, CurveSteady, 0, 5); got != 0 {
		t.Fatalf("daysRunning=0 = %v, want 0", got)
	}
	// age=7 of 14 -> fraction 0.5 -> curve position 3.5 -> halfway between
	// steady[3]=46 and steady[4]=59 -> 52.5.
	if got := ValueAtAge(100, CurveSteady, 14, 7); !almostEqual(got, 52.5) {
		t.Fatalf("age=7/14 = %v, want 52.5", got)
	}
}

func TestPace(t *testing.T) {
	cases := []struct {
		spend, budget int64
		want          float64
	}{
		{50, 100, 50},
		{100, 100, 100},
		{0, 0, 0},
		{43000, 100, 43000},
	}
	for _, c := range cases {
		if got := Pace(c.spend, c.budget); got != c.want {
			t.Fatalf("Pace(%d,%d) = %v, want %v", c.spend, c.budget, got, c.want)
		}
	}
}

func TestPaceClassOf(t *testing.T) {
	cases := []struct {
		pace float64
		want PaceClass
	}{
		{100, PaceOver},
		{150, PaceOver},
		{85, PaceWarn},
		{99, PaceWarn},
		{84.9, PaceGood},
		{0, PaceGood},
	}
	for _, c := range cases {
		if got := PaceClassOf(c.pace); got != c.want {
			t.Fatalf("PaceClassOf(%v) = %v, want %v", c.pace, got, c.want)
		}
	}
}

func TestCPM(t *testing.T) {
	// reach=10L, frequency=2 -> impressions = 10*100000*2 = 2,000,000
	// spend=200000 -> cpm = 200000/(2000000/1000) = 200000/2000 = 100.
	if got := CPM(200000, 10, 2); !almostEqual(got, 100) {
		t.Fatalf("CPM = %v, want 100", got)
	}
	if got := CPM(1000, 0, 0); got != 0 {
		t.Fatalf("CPM with no impressions = %v, want 0", got)
	}
}

func TestMedian(t *testing.T) {
	if got := Median(nil); got != 0 {
		t.Fatalf("Median(nil) = %v, want 0", got)
	}
	if got := Median([]float64{3, 1, 2}); got != 2 {
		t.Fatalf("Median odd = %v, want 2", got)
	}
	if got := Median([]float64{1, 2, 3, 4}); got != 2.5 {
		t.Fatalf("Median even = %v, want 2.5", got)
	}
}

func TestIndex(t *testing.T) {
	if got := Index(140, 100); !almostEqual(got, 1.4) {
		t.Fatalf("Index = %v, want 1.4", got)
	}
	if got := Index(50, 0); got != 0 {
		t.Fatalf("Index with zero median = %v, want 0", got)
	}
}

func mkCampaign(name, category, region string, adType AdType, status Status, reach float64, spend, budget int64, flag string) Campaign {
	return Campaign{
		ID: name, Name: name, Category: category, Region: region, AdType: adType,
		Status: status, Reach: reach, Spend: spend, Budget: budget, FlagReason: flag,
		CurveShape: CurveSteady, DaysRunning: 10,
	}
}

func TestRunningExcludesScheduled(t *testing.T) {
	all := []Campaign{
		mkCampaign("a", "Fintech", "Delhi NCR", AdSocial, StatusLive, 10, 1, 1, ""),
		mkCampaign("b", "Fintech", "Delhi NCR", AdSocial, StatusScheduled, 0, 0, 0, ""),
	}
	got := Running(all)
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("Running = %+v, want only campaign a", got)
	}
}

func TestCategoryBenchmarks(t *testing.T) {
	running := []Campaign{
		mkCampaign("a", "Fintech", "Delhi NCR", AdSocial, StatusLive, 10, 100, 100, ""),
		mkCampaign("b", "Fintech", "Delhi NCR", AdSocial, StatusLive, 30, 200, 100, ""),
		mkCampaign("c", "FMCG", "Mumbai", AdVideo, StatusLive, 5, 50, 100, ""),
	}
	bm := CategoryBenchmarks(running)
	if len(bm) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(bm))
	}
	// Fintech has higher median reach (20 vs 5) so should sort first.
	if bm[0].Category != "Fintech" {
		t.Fatalf("expected Fintech first, got %s", bm[0].Category)
	}
	if bm[0].N != 2 || bm[0].MedianReach != 20 || bm[0].TopCampaign != "b" || bm[0].TotalSpend != 300 {
		t.Fatalf("unexpected fintech benchmark: %+v", bm[0])
	}
}

func TestMoversThresholdAndSort(t *testing.T) {
	running := []Campaign{
		mkCampaign("a", "Fintech", "Delhi NCR", AdSocial, StatusLive, 100, 1, 1, ""), // index 1.0, excluded
		mkCampaign("b", "Fintech", "Delhi NCR", AdSocial, StatusLive, 130, 1, 1, ""), // index 1.3, included
		mkCampaign("c", "Fintech", "Delhi NCR", AdSocial, StatusLive, 200, 1, 1, ""), // index 2.0, included, first
	}
	movers := Movers(running, 100)
	if len(movers) != 2 {
		t.Fatalf("expected 2 movers, got %d: %+v", len(movers), movers)
	}
	if movers[0].Campaign.ID != "c" {
		t.Fatalf("expected c (highest index) first, got %s", movers[0].Campaign.ID)
	}
}

func TestSpotlightPrefersHighestIndexFlagged(t *testing.T) {
	campaigns := []Campaign{
		mkCampaign("a", "Fintech", "Delhi NCR", AdSocial, StatusLive, 50, 1, 1, ""),
		mkCampaign("b", "Fintech", "Delhi NCR", AdSocial, StatusLive, 300, 1, 1, "Reach outlier"),
		mkCampaign("c", "Fintech", "Delhi NCR", AdSocial, StatusLive, 150, 1, 1, "Under-delivering"),
	}
	got, ok := Spotlight(campaigns, 100)
	if !ok || got.ID != "b" {
		t.Fatalf("Spotlight = %+v, ok=%v, want b", got, ok)
	}
}

func TestSpotlightFallsBackToFirstWhenNoneFlagged(t *testing.T) {
	campaigns := []Campaign{
		mkCampaign("a", "Fintech", "Delhi NCR", AdSocial, StatusLive, 50, 1, 1, ""),
		mkCampaign("b", "Fintech", "Delhi NCR", AdSocial, StatusLive, 300, 1, 1, ""),
	}
	got, ok := Spotlight(campaigns, 100)
	if !ok || got.ID != "a" {
		t.Fatalf("Spotlight fallback = %+v, ok=%v, want a", got, ok)
	}
}

func TestSpotlightEmpty(t *testing.T) {
	_, ok := Spotlight(nil, 100)
	if ok {
		t.Fatalf("Spotlight(nil) ok = true, want false")
	}
}

func TestRegionRollups(t *testing.T) {
	all := []Campaign{
		mkCampaign("a", "Fintech", "Delhi NCR", AdSocial, StatusLive, 10, 100, 100, ""),
		mkCampaign("b", "FMCG", "Delhi NCR", AdVideo, StatusEnded, 20, 200, 100, ""),
		mkCampaign("c", "Fintech", "Mumbai", AdSocial, StatusLive, 5, 50, 100, ""),
	}
	rollups := RegionRollups(all)
	if len(rollups) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(rollups))
	}
	// sorted alphabetically: Delhi NCR, Mumbai
	if rollups[0].Region != "Delhi NCR" || rollups[0].Count != 2 || rollups[0].LiveCount != 1 {
		t.Fatalf("unexpected Delhi NCR rollup: %+v", rollups[0])
	}
	if rollups[0].TotalReach != 30 || rollups[0].TotalSpend != 300 {
		t.Fatalf("unexpected Delhi NCR totals: %+v", rollups[0])
	}
}

func TestSparklineAndGrowth(t *testing.T) {
	c := mkCampaign("a", "Fintech", "Delhi NCR", AdSocial, StatusLive, 100, 210000, 100, "")
	c.DaysRunning = 21
	points := Sparkline([]Campaign{c})
	if len(points) != 8 {
		t.Fatalf("expected 8 sparkline points, got %d", len(points))
	}
	if points[0].DaysAgo != 21 || points[7].DaysAgo != 0 {
		t.Fatalf("unexpected offsets: first=%d last=%d", points[0].DaysAgo, points[7].DaysAgo)
	}
	// 21 days ago == campaign start, so age=0 -> reach contribution 0.
	if points[0].Reach != 0 {
		t.Fatalf("reach 21 days ago = %v, want 0", points[0].Reach)
	}
	// today (offset 0) -> age == daysRunning -> full reach.
	if points[7].Reach != 100 {
		t.Fatalf("reach today = %v, want 100 (full)", points[7].Reach)
	}
	growth := GrowthPct(points, func(p SparklinePoint) float64 { return p.Reach })
	if growth <= 0 {
		t.Fatalf("growth = %v, want positive (reach grew from 0)", growth)
	}
}

func TestDetectAnomalyPriority(t *testing.T) {
	base := mkCampaign("a", "Fintech", "Delhi NCR", AdSocial, StatusLive, 10, 0, 100, "")
	base.DaysRunning = 5

	over := base
	over.Spend = 100
	if got := DetectAnomaly(over, 0, 1); got.Reason != "Over budget pace" {
		t.Fatalf("expected over-budget priority, got %+v", got)
	}

	warn := base
	warn.Spend = 95
	if got := DetectAnomaly(warn, 0, 1); got.Reason != "Budget nearly exhausted" {
		t.Fatalf("expected nearly-exhausted, got %+v", got)
	}

	outlier := base
	outlier.Spend = 10
	if got := DetectAnomaly(outlier, 1.8, 1); got.Reason != "Reach outlier" {
		t.Fatalf("expected reach outlier, got %+v", got)
	}

	under := base
	under.Spend = 10
	if got := DetectAnomaly(under, 0, 0.2); got.Reason != "Under-delivering" {
		t.Fatalf("expected under-delivering, got %+v", got)
	}

	fine := base
	fine.Spend = 10
	if got := DetectAnomaly(fine, 0, 1); got.Flagged {
		t.Fatalf("expected no flag, got %+v", got)
	}

	// Under-delivering requires >= 2 days running.
	tooNew := base
	tooNew.Spend = 10
	tooNew.DaysRunning = 1
	if got := DetectAnomaly(tooNew, 0, 0.1); got.Flagged {
		t.Fatalf("expected no flag for day-1 campaign, got %+v", got)
	}
}

func TestApprovalBlock(t *testing.T) {
	base := mkCampaign("a", "Fintech", "Delhi NCR", AdSocial, StatusLive, 10, 0, 100, "")
	base.DaysRunning = 5

	cases := []struct {
		name string
		// wantPace is the percentage the message must quote; "" means the
		// campaign must not be blocked at all.
		spend, budget int64
		wantPace      string
	}{
		{"over budget blocks", 120, 100, "120%"},
		{"exactly at budget blocks", 100, 100, "100%"},
		{"nearly exhausted blocks", 96, 100, "96%"},
		{"at the threshold blocks", 95, 100, "95%"},
		{"just under the threshold is fine", 94, 100, ""},
		{"healthy pace is fine", 40, 100, ""},
		{"nothing spent is fine", 0, 100, ""},
		{"no budget set is fine", 0, 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			camp := base
			camp.Spend, camp.Budget = c.spend, c.budget
			got := ApprovalBlock(camp)
			if c.wantPace == "" {
				if got != "" {
					t.Fatalf("spend %d of %d: expected no block, got %q", c.spend, c.budget, got)
				}
				return
			}
			if got == "" {
				t.Fatalf("spend %d of %d: expected a block, got none", c.spend, c.budget)
			}
			// The approver has to be told what to fix, so the pace is part
			// of the contract of the message, not decoration.
			if !strings.Contains(got, c.wantPace) {
				t.Fatalf("spend %d of %d: message %q should quote %s", c.spend, c.budget, got, c.wantPace)
			}
		})
	}
}

// The guard and the flag must trip together: a campaign the queue shows as
// "Over budget pace" must be one the API refuses to approve, and vice versa.
func TestApprovalBlockAgreesWithBudgetFlags(t *testing.T) {
	base := mkCampaign("a", "Fintech", "Delhi NCR", AdSocial, StatusLive, 10, 0, 100, "")
	base.DaysRunning = 5

	for spend := int64(0); spend <= 120; spend++ {
		camp := base
		camp.Spend = spend
		reason := DetectAnomaly(camp, 0, 1).Reason
		budgetFlagged := reason == "Over budget pace" || reason == "Budget nearly exhausted"
		blocked := ApprovalBlock(camp) != ""
		if budgetFlagged != blocked {
			t.Fatalf("spend %d of 100: budget flag %q but blocked=%v", spend, reason, blocked)
		}
	}
}

// Expected pace follows the flight's own curve, so the same age on the same
// planned length gives different answers for a fast and a slow flight.
func TestExpectedPaceFollowsTheFlightShape(t *testing.T) {
	const flight = 20

	halfway := map[CurveShape]float64{}
	for _, shape := range []CurveShape{CurveFast, CurveSteady, CurveSlow} {
		halfway[shape] = ExpectedPace(shape, flight/2, flight)
	}
	if !(halfway[CurveFast] > halfway[CurveSteady]) {
		t.Errorf("a fast flight should be further through its budget at halfway: fast %v, steady %v",
			halfway[CurveFast], halfway[CurveSteady])
	}
	if !(halfway[CurveSlow] < halfway[CurveSteady]) {
		t.Errorf("a slow flight should be behind steady at halfway: slow %v, steady %v",
			halfway[CurveSlow], halfway[CurveSteady])
	}

	// Both ends are fixed whatever the shape.
	for _, shape := range []CurveShape{CurveFast, CurveSteady, CurveSlow} {
		if got := ExpectedPace(shape, 0, flight); got != 0 {
			t.Errorf("%s at day 0 = %v, want 0", shape, got)
		}
		if got := ExpectedPace(shape, flight, flight); got != 100 {
			t.Errorf("%s at the end = %v, want 100", shape, got)
		}
	}

	// An unknown plan is no answer, never 0%.
	if got := ExpectedPace(CurveSteady, 10, 0); got != 0 {
		t.Errorf("no planned length should give 0 (unknown), got %v", got)
	}
}

func TestPaceVsPlan(t *testing.T) {
	cases := []struct {
		name           string
		pace, expected float64
		want           float64
	}{
		{"exactly on plan", 50, 50, 1},
		{"spending 40% ahead of the flight", 70, 50, 1.4},
		{"spending well behind", 25, 50, 0.5},
		// The case the whole column exists for: half the budget gone at the
		// halfway point is ON plan, which raw pace alone cannot say.
		{"halfway through a flight at half budget is on plan", 50, 50, 1},
		{"unknown plan", 50, 0, 0},
		{"nothing spent yet", 0, 50, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := PaceVsPlan(c.pace, c.expected); got != c.want {
				t.Fatalf("PaceVsPlan(%v, %v) = %v, want %v", c.pace, c.expected, got, c.want)
			}
		})
	}
}
