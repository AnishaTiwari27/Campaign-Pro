package store

import (
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/reports"
	"context"
	"errors"

	"campaigntrackerpro/db/gen"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListReportRuns(ctx context.Context, reportID string, limit int32) ([]reports.ReportRun, error) {
	uid, err := database.UuidParam(reportID)
	if err != nil {
		return nil, database.ErrNotFound
	}
	rs, err := s.db.Queries.ListReportRuns(ctx, gen.ListReportRunsParams{ReportID: uid, Limit: limit})
	if err != nil {
		return nil, err
	}
	return toDomainReportRuns(rs), nil
}

func (s *Store) GetLastReportRun(ctx context.Context, reportID string) (reports.ReportRun, error) {
	uid, err := database.UuidParam(reportID)
	if err != nil {
		return reports.ReportRun{}, database.ErrNotFound
	}
	r, err := s.db.Queries.GetLastReportRun(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return reports.ReportRun{}, database.ErrNotFound
		}
		return reports.ReportRun{}, err
	}
	return toDomainReportRun(r), nil
}

func (s *Store) CreateReportRun(ctx context.Context, reportID string, result string, rowCount int) (reports.ReportRun, error) {
	uid, err := database.UuidParam(reportID)
	if err != nil {
		return reports.ReportRun{}, database.ErrNotFound
	}
	r, err := s.db.Queries.CreateReportRun(ctx, gen.CreateReportRunParams{
		ReportID: uid, Result: gen.ReportResultT(result), RowCount: int32(rowCount),
	})
	if err != nil {
		return reports.ReportRun{}, err
	}
	return toDomainReportRun(r), nil
}
