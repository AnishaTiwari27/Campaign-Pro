package api

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"campaigntrackerpro/internal/data"
	"campaigntrackerpro/internal/models"
)

// ---------------------------------------------------------------------------
// GET /campaigns
// ---------------------------------------------------------------------------

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	filtered := s.Store.Filter(s.parseFilter(r))
	page, limit := parsePagination(r)

	total := len(filtered)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	pageRows := filtered[start:end]

	out := make([]any, len(pageRows))
	for i, c := range pageRows {
		out[i] = c.AsJSON()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": out, "page": page, "limit": limit, "total": total,
	})
}

// ---------------------------------------------------------------------------
// GET /campaigns/{id}
// ---------------------------------------------------------------------------

func (s *Server) handleGetCampaign(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		badRequest(w, "id must be an integer")
		return
	}
	c, ok := s.Store.ByID(id)
	if !ok {
		notFound(w, fmt.Sprintf("no campaign with id %d", id))
		return
	}
	writeJSON(w, http.StatusOK, c.AsJSON())
}

// ---------------------------------------------------------------------------
// GET /campaigns/export
// ---------------------------------------------------------------------------

func (s *Server) handleExportCampaigns(w http.ResponseWriter, r *http.Request) {
	filtered := s.Store.Filter(s.parseFilter(r))

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="campaign-report.csv"`)
	w.WriteHeader(http.StatusOK)

	fmt.Fprint(w, "Brand,Category,Territory,Ad Type,Platform,Start,Reach,Spend\n")
	for _, c := range filtered {
		fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s,%d,%d\n",
			csvEscape(c.Brand), csvEscape(c.Category), csvEscape(c.Territory),
			csvEscape(c.AdType), csvEscape(c.Platform), c.Start.Format("02-Jan-06"),
			c.Reach, c.Spend)
	}
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// ---------------------------------------------------------------------------
// GET /kpis
// ---------------------------------------------------------------------------

func (s *Server) handleKPIs(w http.ResponseWriter, r *http.Request) {
	filtered := s.Store.Filter(s.parseFilter(r))
	today := s.Today

	brandSet := map[string]bool{}
	var totalReach, totalSpend int64
	for _, c := range filtered {
		brandSet[c.Brand] = true
		totalReach += c.Reach
		totalSpend += c.Spend
	}

	countSpark := weeklySpark(filtered, today, func(models.Campaign) int64 { return 1 })
	reachSpark := weeklySpark(filtered, today, func(c models.Campaign) int64 { return c.Reach })
	spendSpark := weeklySpark(filtered, today, func(c models.Campaign) int64 { return c.Spend })
	brandSpark := countSpark // brands-tracked trend approximated by campaign cadence

	countDelta, countUp := deltaFromSpark(countSpark)
	reachDelta, reachUp := deltaFromSpark(reachSpark)
	spendDelta, spendUp := deltaFromSpark(spendSpark)
	brandDelta, brandUp := deltaFromSpark(brandSpark)

	writeJSON(w, http.StatusOK, models.KPISet{
		ActiveCampaigns: models.KPI{Value: int64(len(filtered)), Delta: countDelta, Up: countUp, Spark: countSpark},
		BrandsTracked:   models.KPI{Value: int64(len(brandSet)), Delta: brandDelta, Up: brandUp, Spark: brandSpark},
		EstimatedReach:  models.KPI{Value: totalReach, Delta: reachDelta, Up: reachUp, Spark: reachSpark},
		EstimatedSpend:  models.KPI{Value: totalSpend, Delta: spendDelta, Up: spendUp, Spark: spendSpark},
	})
}

// weeklySpark buckets campaigns into 8 trailing weeks by campaign start date,
// oldest first — mirrors the frontend mock's spark() helper exactly.
func weeklySpark(campaigns []models.Campaign, today time.Time, value func(models.Campaign) int64) []int64 {
	buckets := map[int]int64{}
	for _, c := range campaigns {
		week := int(today.Sub(c.Start).Hours() / 24 / 7)
		buckets[week] += value(c)
	}
	spark := make([]int64, 8)
	for i := 0; i < 8; i++ {
		spark[i] = buckets[7-i]
	}
	return spark
}

// deltaFromSpark compares the last two weekly buckets to produce a
// human-readable trend, e.g. "+12%". A zero previous bucket reads as "+100%"
// growth (or "0%" if both are zero) rather than dividing by zero.
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

// ---------------------------------------------------------------------------
// GET /trend
// ---------------------------------------------------------------------------

func (s *Server) handleTrend(w http.ResponseWriter, r *http.Request) {
	filtered := s.Store.Filter(s.parseFilter(r))

	type bucket struct {
		counts  map[string]int
		earlist time.Time
	}
	buckets := map[string]*bucket{}
	for _, c := range filtered {
		key := c.Start.Format("Jan")
		b, ok := buckets[key]
		if !ok {
			b = &bucket{counts: map[string]int{}, earlist: c.Start}
			buckets[key] = b
		}
		b.counts[c.AdType]++
		if c.Start.Before(b.earlist) {
			b.earlist = c.Start
		}
	}

	months := make([]string, 0, len(buckets))
	for m := range buckets {
		months = append(months, m)
	}
	sort.Slice(months, func(i, j int) bool {
		return buckets[months[i]].earlist.Before(buckets[months[j]].earlist)
	})

	out := make([]map[string]any, 0, len(months))
	for _, m := range months {
		row := map[string]any{"month": m}
		for adType, count := range buckets[m].counts {
			row[adType] = count
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /territories
// ---------------------------------------------------------------------------

func (s *Server) handleTerritories(w http.ResponseWriter, r *http.Request) {
	filtered := s.Store.Filter(s.parseFilter(r))
	counts := map[string]int{}
	for _, c := range filtered {
		counts[c.Territory]++
	}
	out := make([]models.TerritoryCount, 0, len(data.Territories))
	for _, t := range data.Territories {
		out = append(out, models.TerritoryCount{Territory: t, Campaigns: counts[t]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Campaigns > out[j].Campaigns })
	writeJSON(w, http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /benchmark
// ---------------------------------------------------------------------------

func (s *Server) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	filtered := s.Store.Filter(s.parseFilter(r))

	type agg struct {
		brand, category string
		count           int
		reach           int64
		platforms       map[string]bool
		last            time.Time
	}
	byBrand := map[string]*agg{}
	for _, c := range filtered {
		a, ok := byBrand[c.Brand]
		if !ok {
			a = &agg{brand: c.Brand, category: c.Category, platforms: map[string]bool{}, last: c.Start}
			byBrand[c.Brand] = a
		}
		a.count++
		a.reach += c.Reach
		a.platforms[c.Platform] = true
		if c.Start.After(a.last) {
			a.last = c.Start
		}
	}

	rows := make([]*agg, 0, len(byBrand))
	for _, a := range byBrand {
		rows = append(rows, a)
	}

	sortKey := r.URL.Query().Get("sort")
	if sortKey == "" {
		sortKey = "count"
	}
	asc := r.URL.Query().Get("dir") == "asc"

	// Sort ascending on the key, then reverse for descending — keeps the
	// comparator a valid strict order (unlike naively negating "<").
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		switch sortKey {
		case "brand":
			return a.brand < b.brand
		case "reach":
			return a.reach < b.reach
		case "last":
			return a.last.Before(b.last)
		default: // "count"
			return a.count < b.count
		}
	})
	if !asc {
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
	}

	if len(rows) > 10 {
		rows = rows[:10]
	}

	out := make([]models.BenchmarkRow, len(rows))
	for i, a := range rows {
		platforms := make([]string, 0, len(a.platforms))
		for p := range a.platforms {
			platforms = append(platforms, p)
		}
		sort.Strings(platforms)
		out[i] = models.BenchmarkRow{
			Brand: a.brand, Category: a.category, Count: a.count, Reach: a.reach,
			Platforms: platforms, Last: a.last.Format("2006-01-02"),
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /meta
// ---------------------------------------------------------------------------

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, models.Meta{
		Categories:  data.Categories(),
		Territories: data.Territories,
		AdTypes:     data.AdTypes,
	})
}

// ---------------------------------------------------------------------------
// GET /healthz
// ---------------------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
