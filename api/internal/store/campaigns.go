package store

import (
	"context"
	"errors"

	"campaigntrackerpro/internal/db/gen"
	"campaigntrackerpro/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrNotFound = errors.New("not found")

// CampaignFilter mirrors every /campaigns query filter; nil fields mean
// "no constraint."
type CampaignFilter struct {
	SubjectType *string
	Category    *string
	Region      *string
	AdType      *string
	Status      *string
	Approval    *string
	Search      *string
}

func nullText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func (s *Store) ListCampaigns(ctx context.Context) ([]domain.Campaign, error) {
	cs, err := s.Queries.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	return toDomainCampaigns(cs), nil
}

func (s *Store) ListCampaignsFiltered(ctx context.Context, f CampaignFilter) ([]domain.Campaign, error) {
	arg := gen.ListCampaignsFilteredParams{
		Category: nullText(f.Category),
		Region:   nullText(f.Region),
		Search:   nullText(f.Search),
	}
	if f.SubjectType != nil && *f.SubjectType != "" {
		arg.SubjectType = gen.NullSubjectTypeT{SubjectTypeT: gen.SubjectTypeT(*f.SubjectType), Valid: true}
	}
	if f.AdType != nil && *f.AdType != "" {
		arg.AdType = gen.NullAdTypeT{AdTypeT: gen.AdTypeT(*f.AdType), Valid: true}
	}
	if f.Status != nil && *f.Status != "" {
		arg.Status = gen.NullCampaignStatusT{CampaignStatusT: gen.CampaignStatusT(*f.Status), Valid: true}
	}
	if f.Approval != nil && *f.Approval != "" {
		arg.Approval = gen.NullApprovalStatusT{ApprovalStatusT: gen.ApprovalStatusT(*f.Approval), Valid: true}
	}
	cs, err := s.Queries.ListCampaignsFiltered(ctx, arg)
	if err != nil {
		return nil, err
	}
	return toDomainCampaigns(cs), nil
}

func (s *Store) GetCampaign(ctx context.Context, id string) (domain.Campaign, error) {
	c, err := s.Queries.GetCampaign(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Campaign{}, ErrNotFound
		}
		return domain.Campaign{}, err
	}
	return toDomainCampaign(c), nil
}

func (s *Store) ListRunningCampaigns(ctx context.Context) ([]domain.Campaign, error) {
	cs, err := s.Queries.ListRunningCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	return toDomainCampaigns(cs), nil
}

func (s *Store) ListPendingCampaigns(ctx context.Context) ([]domain.Campaign, error) {
	cs, err := s.Queries.ListPendingCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	return toDomainCampaigns(cs), nil
}

func (s *Store) ListFlaggedCampaigns(ctx context.Context) ([]domain.Campaign, error) {
	cs, err := s.Queries.ListFlaggedCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	return toDomainCampaigns(cs), nil
}

func (s *Store) CreateCampaign(ctx context.Context, c domain.Campaign) (domain.Campaign, error) {
	created, err := s.Queries.CreateCampaign(ctx, gen.CreateCampaignParams{
		ID:          c.ID,
		Name:        c.Name,
		SubjectType: gen.SubjectTypeT(c.SubjectType),
		Role:        textParam(c.Role),
		Initials:    c.Initials,
		Category:    c.Category,
		Region:      c.Region,
		AdType:      gen.AdTypeT(c.AdType),
		Platform:    c.Platform,
		Status:      gen.CampaignStatusT(c.Status),
		DaysRunning: int32(c.DaysRunning),
		Reach:       c.Reach,
		Spend:       c.Spend,
		Budget:      c.Budget,
		Frequency:   c.Frequency,
		Approval:    gen.ApprovalStatusT(c.Approval),
		CurveShape:  gen.CurveShapeT(c.CurveShape),
		FlagReason:  textParam(c.FlagReason),
	})
	if err != nil {
		return domain.Campaign{}, err
	}
	return toDomainCampaign(created), nil
}

func (s *Store) UpdateCampaignDecision(ctx context.Context, id string, approval domain.Approval) (domain.Campaign, error) {
	c, err := s.Queries.UpdateCampaignDecision(ctx, gen.UpdateCampaignDecisionParams{
		ID: id, Approval: gen.ApprovalStatusT(approval),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Campaign{}, ErrNotFound
		}
		return domain.Campaign{}, err
	}
	return toDomainCampaign(c), nil
}

func (s *Store) BulkUpdateApproval(ctx context.Context, ids []string, approval domain.Approval) ([]domain.Campaign, error) {
	cs, err := s.Queries.BulkUpdateApproval(ctx, gen.BulkUpdateApprovalParams{
		Ids: ids, Approval: gen.ApprovalStatusT(approval),
	})
	if err != nil {
		return nil, err
	}
	return toDomainCampaigns(cs), nil
}

func (s *Store) UpdateCampaignStatus(ctx context.Context, id string, status domain.Status) (domain.Campaign, error) {
	c, err := s.Queries.UpdateCampaignStatus(ctx, gen.UpdateCampaignStatusParams{
		ID: id, Status: gen.CampaignStatusT(status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Campaign{}, ErrNotFound
		}
		return domain.Campaign{}, err
	}
	return toDomainCampaign(c), nil
}

// UpdateCampaignFlag sets flag_reason; pass "" to clear an existing flag.
func (s *Store) UpdateCampaignFlag(ctx context.Context, id string, reason string) (domain.Campaign, error) {
	c, err := s.Queries.UpdateCampaignFlag(ctx, gen.UpdateCampaignFlagParams{
		ID: id, FlagReason: textParam(reason),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Campaign{}, ErrNotFound
		}
		return domain.Campaign{}, err
	}
	return toDomainCampaign(c), nil
}

func (s *Store) CountPendingApprovals(ctx context.Context) (int64, error) {
	return s.Queries.CountPendingApprovals(ctx)
}

func (s *Store) TotalPendingSpend(ctx context.Context) (int64, error) {
	return s.Queries.TotalPendingSpend(ctx)
}
