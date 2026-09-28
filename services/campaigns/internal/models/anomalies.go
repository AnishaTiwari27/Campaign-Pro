// Anomaly detection response shapes, backing GET /api/v1/campaigns/anomalies
// (services/campaigns/internal/store/anomalies.go's AnomalyReport). See
// docs/ROADMAP.md's Phase B for why this is peer-outlier + stale-category
// detection rather than literal "spend spike / reach cliff": a campaign
// row here is an immutable snapshot (reach/spend set once at creation,
// never incremented), so there's no per-campaign time series to spike or
// dip against — what's real is how a campaign compares to its category
// peers right now, and whether a category that normally has activity has
// gone quiet.
package models

import "fmt"

// AnomalyRatioHigh/Low and AnomalyMinPeers are deliberately conservative —
// tightening/loosening any of these is a one-line change, same precedent
// PacingOf's thresholds set. Exported (unlike PacingOf's own thresholds)
// because store/anomalies.go's SQL needs them as query parameters, not
// just this file's own Go logic. AnomalyMinPeers guards against flagging
// a campaign in a category that's just naturally small (a category of 2
// campaigns has no meaningful "median" to be an outlier against).
const (
	AnomalyRatioHigh = 2.5
	AnomalyRatioLow  = 0.4
	AnomalyMinPeers  = 4

	// StaleCategoryDays/MinHistory: a category needs real history (at
	// least this many campaigns ever) before "nothing new lately" is
	// worth flagging — a category with 1-2 campaigns total going quiet
	// isn't a signal, it's just how small it always was.
	StaleCategoryDays       = 14
	StaleCategoryMinHistory = 3
)

// CampaignAnomaly is one campaign whose reach or spend is a statistical
// outlier against its own category's peers.
type CampaignAnomaly struct {
	ID       int64  `json:"id"`
	Subject  string `json:"subject"`
	Category string `json:"category"`
	Reason   string `json:"reason"`
}

// StaleCategory is a category with real history that's gone quiet — no
// new campaign started in staleCategoryDays.
type StaleCategory struct {
	Category          string `json:"category"`
	LastActivity      string `json:"lastActivity"` // YYYY-MM-DD
	DaysSinceActivity int    `json:"daysSinceActivity"`
}

// AnomalyReport is the full GET /api/v1/campaigns/anomalies response —
// deliberately unfiltered by the caller's category/region/etc. filters
// (see store.AnomalyReport), a global "what needs attention" signal
// rather than a view of whatever's currently on screen.
type AnomalyReport struct {
	Campaigns       []CampaignAnomaly `json:"campaigns"`
	StaleCategories []StaleCategory   `json:"staleCategories"`
}

// AnomalyReason picks whichever of reach/spend deviates furthest from its
// category median (as a ratio, high or low) and renders a short
// human-readable explanation, e.g. "reach 3.1x category median". Pure and
// unit-tested without a database — the SQL query (store/anomalies.go)
// does the filtering; this only explains a row already known to qualify.
// medianReach/medianSpend of 0 mean "no usable median" (shouldn't happen
// for a row the SQL query already flagged, but guarded rather than risking
// a divide-by-zero on a data edge case).
func AnomalyReason(reach, medianReach, spendPaise, medianSpend int64) string {
	type candidate struct {
		metric string
		ratio  float64
	}
	var best *candidate
	consider := func(metric string, ratio float64) {
		if best == nil || absDiff(ratio, 1) > absDiff(best.ratio, 1) {
			best = &candidate{metric: metric, ratio: ratio}
		}
	}
	if medianReach > 0 {
		consider("reach", float64(reach)/float64(medianReach))
	}
	if medianSpend > 0 {
		consider("spend", float64(spendPaise)/float64(medianSpend))
	}
	if best == nil {
		return "" // no usable peer median — shouldn't reach here in practice
	}
	return fmt.Sprintf("%s %.1fx category median", best.metric, best.ratio)
}

func absDiff(a, b float64) float64 {
	d := a - b
	if d < 0 {
		return -d
	}
	return d
}
