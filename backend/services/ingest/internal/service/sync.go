// Package service runs a sync: fetch from a provider, upsert each row,
// and record what happened.
package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"campaigntrackerpro/services/ingest"
	"campaigntrackerpro/services/ingest/internal/store"
)

type Sync struct {
	store  *store.Store
	logger *slog.Logger
}

func New(st *store.Store, logger *slog.Logger) *Sync {
	return &Sync{store: st, logger: logger}
}

// Run pulls from one provider and writes what it finds.
//
// A row that fails is counted and skipped rather than aborting the
// sync. One malformed campaign in an account of four hundred should not
// stop the other three hundred and ninety-nine from updating, and the
// failure count in the run history is what makes the skip visible
// rather than silent.
func (s *Sync) Run(ctx context.Context, p ingest.Provider, since time.Time) (ingest.Result, error) {
	result := ingest.Result{Source: p.Source()}

	runID, err := s.store.StartRun(ctx, p.Source())
	if err != nil {
		return result, err
	}
	// The run is closed on every path, including a panic unwinding
	// through here: a run left open forever reads as "still running"
	// and is worse than one recorded as failed.
	defer func() {
		if err := s.store.FinishRun(ctx, runID, result); err != nil {
			s.logger.Error("could not close ingest run", "run", runID, "err", err)
		}
	}()

	rows, err := p.Fetch(ctx, since)
	if err != nil {
		result.Err = err
		if errors.Is(err, ingest.ErrNotConfigured) {
			// Not an error worth alarming about: it means nobody has set
			// this provider up, which is the normal state of a provider
			// nobody uses.
			s.logger.Info("ingest skipped: provider not configured", "source", p.Source())
			return result, err
		}
		s.logger.Error("ingest fetch failed", "source", p.Source(), "err", err)
		return result, err
	}

	result.Seen = len(rows)
	for _, row := range rows {
		created, err := s.store.Upsert(ctx, p.Source(), row)
		if err != nil {
			result.Failed++
			s.logger.Error("ingest row failed", "source", p.Source(), "external_id", row.ExternalID, "err", err)
			continue
		}
		if created {
			result.Created++
		} else {
			result.Updated++
		}
	}
	s.logger.Info("ingest finished", "source", p.Source(),
		"seen", result.Seen, "created", result.Created, "updated", result.Updated, "failed", result.Failed)
	return result, nil
}

// Runs is the history, most recent first.
func (s *Sync) Runs(ctx context.Context) ([]store.Run, error) {
	return s.store.ListRuns(ctx)
}
