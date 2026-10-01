package service

import (
	"context"
	"log/slog"

	"campaigntrackerpro/internal/domain"
)

// AnomalyDetector re-evaluates every campaign's flag on each run — on a
// ticker, and on demand (e.g. right after a spend/budget-affecting change).
// It reports which campaigns transitioned from unflagged to flagged, so the
// caller can fire on_flag reports for exactly those.
type AnomalyDetector struct {
	Campaigns *Campaigns
	Logger    *slog.Logger
}

func NewAnomalyDetector(c *Campaigns, logger *slog.Logger) *AnomalyDetector {
	return &AnomalyDetector{Campaigns: c, Logger: logger}
}

// Run scans every campaign, applies domain.DetectAnomaly, and writes any
// change (new flag, changed reason, or cleared flag) plus its audit event.
// Returns the campaigns that newly became flagged this run.
func (d *AnomalyDetector) Run(ctx context.Context) ([]domain.Campaign, error) {
	all, err := d.Campaigns.Store.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	running := domain.Running(all)
	medAll := domain.MedianReach(running)
	benchmarks := domain.CategoryBenchmarks(running)

	var newlyFlagged []domain.Campaign
	for _, c := range all {
		if !c.IsRunning() {
			continue
		}
		categoryIndex := domain.CategoryIndexOf(c, benchmarks)
		indexVsAll := domain.Index(c.Reach, medAll)
		result := domain.DetectAnomaly(c, categoryIndex, indexVsAll)

		wasFlagged := c.IsFlagged()
		switch {
		case result.Flagged && c.FlagReason != result.Reason:
			updated, err := d.Campaigns.Store.UpdateCampaignFlag(ctx, c.ID, result.Reason)
			if err != nil {
				return nil, err
			}
			if _, err := d.Campaigns.Store.CreateAuditEvent(ctx, c.ID, "Anomaly detection", result.Reason, "alert"); err != nil {
				return nil, err
			}
			if !wasFlagged {
				newlyFlagged = append(newlyFlagged, updated)
			}
			d.Logger.Info("anomaly flagged", "campaign", c.ID, "reason", result.Reason)
		case !result.Flagged && wasFlagged:
			if _, err := d.Campaigns.Store.UpdateCampaignFlag(ctx, c.ID, ""); err != nil {
				return nil, err
			}
			if _, err := d.Campaigns.Store.CreateAuditEvent(ctx, c.ID, "Anomaly detection", "Flag cleared", "sys"); err != nil {
				return nil, err
			}
			d.Logger.Info("anomaly cleared", "campaign", c.ID)
		}
	}
	return newlyFlagged, nil
}
