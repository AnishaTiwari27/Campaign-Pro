package store

import (
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/campaigns"
	"context"
	"errors"

	"campaigntrackerpro/db/gen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

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

func (s *Store) ListCampaigns(ctx context.Context) ([]campaigns.Campaign, error) {
	cs, err := s.db.Queries.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	return scoped(ctx, toDomainCampaigns(cs)), nil
}

func (s *Store) ListCampaignsFiltered(ctx context.Context, f CampaignFilter) ([]campaigns.Campaign, error) {
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
	cs, err := s.db.Queries.ListCampaignsFiltered(ctx, arg)
	if err != nil {
		return nil, err
	}
	return scoped(ctx, toDomainCampaigns(cs)), nil
}

func (s *Store) GetCampaign(ctx context.Context, id string) (campaigns.Campaign, error) {
	// Out of scope reads as absent, not as forbidden: telling a client
	// that a campaign exists but is not theirs is itself a disclosure.
	if !scope.CampaignsFrom(ctx).Allows(id) {
		return campaigns.Campaign{}, database.ErrNotFound
	}
	c, err := s.db.Queries.GetCampaign(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return campaigns.Campaign{}, database.ErrNotFound
		}
		return campaigns.Campaign{}, err
	}
	return toDomainCampaign(c), nil
}

func (s *Store) ListRunningCampaigns(ctx context.Context) ([]campaigns.Campaign, error) {
	cs, err := s.db.Queries.ListRunningCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	return scoped(ctx, toDomainCampaigns(cs)), nil
}

func (s *Store) ListPendingCampaigns(ctx context.Context) ([]campaigns.Campaign, error) {
	cs, err := s.db.Queries.ListPendingCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	return scoped(ctx, toDomainCampaigns(cs)), nil
}

func (s *Store) ListFlaggedCampaigns(ctx context.Context) ([]campaigns.Campaign, error) {
	cs, err := s.db.Queries.ListFlaggedCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	return scoped(ctx, toDomainCampaigns(cs)), nil
}

func (s *Store) CreateCampaign(ctx context.Context, c campaigns.Campaign) (campaigns.Campaign, error) {
	created, err := s.db.Queries.CreateCampaign(ctx, gen.CreateCampaignParams{
		ID:          c.ID,
		Name:        c.Name,
		SubjectType: gen.SubjectTypeT(c.SubjectType),
		Role:        database.TextParam(c.Role),
		Initials:    c.Initials,
		Category:    c.Category,
		Region:      c.Region,
		AdType:      gen.AdTypeT(c.AdType),
		Platform:    c.Platform,
		Status:      gen.CampaignStatusT(c.Status),
		DaysRunning: int32(c.DaysRunning),
		FlightDays:  database.Int32Param(c.FlightDays),
		Reach:       c.Reach,
		Spend:       c.Spend,
		Budget:      c.Budget,
		Frequency:   c.Frequency,
		Approval:    gen.ApprovalStatusT(c.Approval),
		CurveShape:  gen.CurveShapeT(c.CurveShape),
		FlagReason:  database.TextParam(c.FlagReason),
		BrandDomain: database.TextParam(c.BrandDomain),
	})
	if err != nil {
		return campaigns.Campaign{}, err
	}
	return toDomainCampaign(created), nil
}

func (s *Store) UpdateCampaignDecision(ctx context.Context, id string, approval campaigns.Approval) (campaigns.Campaign, error) {
	c, err := s.db.Queries.UpdateCampaignDecision(ctx, gen.UpdateCampaignDecisionParams{
		ID: id, Approval: gen.ApprovalStatusT(approval),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return campaigns.Campaign{}, database.ErrNotFound
		}
		return campaigns.Campaign{}, err
	}
	return toDomainCampaign(c), nil
}

func (s *Store) BulkUpdateApproval(ctx context.Context, ids []string, approval campaigns.Approval) ([]campaigns.Campaign, error) {
	cs, err := s.db.Queries.BulkUpdateApproval(ctx, gen.BulkUpdateApprovalParams{
		Ids: ids, Approval: gen.ApprovalStatusT(approval),
	})
	if err != nil {
		return nil, err
	}
	return scoped(ctx, toDomainCampaigns(cs)), nil
}

func (s *Store) UpdateCampaignStatus(ctx context.Context, id string, status campaigns.Status) (campaigns.Campaign, error) {
	c, err := s.db.Queries.UpdateCampaignStatus(ctx, gen.UpdateCampaignStatusParams{
		ID: id, Status: gen.CampaignStatusT(status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return campaigns.Campaign{}, database.ErrNotFound
		}
		return campaigns.Campaign{}, err
	}
	return toDomainCampaign(c), nil
}

// UpdateCampaignFlag sets flag_reason; pass "" to clear an existing flag.
func (s *Store) UpdateCampaignFlag(ctx context.Context, id string, reason string) (campaigns.Campaign, error) {
	c, err := s.db.Queries.UpdateCampaignFlag(ctx, gen.UpdateCampaignFlagParams{
		ID: id, FlagReason: database.TextParam(reason),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return campaigns.Campaign{}, database.ErrNotFound
		}
		return campaigns.Campaign{}, err
	}
	return toDomainCampaign(c), nil
}

func (s *Store) CountPendingApprovals(ctx context.Context) (int64, error) {
	return s.db.Queries.CountPendingApprovals(ctx)
}

func (s *Store) TotalPendingSpend(ctx context.Context) (int64, error) {
	return s.db.Queries.TotalPendingSpend(ctx)
}

// SetCampaignCreator links a campaign to the creator who ran it. The
// campaigns table is this service's, so the write lives here even though
// the creators service is what triggers it.
func (s *Store) SetCampaignCreator(ctx context.Context, campaignID, creatorID string) error {
	return s.db.Queries.SetCampaignCreator(ctx, gen.SetCampaignCreatorParams{
		ID: campaignID, CreatorID: database.TextParam(creatorID),
	})
}

// ListCampaignsByCreator returns one creator's campaigns.
func (s *Store) ListCampaignsByCreator(ctx context.Context, creatorID string) ([]campaigns.Campaign, error) {
	cs, err := s.db.Queries.ListCampaignsByCreator(ctx, database.TextParam(creatorID))
	if err != nil {
		return nil, err
	}
	return scoped(ctx, toDomainCampaigns(cs)), nil
}

// scoped drops anything the caller may not see. Applied at every list
// return in this file rather than in the SQL, because the filter must hold
// for all six of them and a WHERE clause added to five is a hole.
//
// With an unrestricted scope this is a no-op, which is every caller that
// is not a client request.
func scoped(ctx context.Context, cs []campaigns.Campaign) []campaigns.Campaign {
	sc := scope.CampaignsFrom(ctx)
	if !sc.Restricted {
		return cs
	}
	out := make([]campaigns.Campaign, 0, len(cs))
	for _, c := range cs {
		if sc.Allows(c.ID) {
			out = append(out, c)
		}
	}
	return out
}
