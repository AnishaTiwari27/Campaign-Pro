package store

import (
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/reports"
	"context"
	"encoding/json"
	"errors"

	"campaigntrackerpro/db/gen"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListReports(ctx context.Context) ([]reports.Report, error) {
	rs, err := s.db.Queries.ListReports(ctx)
	if err != nil {
		return nil, err
	}
	return toDomainReports(rs), nil
}

func (s *Store) ListEnabledReportsByCadence(ctx context.Context, cadence string) ([]reports.Report, error) {
	rs, err := s.db.Queries.ListEnabledReportsByCadence(ctx, gen.ReportCadenceT(cadence))
	if err != nil {
		return nil, err
	}
	return toDomainReports(rs), nil
}

func (s *Store) GetReport(ctx context.Context, id string) (reports.Report, error) {
	uid, err := database.UuidParam(id)
	if err != nil {
		return reports.Report{}, database.ErrNotFound
	}
	r, err := s.db.Queries.GetReport(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return reports.Report{}, database.ErrNotFound
		}
		return reports.Report{}, err
	}
	return toDomainReport(r), nil
}

func (s *Store) CreateReport(ctx context.Context, r reports.Report) (reports.Report, error) {
	scope, err := json.Marshal(r.ScopeFilters)
	if err != nil {
		return reports.Report{}, err
	}
	created, err := s.db.Queries.CreateReport(ctx, gen.CreateReportParams{
		Name:         r.Name,
		Enabled:      r.Enabled,
		Cadence:      gen.ReportCadenceT(r.Cadence),
		Recipients:   r.Recipients,
		ScopeFilters: scope,
		ScopeLabel:   r.ScopeLabel,
		Columns:      r.Columns,
	})
	if err != nil {
		return reports.Report{}, err
	}
	return toDomainReport(created), nil
}

// ReportUpdate carries only the fields a PATCH actually sent; nil/empty
// means "leave unchanged" (empty slices are indistinguishable from "not
// sent" here, which is acceptable since every editable list field always
// has at least one element in this app's UI).
type ReportUpdate struct {
	Name         *string
	Enabled      *bool
	Cadence      *string
	Recipients   []string
	ScopeFilters map[string]any
	ScopeLabel   *string
	Columns      []string
}

func (s *Store) UpdateReport(ctx context.Context, id string, u ReportUpdate) (reports.Report, error) {
	uid, err := database.UuidParam(id)
	if err != nil {
		return reports.Report{}, database.ErrNotFound
	}
	arg := gen.UpdateReportParams{
		ID:         uid,
		Recipients: u.Recipients,
		Columns:    u.Columns,
	}
	if u.Name != nil {
		arg.Name = database.TextParam(*u.Name)
	}
	if u.Enabled != nil {
		arg.Enabled = database.PgxBool(*u.Enabled)
	}
	if u.Cadence != nil {
		arg.Cadence = gen.NullReportCadenceT{ReportCadenceT: gen.ReportCadenceT(*u.Cadence), Valid: true}
	}
	if u.ScopeLabel != nil {
		arg.ScopeLabel = database.TextParam(*u.ScopeLabel)
	}
	if u.ScopeFilters != nil {
		b, err := json.Marshal(u.ScopeFilters)
		if err != nil {
			return reports.Report{}, err
		}
		arg.ScopeFilters = b
	}
	r, err := s.db.Queries.UpdateReport(ctx, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return reports.Report{}, database.ErrNotFound
		}
		return reports.Report{}, err
	}
	return toDomainReport(r), nil
}

func (s *Store) DeleteReport(ctx context.Context, id string) error {
	uid, err := database.UuidParam(id)
	if err != nil {
		return database.ErrNotFound
	}
	return s.db.Queries.DeleteReport(ctx, uid)
}
