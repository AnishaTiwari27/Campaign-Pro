// Package worker runs scheduled reports and anomaly detection in the same
// binary as the API — a goroutine + ticker is enough at this scale, per the
// spec's own "background worker ... in the same binary" instruction.
package worker

import (
	"context"
	"log/slog"
	"time"

	"campaigntrackerpro/internal/service"
)

type Worker struct {
	Anomaly  *service.AnomalyDetector
	Reports  *service.Reports
	Logger   *slog.Logger
	Interval time.Duration
}

func New(anomaly *service.AnomalyDetector, reports *service.Reports, logger *slog.Logger, interval time.Duration) *Worker {
	return &Worker{Anomaly: anomaly, Reports: reports, Logger: logger, Interval: interval}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	w.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	newlyFlagged, err := w.Anomaly.Run(ctx)
	if err != nil {
		w.Logger.Error("anomaly detection failed", "err", err)
	} else if len(newlyFlagged) > 0 {
		w.fireOnFlagReports(ctx)
	}

	due, err := w.Reports.DueScheduledReports(ctx, time.Now())
	if err != nil {
		w.Logger.Error("due-reports lookup failed", "err", err)
		return
	}
	for _, rep := range due {
		if _, err := w.Reports.Run(ctx, rep.ID); err != nil {
			w.Logger.Error("scheduled report run failed", "report", rep.ID, "err", err)
		} else {
			w.Logger.Info("scheduled report sent", "report", rep.ID, "cadence", rep.Cadence)
		}
	}
}

// fireOnFlagReports runs every enabled on_flag report immediately after a
// campaign newly flags — comfortably inside the spec's "within 15 min"
// bound, since the worker's tick interval is far shorter than that.
func (w *Worker) fireOnFlagReports(ctx context.Context) {
	reports, err := w.Reports.OnFlagReports(ctx)
	if err != nil {
		w.Logger.Error("on-flag reports lookup failed", "err", err)
		return
	}
	for _, rep := range reports {
		if _, err := w.Reports.Run(ctx, rep.ID); err != nil {
			w.Logger.Error("on-flag report run failed", "report", rep.ID, "err", err)
		} else {
			w.Logger.Info("on-flag report sent", "report", rep.ID)
		}
	}
}
