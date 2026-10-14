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
	sys, org := orgParams(ctx)
	cs, err := s.db.Queries.ListCampaigns(ctx, gen.ListCampaignsParams{OrgSystem: sys, OrgID: org})
	if err != nil {
		return nil, err
	}
	return scoped(ctx, toDomainCampaigns(cs)), nil
}

func (s *Store) ListCampaignsFiltered(ctx context.Context, f CampaignFilter) ([]campaigns.Campaign, error) {
	sys, org := orgParams(ctx)
	arg := gen.ListCampaignsFilteredParams{
		OrgSystem: sys,
		OrgID:     org,
		Category:  nullText(f.Category),
		Region:    nullText(f.Region),
		Search:    nullText(f.Search),
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
	sys, org := orgParams(ctx)
	// Out of scope reads as absent, not as forbidden: telling a client
	// that a campaign exists but is not theirs is itself a disclosure.
	if !scope.CampaignsFrom(ctx).Allows(id) {
		return campaigns.Campaign{}, database.ErrNotFound
	}
	c, err := s.db.Queries.GetCampaign(ctx, gen.GetCampaignParams{ID: id, OrgSystem: sys, OrgID: org})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return campaigns.Campaign{}, database.ErrNotFound
		}
		return campaigns.Campaign{}, err
	}
	return toDomainCampaign(c), nil
}

func (s *Store) ListRunningCampaigns(ctx context.Context) ([]campaigns.Campaign, error) {
	sys, org := orgParams(ctx)
	cs, err := s.db.Queries.ListRunningCampaigns(ctx, gen.ListRunningCampaignsParams{OrgSystem: sys, OrgID: org})
	if err != nil {
		return nil, err
	}
	return scoped(ctx, toDomainCampaigns(cs)), nil
}

func (s *Store) ListPendingCampaigns(ctx context.Context) ([]campaigns.Campaign, error) {
	sys, org := orgParams(ctx)
	cs, err := s.db.Queries.ListPendingCampaigns(ctx, gen.ListPendingCampaignsParams{OrgSystem: sys, OrgID: org})
	if err != nil {
		return nil, err
	}
	return scoped(ctx, toDomainCampaigns(cs)), nil
}

func (s *Store) ListFlaggedCampaigns(ctx context.Context) ([]campaigns.Campaign, error) {
	sys, org := orgParams(ctx)
	cs, err := s.db.Queries.ListFlaggedCampaigns(ctx, gen.ListFlaggedCampaignsParams{OrgSystem: sys, OrgID: org})
	if err != nil {
		return nil, err
	}
	return scoped(ctx, toDomainCampaigns(cs)), nil
}

func (s *Store) CreateCampaign(ctx context.Context, c campaigns.Campaign) (campaigns.Campaign, error) {
	_, org := orgParams(ctx)
	created, err := s.db.Queries.CreateCampaign(ctx, gen.CreateCampaignParams{
		OrgID:       org,
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
	sys, org := orgParams(ctx)
	c, err := s.db.Queries.UpdateCampaignDecision(ctx, gen.UpdateCampaignDecisionParams{
		OrgSystem: sys, OrgID: org,
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
	sys, org := orgParams(ctx)
	cs, err := s.db.Queries.BulkUpdateApproval(ctx, gen.BulkUpdateApprovalParams{
		OrgSystem: sys, OrgID: org,
		Ids: ids, Approval: gen.ApprovalStatusT(approval),
	})
	if err != nil {
		return nil, err
	}
	return scoped(ctx, toDomainCampaigns(cs)), nil
}

func (s *Store) UpdateCampaignStatus(ctx context.Context, id string, status campaigns.Status) (campaigns.Campaign, error) {
	sys, org := orgParams(ctx)
	c, err := s.db.Queries.UpdateCampaignStatus(ctx, gen.UpdateCampaignStatusParams{
		OrgSystem: sys, OrgID: org,
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
	sys, org := orgParams(ctx)
	c, err := s.db.Queries.UpdateCampaignFlag(ctx, gen.UpdateCampaignFlagParams{
		OrgSystem: sys, OrgID: org,
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
	sys, org := orgParams(ctx)
	restricted, ids := scopeParams(ctx)
	return s.db.Queries.CountPendingApprovals(ctx, gen.CountPendingApprovalsParams{
		OrgSystem: sys, OrgID: org,
		Restricted: restricted, VisibleIds: ids,
	})
}

func (s *Store) TotalPendingSpend(ctx context.Context) (int64, error) {
	sys, org := orgParams(ctx)
	restricted, ids := scopeParams(ctx)
	return s.db.Queries.TotalPendingSpend(ctx, gen.TotalPendingSpendParams{
		OrgSystem: sys, OrgID: org,
		Restricted: restricted, VisibleIds: ids,
	})
}

// OverviewTotals is the whole of the overview's headline arithmetic, done
// in one query rather than by loading every campaign and summing it here.
func (s *Store) OverviewTotals(ctx context.Context) (campaigns.Totals, error) {
	sys, org := orgParams(ctx)
	restricted, ids := scopeParams(ctx)
	row, err := s.db.Queries.OverviewTotals(ctx, gen.OverviewTotalsParams{
		OrgSystem: sys, OrgID: org,
		Restricted: restricted, VisibleIds: ids,
	})
	if err != nil {
		return campaigns.Totals{}, err
	}
	return campaigns.Totals{
		LiveCount:      row.LiveCount,
		ReachLive:      row.ReachLive,
		SpendWindow:    row.SpendWindow,
		ApprovedBudget: row.ApprovedBudget,
	}, nil
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
	sys, org := orgParams(ctx)
	cs, err := s.db.Queries.ListCampaignsByCreator(ctx, gen.ListCampaignsByCreatorParams{CreatorID: database.TextParam(creatorID), OrgSystem: sys, OrgID: org})
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
// scopeParams is the aggregate counterpart of scoped(). A row query can
// load and then filter; an aggregate cannot, because by the time it
// returns there are no rows left to drop. Anything that returns a number
// rather than rows has to go through here.
//
// The empty-but-restricted case is the one that matters: a client whose
// grants have not been set yet gets an empty id list, and `id = ANY('{}')`
// matches nothing — which is the safe answer. Treating it as unrestricted
// instead, the easy mistake because an empty filter usually means no
// filter, would hand them the whole agency's totals.
func scopeParams(ctx context.Context) (bool, []string) {
	sc := scope.CampaignsFrom(ctx)
	if !sc.Restricted {
		return false, nil
	}
	ids := sc.IDs
	if ids == nil {
		ids = []string{}
	}
	return true, ids
}

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

// orgParams turns the request's tenant into query parameters. Every
// statement touching an org-owned table takes these; platform/arch's
// tenancy test fails the build if one is added that does not.
//
// A system context spans organisations, which is right for the worker,
// the seeder and the CLI. Anything else is confined to one, and an
// unresolved organisation is the empty string — which matches no row, so
// forgetting to set it yields an empty result rather than a leak.
func orgParams(ctx context.Context) (bool, string) {
	o := scope.OrgFrom(ctx)
	return o.System, o.ID
}
