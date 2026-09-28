// Anomaly detection queries backing GET /api/v1/campaigns/anomalies — see
// models/anomalies.go for why this is peer-outlier + stale-category
// detection, not literal time-series spike/cliff math a snapshot-per-row
// schema can't honestly support. Both queries here are deliberately
// unfiltered by the caller's category/region/etc. filters (no Filter
// param, unlike every other method in this package) — a global "what
// needs attention" signal, not a view of the current filtered slice.
package store

import (
	"context"
	"fmt"
	"time"

	"campaigntrackerpro/services/campaigns/internal/models"
)

// AnomalyReport runs both detection queries and assembles the combined
// response. Two small, single-purpose queries rather than one bigger one
// — they answer different questions (one row vs. one category) and don't
// share a WHERE clause worth factoring out.
func (s *Store) AnomalyReport(ctx context.Context, today time.Time) (models.AnomalyReport, error) {
	campaigns, err := s.peerOutliers(ctx)
	if err != nil {
		return models.AnomalyReport{}, fmt.Errorf("peer outliers: %w", err)
	}
	stale, err := s.staleCategories(ctx, today)
	if err != nil {
		return models.AnomalyReport{}, fmt.Errorf("stale categories: %w", err)
	}
	return models.AnomalyReport{Campaigns: campaigns, StaleCategories: stale}, nil
}

// peerOutliers flags campaigns whose reach or spend is far from their own
// category's median — a CTE computes each category's median reach/spend
// and peer count once, joined back to every row in that category so the
// comparison is against the whole category's history, not whatever the
// request happens to have filtered to.
func (s *Store) peerOutliers(ctx context.Context) ([]models.CampaignAnomaly, error) {
	sql := `
		WITH peer_stats AS (
			SELECT subject_category,
			       percentile_cont(0.5) WITHIN GROUP (ORDER BY reach) AS median_reach,
			       percentile_cont(0.5) WITHIN GROUP (ORDER BY spend_paise) AS median_spend,
			       count(*) AS peer_count
			FROM campaigns.campaigns
			GROUP BY subject_category
		)
		SELECT c.id, c.subject_name, c.subject_category, c.reach, c.spend_paise,
		       p.median_reach, p.median_spend
		FROM campaigns.campaigns c
		JOIN peer_stats p ON p.subject_category = c.subject_category
		WHERE p.peer_count >= $1
		  AND (
		    (p.median_reach > 0 AND (c.reach > $2 * p.median_reach OR c.reach < $3 * p.median_reach))
		    OR (p.median_spend > 0 AND (c.spend_paise > $2 * p.median_spend OR c.spend_paise < $3 * p.median_spend))
		  )
		ORDER BY c.start_date DESC
		LIMIT 100`

	rows, err := s.pool.Query(ctx, sql, models.AnomalyMinPeers, models.AnomalyRatioHigh, models.AnomalyRatioLow)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.CampaignAnomaly
	for rows.Next() {
		var a models.CampaignAnomaly
		var reach, spendPaise int64
		var medianReach, medianSpend float64
		if err := rows.Scan(&a.ID, &a.Subject, &a.Category, &reach, &spendPaise, &medianReach, &medianSpend); err != nil {
			return nil, err
		}
		a.Reason = models.AnomalyReason(reach, int64(medianReach), spendPaise, int64(medianSpend))
		out = append(out, a)
	}
	return out, rows.Err()
}

// staleCategories finds categories with real history that have gone
// quiet — no campaign started in the last StaleCategoryDays days, despite
// having enough total history (StaleCategoryMinHistory) to mean something.
// The cutoff is computed in Go, not as an interval expression bound
// through a placeholder — same reasoning whereClause's Days filter
// already computes its own cutoff in Go (postgres.go) rather than passing
// a raw day-count into an interval expression, which is ambiguous for
// Postgres to type-infer through a bind parameter.
func (s *Store) staleCategories(ctx context.Context, today time.Time) ([]models.StaleCategory, error) {
	cutoff := today.AddDate(0, 0, -models.StaleCategoryDays)
	sql := `
		SELECT subject_category, max(start_date)
		FROM campaigns.campaigns
		GROUP BY subject_category
		HAVING max(start_date) < $1
		   AND count(*) >= $2
		ORDER BY max(start_date) ASC`

	rows, err := s.pool.Query(ctx, sql, cutoff, models.StaleCategoryMinHistory)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.StaleCategory
	for rows.Next() {
		var category string
		var lastActivity time.Time
		if err := rows.Scan(&category, &lastActivity); err != nil {
			return nil, err
		}
		out = append(out, models.StaleCategory{
			Category:          category,
			LastActivity:      lastActivity.Format("2006-01-02"),
			DaysSinceActivity: int(today.Sub(lastActivity).Hours() / 24),
		})
	}
	return out, rows.Err()
}
