package store

import (
	"context"
	"encoding/json"
	"errors"

	"campaigntrackerpro/internal/db/gen"
	"campaigntrackerpro/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListReports(ctx context.Context) ([]domain.Report, error) {
	rs, err := s.Queries.ListReports(ctx)
	if err != nil {
		return nil, err
	}
	return toDomainReports(rs), nil
}

func (s *Store) ListEnabledReportsByCadence(ctx context.Context, cadence string) ([]domain.Report, error) {
	rs, err := s.Queries.ListEnabledReportsByCadence(ctx, gen.ReportCadenceT(cadence))
	if err != nil {
		return nil, err
	}
	return toDomainReports(rs), nil
}

func (s *Store) GetReport(ctx context.Context, id string) (domain.Report, error) {
	uid, err := uuidParam(id)
	if err != nil {
		return domain.Report{}, ErrNotFound
	}
	r, err := s.Queries.GetReport(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Report{}, ErrNotFound
		}
		return domain.Report{}, err
	}
	return toDomainReport(r), nil
}

func (s *Store) CreateReport(ctx context.Context, r domain.Report) (domain.Report, error) {
	scope, err := json.Marshal(r.ScopeFilters)
	if err != nil {
		return domain.Report{}, err
	}
	created, err := s.Queries.CreateReport(ctx, gen.CreateReportParams{
		Name:         r.Name,
		Enabled:      r.Enabled,
		Cadence:      gen.ReportCadenceT(r.Cadence),
		Recipients:   r.Recipients,
		ScopeFilters: scope,
		ScopeLabel:   r.ScopeLabel,
		Columns:      r.Columns,
	})
	if err != nil {
		return domain.Report{}, err
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

func (s *Store) UpdateReport(ctx context.Context, id string, u ReportUpdate) (domain.Report, error) {
	uid, err := uuidParam(id)
	if err != nil {
		return domain.Report{}, ErrNotFound
	}
	arg := gen.UpdateReportParams{
		ID:         uid,
		Recipients: u.Recipients,
		Columns:    u.Columns,
	}
	if u.Name != nil {
		arg.Name = textParam(*u.Name)
	}
	if u.Enabled != nil {
		arg.Enabled = pgxBool(*u.Enabled)
	}
	if u.Cadence != nil {
		arg.Cadence = gen.NullReportCadenceT{ReportCadenceT: gen.ReportCadenceT(*u.Cadence), Valid: true}
	}
	if u.ScopeLabel != nil {
		arg.ScopeLabel = textParam(*u.ScopeLabel)
	}
	if u.ScopeFilters != nil {
		b, err := json.Marshal(u.ScopeFilters)
		if err != nil {
			return domain.Report{}, err
		}
		arg.ScopeFilters = b
	}
	r, err := s.Queries.UpdateReport(ctx, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Report{}, ErrNotFound
		}
		return domain.Report{}, err
	}
	return toDomainReport(r), nil
}

func (s *Store) DeleteReport(ctx context.Context, id string) error {
	uid, err := uuidParam(id)
	if err != nil {
		return ErrNotFound
	}
	return s.Queries.DeleteReport(ctx, uid)
}
