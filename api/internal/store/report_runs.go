package store

import (
	"context"
	"errors"

	"campaigntrackerpro/internal/db/gen"
	"campaigntrackerpro/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListReportRuns(ctx context.Context, reportID string, limit int32) ([]domain.ReportRun, error) {
	uid, err := uuidParam(reportID)
	if err != nil {
		return nil, ErrNotFound
	}
	rs, err := s.Queries.ListReportRuns(ctx, gen.ListReportRunsParams{ReportID: uid, Limit: limit})
	if err != nil {
		return nil, err
	}
	return toDomainReportRuns(rs), nil
}

func (s *Store) GetLastReportRun(ctx context.Context, reportID string) (domain.ReportRun, error) {
	uid, err := uuidParam(reportID)
	if err != nil {
		return domain.ReportRun{}, ErrNotFound
	}
	r, err := s.Queries.GetLastReportRun(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ReportRun{}, ErrNotFound
		}
		return domain.ReportRun{}, err
	}
	return toDomainReportRun(r), nil
}

func (s *Store) CreateReportRun(ctx context.Context, reportID string, result string, rowCount int) (domain.ReportRun, error) {
	uid, err := uuidParam(reportID)
	if err != nil {
		return domain.ReportRun{}, ErrNotFound
	}
	r, err := s.Queries.CreateReportRun(ctx, gen.CreateReportRunParams{
		ReportID: uid, Result: gen.ReportResultT(result), RowCount: int32(rowCount),
	})
	if err != nil {
		return domain.ReportRun{}, err
	}
	return toDomainReportRun(r), nil
}
