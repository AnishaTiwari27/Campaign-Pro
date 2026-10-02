package campaigns

import (
	"math"
	"sort"
)

// curveWeights are the 8-point cumulative-reach-share series for each flight
// shape, exactly as specified: fast flights front-load delivery, slow
// flights back-load it.
var curveWeights = map[CurveShape][8]float64{
	CurveFast:   {.10, .34, .55, .70, .81, .89, .95, 1},
	CurveSteady: {.08, .20, .33, .46, .59, .72, .86, 1},
	CurveSlow:   {.04, .09, .17, .28, .42, .60, .79, 1},
}

// ReachCurve returns the 8-point cumulative reach series for a campaign's
// current total reach and flight shape.
func ReachCurve(reach float64, shape CurveShape) [8]float64 {
	w, ok := curveWeights[shape]
	if !ok {
		w = curveWeights[CurveSteady]
	}
	var out [8]float64
	for i, f := range w {
		out[i] = reach * f
	}
	return out
}

// ValueAtAge interpolates a cumulative value (reach in lakh, or spend in
// rupees) at `age` days into a campaign's flight, along its 8-point curve.
// The curve's 8 checkpoints are spread evenly across [0, daysRunning]; ages
// before the flight started or after "today" clamp to 0 / total.
func ValueAtAge(total float64, shape CurveShape, daysRunning, age int) float64 {
	if daysRunning <= 0 || age <= 0 {
		return 0
	}
	if age >= daysRunning {
		return total
	}
	curve := ReachCurve(total, shape)
	frac := float64(age) / float64(daysRunning)
	pos := frac * 7
	lo := int(math.Floor(pos))
	if lo > 6 {
		lo = 6
	}
	hi := lo + 1
	t := pos - float64(lo)
	return curve[lo] + (curve[hi]-curve[lo])*t
}

// Pace is the budget-pace percentage: how much of the budget has been spent.
func Pace(spend, budget int64) float64 {
	if budget == 0 {
		return 0
	}
	return math.Round(float64(spend) / float64(budget) * 100)
}

type PaceClass string

const (
	PaceOver PaceClass = "over"
	PaceWarn PaceClass = "warn"
	PaceGood PaceClass = "good"
)

func PaceClassOf(pace float64) PaceClass {
	switch {
	case pace >= 100:
		return PaceOver
	case pace >= 85:
		return PaceWarn
	default:
		return PaceGood
	}
}

// CPM is cost per 1,000 impressions. Impressions are reach (lakh) converted
// to people, times average frequency. Returns 0 when there's nothing to
// divide by; callers render that as "—".
func CPM(spend int64, reachLakh, frequency float64) float64 {
	impressions := reachLakh * 100000 * frequency
	if impressions <= 0 {
		return 0
	}
	return float64(spend) / (impressions / 1000)
}

// Median returns the median of values; 0 for an empty slice.
func Median(values []float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := n / 2
	if n%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// Index expresses reach as a multiple of a baseline median, e.g. 1.4 -> "1.4x".
func Index(reach, median float64) float64 {
	if median == 0 {
		return 0
	}
	return reach / median
}

// Running filters campaigns to the "running set": everything except scheduled.
func Running(campaigns []Campaign) []Campaign {
	out := make([]Campaign, 0, len(campaigns))
	for _, c := range campaigns {
		if c.IsRunning() {
			out = append(out, c)
		}
	}
	return out
}

// MedianReach is MED_ALL: the all-campaign baseline every index in the app
// is expressed against.
func MedianReach(running []Campaign) float64 {
	vals := make([]float64, 0, len(running))
	for _, c := range running {
		vals = append(vals, c.Reach)
	}
	return Median(vals)
}

type CategoryBenchmark struct {
	Category    string
	N           int
	MedianReach float64
	TopCampaign string
	TopReach    float64
	TotalSpend  int64
}

// CategoryBenchmarks computes, per category over the running set: count,
// median reach, the top campaign by reach, and total spend. Sorted by
// median reach descending.
func CategoryBenchmarks(running []Campaign) []CategoryBenchmark {
	byCat := map[string][]Campaign{}
	order := []string{}
	for _, c := range running {
		if _, ok := byCat[c.Category]; !ok {
			order = append(order, c.Category)
		}
		byCat[c.Category] = append(byCat[c.Category], c)
	}

	out := make([]CategoryBenchmark, 0, len(byCat))
	for _, cat := range order {
		cs := byCat[cat]
		reaches := make([]float64, 0, len(cs))
		var totalSpend int64
		top := cs[0]
		for _, c := range cs {
			reaches = append(reaches, c.Reach)
			totalSpend += c.Spend
			if c.Reach > top.Reach {
				top = c
			}
		}
		out = append(out, CategoryBenchmark{
			Category:    cat,
			N:           len(cs),
			MedianReach: Median(reaches),
			TopCampaign: top.Name,
			TopReach:    top.Reach,
			TotalSpend:  totalSpend,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MedianReach > out[j].MedianReach })
	return out
}

// CategoryIndexOf finds c's category in benchmarks and returns reach /
// category median (0 if the category isn't present, e.g. an empty set).
func CategoryIndexOf(c Campaign, benchmarks []CategoryBenchmark) float64 {
	for _, b := range benchmarks {
		if b.Category == c.Category {
			return Index(c.Reach, b.MedianReach)
		}
	}
	return 0
}

type Mover struct {
	Campaign Campaign
	Index    float64
}

// Movers ("Beating the median") are running campaigns with index >= 1.3
// against MED_ALL, sorted by index descending.
func Movers(running []Campaign, medAll float64) []Mover {
	out := []Mover{}
	for _, c := range running {
		idx := Index(c.Reach, medAll)
		if idx >= 1.3 {
			out = append(out, Mover{Campaign: c, Index: idx})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index > out[j].Index })
	return out
}

// Spotlight is the flagged campaign with the highest index against medAll;
// falls back to the first campaign in the slice if none are flagged.
func Spotlight(campaigns []Campaign, medAll float64) (Campaign, bool) {
	var best Campaign
	bestIdx := -1.0
	found := false
	for _, c := range campaigns {
		if !c.IsFlagged() {
			continue
		}
		idx := Index(c.Reach, medAll)
		if idx > bestIdx {
			bestIdx, best, found = idx, c, true
		}
	}
	if found {
		return best, true
	}
	if len(campaigns) > 0 {
		return campaigns[0], true
	}
	return Campaign{}, false
}

type AdTypeCount struct {
	AdType AdType
	Count  int
}

type RegionRollup struct {
	Region      string
	Count       int
	LiveCount   int
	TotalReach  float64
	TotalSpend  int64
	AdTypeSplit []AdTypeCount
}

// RegionRollups aggregates every campaign (not just running) by region.
func RegionRollups(campaigns []Campaign) []RegionRollup {
	byRegion := map[string][]Campaign{}
	order := []string{}
	for _, c := range campaigns {
		if _, ok := byRegion[c.Region]; !ok {
			order = append(order, c.Region)
		}
		byRegion[c.Region] = append(byRegion[c.Region], c)
	}

	out := make([]RegionRollup, 0, len(byRegion))
	for _, region := range order {
		cs := byRegion[region]
		r := RegionRollup{Region: region}
		adCounts := map[AdType]int{}
		for _, c := range cs {
			r.Count++
			if c.Status == StatusLive {
				r.LiveCount++
			}
			r.TotalReach += c.Reach
			r.TotalSpend += c.Spend
			adCounts[c.AdType]++
		}
		for at, n := range adCounts {
			r.AdTypeSplit = append(r.AdTypeSplit, AdTypeCount{AdType: at, Count: n})
		}
		sort.Slice(r.AdTypeSplit, func(i, j int) bool {
			if r.AdTypeSplit[i].Count != r.AdTypeSplit[j].Count {
				return r.AdTypeSplit[i].Count > r.AdTypeSplit[j].Count
			}
			return r.AdTypeSplit[i].AdType < r.AdTypeSplit[j].AdType
		})
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Region < out[j].Region })
	return out
}

// SparklineOffsets are the days-ago marks the overview KPI sparklines and
// spend/reach trend lines sample at.
var SparklineOffsets = [8]int{21, 18, 15, 12, 9, 6, 3, 0}

type SparklinePoint struct {
	DaysAgo   int
	Reach     float64
	Spend     float64
	LiveCount int
}

// Sparkline reconstructs an 8-point historical trend for the running set by
// walking each campaign's own reach curve backward. There's no stored time
// series, so spend is assumed to accrue on the same curve shape as reach —
// the only timing model the campaign carries.
func Sparkline(running []Campaign) []SparklinePoint {
	out := make([]SparklinePoint, len(SparklineOffsets))
	for i, offset := range SparklineOffsets {
		var reach, spend float64
		var live int
		for _, c := range running {
			age := c.DaysRunning - offset
			if age <= 0 {
				continue
			}
			reach += ValueAtAge(c.Reach, c.CurveShape, c.DaysRunning, age)
			spend += ValueAtAge(float64(c.Spend), c.CurveShape, c.DaysRunning, age)
			live++
		}
		out[i] = SparklinePoint{DaysAgo: offset, Reach: reach, Spend: spend, LiveCount: live}
	}
	return out
}

// GrowthPct is the % change from a sparkline's first point to its last.
func GrowthPct(points []SparklinePoint, pick func(SparklinePoint) float64) float64 {
	if len(points) < 2 {
		return 0
	}
	first := pick(points[0])
	last := pick(points[len(points)-1])
	if first == 0 {
		if last == 0 {
			return 0
		}
		return 100
	}
	return (last - first) / first * 100
}

// AnomalyResult is what the anomaly-detection service checks against a
// campaign; Flagged=false means any existing flag should be cleared.
type AnomalyResult struct {
	Flagged bool
	Reason  string
}

// DetectAnomaly applies the four flag conditions in priority order: an
// over-budget campaign is more urgent to surface than a "trending" one, so
// budget conditions are checked first.
func DetectAnomaly(c Campaign, categoryIndex, indexVsAll float64) AnomalyResult {
	pace := Pace(c.Spend, c.Budget)
	switch {
	case pace >= 100:
		return AnomalyResult{true, "Over budget pace"}
	case pace >= 95:
		return AnomalyResult{true, "Budget nearly exhausted"}
	case categoryIndex >= 1.8:
		return AnomalyResult{true, "Reach outlier"}
	case indexVsAll <= 0.3 && c.DaysRunning >= 2:
		return AnomalyResult{true, "Under-delivering"}
	default:
		return AnomalyResult{false, ""}
	}
}
