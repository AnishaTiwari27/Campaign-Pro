// Aggregation queries backing /internal/aggregates/* — GROUP BY in
// Postgres instead of scanning every matching row into Go and summing
// there (that was the real DB bottleneck at scale; see
// docs/ROADMAP.md's Phase A). Each query reuses whereClause/Filter from
// postgres.go so filter semantics can't drift between the paginated list,
// the CSV export, and these.
package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"campaigntrackerpro/services/campaigns/internal/models"
)

// KPITotals returns the full-filtered-set totals — one row, not one row
// per campaign.
func (s *Store) KPITotals(ctx context.Context, f Filter) (models.KPITotals, error) {
	where, args := whereClause(f)
	sql := fmt.Sprintf(`
		SELECT count(*), count(DISTINCT subject_name), coalesce(sum(reach), 0), coalesce(sum(spend_paise), 0)
		FROM campaigns.campaigns WHERE %s`, where)

	var t models.KPITotals
	var spendPaise int64
	err := s.pool.QueryRow(ctx, sql, args...).Scan(&t.Count, &t.DistinctSubjects, &t.Reach, &spendPaise)
	t.Spend = spendPaise / 100
	return t, err
}

// WeeklyBuckets returns per-week totals, keyed by how many weeks before
// today each campaign's start_date falls — the same "week := int(today.Sub(start).Hours()/24/7)"
// bucketing the old in-memory version used, just computed once in SQL
// instead of once per row in Go. Only weeks 0-7 are read back by the
// caller (an 8-point sparkline); other weeks come back too but are
// harmless — GROUP BY on a handful of week numbers is cheap regardless of
// how many campaigns fall outside that window.
func (s *Store) WeeklyBuckets(ctx context.Context, f Filter, today time.Time) ([]models.WeeklyBucket, error) {
	where, args := whereClause(f)
	args = append(args, today)
	todayIdx := len(args)

	sql := fmt.Sprintf(`
		SELECT floor(extract(epoch FROM ($%d::timestamptz - start_date)) / 604800)::int AS weeks_ago,
		       count(*), coalesce(sum(reach), 0), coalesce(sum(spend_paise), 0)
		FROM campaigns.campaigns WHERE %s
		GROUP BY weeks_ago
		HAVING floor(extract(epoch FROM ($%d::timestamptz - start_date)) / 604800)::int BETWEEN 0 AND 7`,
		todayIdx, where, todayIdx)

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.WeeklyBucket
	for rows.Next() {
		var b models.WeeklyBucket
		var spendPaise int64
		if err := rows.Scan(&b.WeeksAgo, &b.Count, &b.Reach, &spendPaise); err != nil {
			return nil, err
		}
		b.Spend = spendPaise / 100
		out = append(out, b)
	}
	return out, rows.Err()
}

// TrendBuckets returns one row per (month, ad type) combination present in
// the filtered set — enough for the "activity by format" chart, without
// pulling every campaign to group them client-side.
func (s *Store) TrendBuckets(ctx context.Context, f Filter) ([]models.TrendBucket, error) {
	where, args := whereClause(f)
	sql := fmt.Sprintf(`
		SELECT to_char(start_date, 'Mon') AS month, to_char(date_trunc('month', start_date), 'YYYY-MM-DD') AS month_start,
		       ad_type, count(*)
		FROM campaigns.campaigns WHERE %s
		GROUP BY month, date_trunc('month', start_date), ad_type
		ORDER BY date_trunc('month', start_date)`, where)

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.TrendBucket
	for rows.Next() {
		var b models.TrendBucket
		if err := rows.Scan(&b.Month, &b.MonthStart, &b.AdType, &b.Count); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// RegionCounts returns campaign counts per region — only regions with at
// least one matching campaign; zero-filling the rest of the canonical
// list is analytics-service's job (aggregate.Regions), same split of
// responsibility as before.
func (s *Store) RegionCounts(ctx context.Context, f Filter) ([]models.RegionCount, error) {
	where, args := whereClause(f)
	sql := fmt.Sprintf(`SELECT region, count(*) FROM campaigns.campaigns WHERE %s GROUP BY region`, where)

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.RegionCount
	for rows.Next() {
		var c models.RegionCount
		if err := rows.Scan(&c.Region, &c.Campaigns); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// benchmarkSortColumn maps a sort key to a literal ORDER BY expression —
// a Go-side whitelist switch, never the raw sortKey string concatenated
// into SQL, so an unrecognized or hostile sort value can't reach the
// query (falls back to the same "count" default the old Go-side sort used).
func benchmarkSortColumn(sortKey string) string {
	switch sortKey {
	case "subject":
		return "subject_name"
	case "reach":
		return "sum(reach)"
	case "last":
		return "max(start_date)"
	default:
		return "count(*)"
	}
}

// BenchmarkRows returns the top 10 subjects by sortKey/asc — grouped,
// sorted, and capped in SQL, not recomputed from raw rows downstream.
// Each result also carries up to 8 recent benchmark_snapshots points via
// one follow-up query (attachBenchmarkTrends), not N.
func (s *Store) BenchmarkRows(ctx context.Context, f Filter, sortKey string, asc bool) ([]models.BenchmarkAgg, error) {
	where, args := whereClause(f)
	dir := "DESC"
	if asc {
		dir = "ASC"
	}
	sql := fmt.Sprintf(`
		SELECT subject_name, subject_type, subject_category, count(*), coalesce(sum(reach), 0),
		       array_agg(DISTINCT platform ORDER BY platform), to_char(max(start_date), 'YYYY-MM-DD')
		FROM campaigns.campaigns WHERE %s
		GROUP BY subject_name, subject_type, subject_category
		ORDER BY %s %s
		LIMIT 10`, where, benchmarkSortColumn(sortKey), dir)

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.BenchmarkAgg
	for rows.Next() {
		var b models.BenchmarkAgg
		if err := rows.Scan(&b.Subject, &b.SubjectType, &b.Category, &b.Count, &b.Reach, &b.Platforms, &b.Last); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := s.attachBenchmarkTrends(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// attachBenchmarkTrends fills in each row's Trend field with up to 8
// recent campaigns.benchmark_snapshots points (oldest -> newest) for
// exactly the subjects BenchmarkRows already selected — one query here,
// not one per subject. A window function (row_number partitioned per
// subject) picks the 8 most recent snapshots per subject in SQL, not by
// pulling every snapshot ever taken and trimming in Go.
func (s *Store) attachBenchmarkTrends(ctx context.Context, rows []models.BenchmarkAgg) error {
	if len(rows) == 0 {
		return nil
	}

	var args []any
	placeholders := make([]string, len(rows))
	for i, b := range rows {
		args = append(args, b.Subject, b.SubjectType)
		placeholders[i] = fmt.Sprintf("($%d, $%d)", len(args)-1, len(args))
	}

	sql := fmt.Sprintf(`
		WITH recent AS (
			SELECT subject_name, subject_type, total_reach, snapshotted_at,
			       row_number() OVER (PARTITION BY subject_name, subject_type ORDER BY snapshotted_at DESC) AS rn
			FROM campaigns.benchmark_snapshots
			WHERE (subject_name, subject_type) IN (%s)
		)
		SELECT subject_name, subject_type, total_reach
		FROM recent WHERE rn <= 8
		ORDER BY subject_name, subject_type, snapshotted_at ASC`, strings.Join(placeholders, ", "))

	trendRows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer trendRows.Close()

	byKey := make(map[string][]int64, len(rows))
	for trendRows.Next() {
		var name, subjType string
		var reach int64
		if err := trendRows.Scan(&name, &subjType, &reach); err != nil {
			return err
		}
		byKey[name+"\x00"+subjType] = append(byKey[name+"\x00"+subjType], reach)
	}
	if err := trendRows.Err(); err != nil {
		return err
	}

	for i := range rows {
		rows[i].Trend = byKey[rows[i].Subject+"\x00"+rows[i].SubjectType]
	}
	return nil
}

// SnapshotBenchmark records one row per subject across the *whole* table
// (not the top-10 cap BenchmarkRows uses — trending needs full history to
// re-rank later) — called by the BENCHMARK_SNAPSHOT_ENABLED ticker in
// cmd/server/main.go.
func (s *Store) SnapshotBenchmark(ctx context.Context, today time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO campaigns.benchmark_snapshots (subject_name, subject_type, subject_category, campaign_count, total_reach, snapshotted_at)
		SELECT subject_name, subject_type, subject_category, count(*), coalesce(sum(reach), 0), $1
		FROM campaigns.campaigns
		GROUP BY subject_name, subject_type, subject_category`, today)
	return err
}
