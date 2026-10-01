// Package service holds the app's use cases: it composes store (data
// access) and domain (pure formulas) into the operations the HTTP layer
// calls directly.
package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"campaigntrackerpro/internal/domain"
	"campaigntrackerpro/internal/store"
)

var ErrNotFound = store.ErrNotFound
var ErrForbidden = fmt.Errorf("forbidden")
var ErrValidation = fmt.Errorf("validation")

type Campaigns struct {
	Store *store.Store
}

func NewCampaigns(s *store.Store) *Campaigns { return &Campaigns{Store: s} }

// CampaignRow is a campaign plus every value derived from it that a list or
// detail view needs to render without recomputing formulas itself.
type CampaignRow struct {
	domain.Campaign
	Pace          float64
	PaceClass     domain.PaceClass
	CPM           float64
	Index         float64
	CategoryIndex float64
}

// ListParams mirrors every GET /campaigns query parameter.
type ListParams struct {
	Search      string
	SubjectType string // "" | brand | person
	Category    string
	Region      string
	AdType      string
	Status      string
	Approval    string
	Range       string // "" | "7" | "30" (All time = no filter)
	Sort        string
	Dir         string // asc | desc
	Page        int
	Per         int
}

type ListResult struct {
	Items []CampaignRow
	Total int
	Page  int
	Pages int
}

func (c *Campaigns) enrich(ctx context.Context, all []domain.Campaign) (map[string]CampaignRow, float64, []domain.CategoryBenchmark, error) {
	running := domain.Running(all)
	medAll := domain.MedianReach(running)
	benchmarks := domain.CategoryBenchmarks(running)

	rows := make(map[string]CampaignRow, len(all))
	for _, camp := range all {
		pace := domain.Pace(camp.Spend, camp.Budget)
		rows[camp.ID] = CampaignRow{
			Campaign:      camp,
			Pace:          pace,
			PaceClass:     domain.PaceClassOf(pace),
			CPM:           domain.CPM(camp.Spend, camp.Reach, camp.Frequency),
			Index:         domain.Index(camp.Reach, medAll),
			CategoryIndex: domain.CategoryIndexOf(camp, benchmarks),
		}
	}
	return rows, medAll, benchmarks, nil
}

// List applies every filter, then sort/page in Go (sqlc can't parameterize
// ORDER BY), returning derived fields alongside each row.
func (c *Campaigns) List(ctx context.Context, p ListParams) (ListResult, error) {
	filter := store.CampaignFilter{}
	if p.SubjectType != "" {
		filter.SubjectType = &p.SubjectType
	}
	if p.Category != "" {
		filter.Category = &p.Category
	}
	if p.Region != "" {
		filter.Region = &p.Region
	}
	if p.AdType != "" {
		filter.AdType = &p.AdType
	}
	if p.Status != "" {
		filter.Status = &p.Status
	}
	if p.Approval != "" {
		filter.Approval = &p.Approval
	}
	if p.Search != "" {
		filter.Search = &p.Search
	}

	matched, err := c.Store.ListCampaignsFiltered(ctx, filter)
	if err != nil {
		return ListResult{}, err
	}

	all, err := c.Store.ListCampaigns(ctx)
	if err != nil {
		return ListResult{}, err
	}
	rowsByID, _, _, _ := c.enrich(ctx, all)

	if p.Range == "7" || p.Range == "30" {
		days := 7
		if p.Range == "30" {
			days = 30
		}
		filtered := matched[:0]
		for _, camp := range matched {
			if camp.DaysRunning <= days {
				filtered = append(filtered, camp)
			}
		}
		matched = filtered
	}

	rows := make([]CampaignRow, 0, len(matched))
	for _, camp := range matched {
		rows = append(rows, rowsByID[camp.ID])
	}

	sortRows(rows, p.Sort, p.Dir)

	total := len(rows)
	per := p.Per
	if per <= 0 {
		per = 10
	}
	pages := (total + per - 1) / per
	if pages < 1 {
		pages = 1
	}
	page := p.Page
	if page < 1 {
		page = 1
	}
	if page > pages {
		page = pages
	}
	start := (page - 1) * per
	end := start + per
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	return ListResult{Items: rows[start:end], Total: total, Page: page, Pages: pages}, nil
}

func sortRows(rows []CampaignRow, sortKey, dir string) {
	asc := dir != "desc"
	less := func(i, j int) bool {
		a, b := rows[i], rows[j]
		var cmp bool
		switch sortKey {
		case "name":
			cmp = a.Name < b.Name
		case "category":
			cmp = a.Category < b.Category
		case "region":
			cmp = a.Region < b.Region
		case "status":
			cmp = a.Status < b.Status
		case "approval":
			cmp = a.Approval < b.Approval
		case "budget":
			cmp = a.Budget < b.Budget
		case "pace":
			cmp = a.Pace < b.Pace
		case "cpm":
			cmp = a.CPM < b.CPM
		case "index":
			cmp = a.Index < b.Index
		case "createdAt":
			cmp = a.CreatedAt.Before(b.CreatedAt)
		case "spend":
			cmp = a.Spend < b.Spend
		case "reach":
			fallthrough
		default:
			cmp = a.Reach < b.Reach
		}
		if asc {
			return cmp
		}
		return !cmp && a.ID != b.ID
	}
	sort.SliceStable(rows, less)
}

type CampaignDetail struct {
	CampaignRow
	Creatives []domain.Creative
	Audit     []domain.AuditEvent
	Similar   []CampaignRow
	Position  int // 1-based position within the current filtered+sorted list
	Total     int
	PrevID    string
	NextID    string
	MedAll    float64
}

// GetDetail returns a campaign plus everything its detail page needs. The
// position/total pair is computed against the same filter+sort the caller
// was browsing, so prev/next on the detail page follows that exact order.
func (c *Campaigns) GetDetail(ctx context.Context, id string, listParams ListParams) (CampaignDetail, error) {
	camp, err := c.Store.GetCampaign(ctx, id)
	if err != nil {
		return CampaignDetail{}, err
	}

	all, err := c.Store.ListCampaigns(ctx)
	if err != nil {
		return CampaignDetail{}, err
	}
	rowsByID, medAll, _, _ := c.enrich(ctx, all)
	row := rowsByID[id]

	creatives, err := c.Store.ListCreativesByCampaign(ctx, id)
	if err != nil {
		return CampaignDetail{}, err
	}
	audit, err := c.Store.ListAuditEventsByCampaign(ctx, id)
	if err != nil {
		return CampaignDetail{}, err
	}

	// Similar: up to 5 campaigns sharing category or ad type, nearest reach first.
	type scored struct {
		row  CampaignRow
		dist float64
	}
	var candidates []scored
	for _, other := range all {
		if other.ID == id {
			continue
		}
		if other.Category == camp.Category || other.AdType == camp.AdType {
			candidates = append(candidates, scored{rowsByID[other.ID], absF(other.Reach - camp.Reach)})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].dist < candidates[j].dist })
	similar := make([]CampaignRow, 0, 5)
	for i := 0; i < len(candidates) && i < 5; i++ {
		similar = append(similar, candidates[i].row)
	}

	// Position within the list the caller was browsing: re-run the same
	// filter+sort unpaged so prev/next matches whatever order they were
	// looking at.
	position, total := 0, 0
	var prevID, nextID string
	unpagedParams := listParams
	unpagedParams.Page = 1
	unpagedParams.Per = len(all) + 1
	if full, ferr := c.List(ctx, unpagedParams); ferr == nil {
		total = full.Total
		for i, r := range full.Items {
			if r.ID == id {
				position = i + 1
				if i > 0 {
					prevID = full.Items[i-1].ID
				}
				if i < len(full.Items)-1 {
					nextID = full.Items[i+1].ID
				}
				break
			}
		}
	}

	return CampaignDetail{
		CampaignRow: row,
		Creatives:   creatives,
		Audit:       audit,
		Similar:     similar,
		Position:    position,
		Total:       total,
		PrevID:      prevID,
		NextID:      nextID,
		MedAll:      medAll,
	}, nil
}

func absF(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// Decision applies approve/reject/reopen, enforcing can_approve server-side
// and writing an audit event.
func (c *Campaigns) Decision(ctx context.Context, id, action, actor string, canApprove bool) (domain.Campaign, error) {
	if !canApprove {
		return domain.Campaign{}, ErrForbidden
	}
	var approval domain.Approval
	var auditAction string
	switch action {
	case "approve":
		approval, auditAction = domain.ApprovalApproved, "Approved"
	case "reject":
		approval, auditAction = domain.ApprovalRejected, "Rejected"
	case "reopen":
		approval, auditAction = domain.ApprovalPending, "Reopened for review"
	default:
		return domain.Campaign{}, ErrValidation
	}
	camp, err := c.Store.UpdateCampaignDecision(ctx, id, approval)
	if err != nil {
		return domain.Campaign{}, err
	}
	if _, err := c.Store.CreateAuditEvent(ctx, id, actor, auditAction, "user"); err != nil {
		return domain.Campaign{}, err
	}
	return camp, nil
}

func (c *Campaigns) BulkDecision(ctx context.Context, ids []string, action, actor string, canApprove bool) ([]domain.Campaign, error) {
	if !canApprove {
		return nil, ErrForbidden
	}
	var approval domain.Approval
	var auditAction string
	switch action {
	case "approve":
		approval, auditAction = domain.ApprovalApproved, "Approved (bulk)"
	case "reject":
		approval, auditAction = domain.ApprovalRejected, "Rejected (bulk)"
	default:
		return nil, ErrValidation
	}
	camps, err := c.Store.BulkUpdateApproval(ctx, ids, approval)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if _, err := c.Store.CreateAuditEvent(ctx, id, actor, auditAction, "user"); err != nil {
			return nil, err
		}
	}
	return camps, nil
}

// Pause toggles live <-> paused; any other current status is a no-op error.
func (c *Campaigns) Pause(ctx context.Context, id, actor string) (domain.Campaign, error) {
	camp, err := c.Store.GetCampaign(ctx, id)
	if err != nil {
		return domain.Campaign{}, err
	}
	var next domain.Status
	var auditAction string
	switch camp.Status {
	case domain.StatusLive:
		next, auditAction = domain.StatusPaused, "Paused"
	case domain.StatusPaused:
		next, auditAction = domain.StatusLive, "Resumed"
	default:
		return domain.Campaign{}, ErrValidation
	}
	updated, err := c.Store.UpdateCampaignStatus(ctx, id, next)
	if err != nil {
		return domain.Campaign{}, err
	}
	if _, err := c.Store.CreateAuditEvent(ctx, id, actor, auditAction, "user"); err != nil {
		return domain.Campaign{}, err
	}
	return updated, nil
}

func (c *Campaigns) AddNote(ctx context.Context, id, text, actor string) (domain.AuditEvent, error) {
	if strings.TrimSpace(text) == "" {
		return domain.AuditEvent{}, ErrValidation
	}
	if _, err := c.Store.GetCampaign(ctx, id); err != nil {
		return domain.AuditEvent{}, err
	}
	return c.Store.CreateAuditEvent(ctx, id, actor, text, "user")
}

func (c *Campaigns) Anomalies(ctx context.Context) ([]domain.Campaign, error) {
	return c.Store.ListFlaggedCampaigns(ctx)
}

// RowOf recomputes derived fields for a single campaign against the full
// current campaign set — handlers use this to return an enriched row right
// after a write (decision/pause/note) without duplicating enrich's logic.
func (c *Campaigns) RowOf(ctx context.Context, camp domain.Campaign) CampaignRow {
	all, err := c.Store.ListCampaigns(ctx)
	if err != nil {
		return CampaignRow{Campaign: camp}
	}
	rowsByID, _, _, _ := c.enrich(ctx, all)
	if row, ok := rowsByID[camp.ID]; ok {
		return row
	}
	return CampaignRow{Campaign: camp}
}
