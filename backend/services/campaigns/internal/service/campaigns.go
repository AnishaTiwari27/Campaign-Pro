// Package service holds the app's use cases: it composes store (data
// access) and domain (pure formulas) into the operations the HTTP layer
// calls directly.
package service

import (
	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/campaigns"
	"context"
	"sort"
	"strconv"
	"strings"

	"campaigntrackerpro/services/campaigns/internal/store"
)

var ErrNotFound = httpx.ErrNotFound
var ErrForbidden = httpx.ErrForbidden
var ErrValidation = httpx.ErrValidation

type Campaigns struct {
	Store *store.Store
}

func NewCampaigns(s *store.Store) *Campaigns { return &Campaigns{Store: s} }

func (c *Campaigns) enrich(ctx context.Context, all []campaigns.Campaign) (map[string]campaigns.CampaignRow, float64, []campaigns.CategoryBenchmark, error) {
	running := campaigns.Running(all)
	medAll := campaigns.MedianReach(running)
	benchmarks := campaigns.CategoryBenchmarks(running)

	rows := make(map[string]campaigns.CampaignRow, len(all))
	for _, camp := range all {
		pace := campaigns.Pace(camp.Spend, camp.Budget)
		rows[camp.ID] = campaigns.CampaignRow{
			Campaign:      camp,
			Pace:          pace,
			PaceClass:     campaigns.PaceClassOf(pace),
			CPM:           campaigns.CPM(camp.Spend, camp.Reach, camp.Frequency),
			Index:         campaigns.Index(camp.Reach, medAll),
			CategoryIndex: campaigns.CategoryIndexOf(camp, benchmarks),
		}
	}
	return rows, medAll, benchmarks, nil
}

// List applies every filter, then sort/page in Go (sqlc can't parameterize
// ORDER BY), returning derived fields alongside each row.
func (c *Campaigns) List(ctx context.Context, p campaigns.ListParams) (campaigns.ListResult, error) {
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
		return campaigns.ListResult{}, err
	}

	all, err := c.Store.ListCampaigns(ctx)
	if err != nil {
		return campaigns.ListResult{}, err
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

	rows := make([]campaigns.CampaignRow, 0, len(matched))
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

	return campaigns.ListResult{Items: rows[start:end], Total: total, Page: page, Pages: pages}, nil
}

func sortRows(rows []campaigns.CampaignRow, sortKey, dir string) {
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
	campaigns.CampaignRow
	Creatives []campaigns.Creative
	Audit     []campaigns.AuditEvent
	Similar   []campaigns.CampaignRow
	Position  int // 1-based position within the current filtered+sorted list
	Total     int
	PrevID    string
	NextID    string
	MedAll    float64
}

// GetDetail returns a campaign plus everything its detail page needs. The
// position/total pair is computed against the same filter+sort the caller
// was browsing, so prev/next on the detail page follows that exact order.
func (c *Campaigns) GetDetail(ctx context.Context, id string, listParams campaigns.ListParams) (CampaignDetail, error) {
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
		row  campaigns.CampaignRow
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
	similar := make([]campaigns.CampaignRow, 0, 5)
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

// Actor is who is performing a write, in the two forms the audit trail
// needs: ID so "who approved this?" is answerable by join, and Name as the
// display label the Activity tab renders. They travel together because an
// event carrying only one of them is a trail with a hole in it.
type Actor struct {
	ID   string
	Name string
}

// ApprovalBlockedError is returned when the caller may approve in principle
// but this campaign's own state forbids it. It is distinct from ErrForbidden
// (which is about the caller) and from ErrValidation (which is about the
// request): the request is well-formed and permitted, and the campaign has
// to change before it can be approved. Reason is written for the approver.
type ApprovalBlockedError struct {
	// Blocked maps campaign ID to the reason it cannot be approved. A
	// single decision populates one entry; a bulk decision may name several.
	Blocked map[string]string
}

func (e ApprovalBlockedError) Error() string {
	if len(e.Blocked) == 1 {
		for id, reason := range e.Blocked {
			return "approval blocked for " + id + ": " + reason
		}
	}
	return "approval blocked for " + strconv.Itoa(len(e.Blocked)) + " campaigns"
}

// Unwrap makes errors.Is(err, httpx.ErrConflict) true, so a handler that
// forgets to pull out the specific reason still answers 409 rather than 500.
func (e ApprovalBlockedError) Unwrap() error { return httpx.ErrConflict }

// Reason is the message to show when exactly one campaign was blocked, which
// is every single-campaign decision.
func (e ApprovalBlockedError) Reason() string {
	for _, reason := range e.Blocked {
		return reason
	}
	return "this campaign cannot be approved in its current state"
}

// blockedIfApproving returns an ApprovalBlockedError naming every campaign in
// ids that campaigns.ApprovalBlock refuses. It reads state before writing any
// of it, so a bulk approve containing one over-budget campaign changes
// nothing rather than approving the rest and leaving the caller to guess
// which were skipped.
func (c *Campaigns) blockedIfApproving(ctx context.Context, ids []string) error {
	blocked := map[string]string{}
	for _, id := range ids {
		camp, err := c.Store.GetCampaign(ctx, id)
		if err != nil {
			return err
		}
		if reason := campaigns.ApprovalBlock(camp); reason != "" {
			blocked[id] = reason
		}
	}
	if len(blocked) > 0 {
		return ApprovalBlockedError{Blocked: blocked}
	}
	return nil
}

// Decision applies approve/reject/reopen, enforcing can_approve server-side,
// refusing an approval the campaign's budget state forbids, and writing an
// audit event attributed to the acting user.
//
// Only approve is guarded. Rejecting or reopening an over-budget campaign is
// always allowed — those are how an approver responds to one.
func (c *Campaigns) Decision(ctx context.Context, id, action string, actor Actor, canApprove bool) (campaigns.Campaign, error) {
	if !canApprove {
		return campaigns.Campaign{}, ErrForbidden
	}
	var approval campaigns.Approval
	var auditAction string
	switch action {
	case "approve":
		approval, auditAction = campaigns.ApprovalApproved, "Approved"
	case "reject":
		approval, auditAction = campaigns.ApprovalRejected, "Rejected"
	case "reopen":
		approval, auditAction = campaigns.ApprovalPending, "Reopened for review"
	default:
		return campaigns.Campaign{}, ErrValidation
	}
	if action == "approve" {
		if err := c.blockedIfApproving(ctx, []string{id}); err != nil {
			return campaigns.Campaign{}, err
		}
	}
	camp, err := c.Store.UpdateCampaignDecision(ctx, id, approval)
	if err != nil {
		return campaigns.Campaign{}, err
	}
	if _, err := c.Store.CreateAuditEvent(ctx, id, actor.ID, actor.Name, auditAction, "user"); err != nil {
		return campaigns.Campaign{}, err
	}
	return camp, nil
}

func (c *Campaigns) BulkDecision(ctx context.Context, ids []string, action string, actor Actor, canApprove bool) ([]campaigns.Campaign, error) {
	if !canApprove {
		return nil, ErrForbidden
	}
	var approval campaigns.Approval
	var auditAction string
	switch action {
	case "approve":
		approval, auditAction = campaigns.ApprovalApproved, "Approved (bulk)"
	case "reject":
		approval, auditAction = campaigns.ApprovalRejected, "Rejected (bulk)"
	default:
		return nil, ErrValidation
	}
	if action == "approve" {
		if err := c.blockedIfApproving(ctx, ids); err != nil {
			return nil, err
		}
	}
	camps, err := c.Store.BulkUpdateApproval(ctx, ids, approval)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if _, err := c.Store.CreateAuditEvent(ctx, id, actor.ID, actor.Name, auditAction, "user"); err != nil {
			return nil, err
		}
	}
	return camps, nil
}

// Pause toggles live <-> paused; any other current status is a no-op error.
func (c *Campaigns) Pause(ctx context.Context, id string, actor Actor) (campaigns.Campaign, error) {
	camp, err := c.Store.GetCampaign(ctx, id)
	if err != nil {
		return campaigns.Campaign{}, err
	}
	var next campaigns.Status
	var auditAction string
	switch camp.Status {
	case campaigns.StatusLive:
		next, auditAction = campaigns.StatusPaused, "Paused"
	case campaigns.StatusPaused:
		next, auditAction = campaigns.StatusLive, "Resumed"
	default:
		return campaigns.Campaign{}, ErrValidation
	}
	updated, err := c.Store.UpdateCampaignStatus(ctx, id, next)
	if err != nil {
		return campaigns.Campaign{}, err
	}
	if _, err := c.Store.CreateAuditEvent(ctx, id, actor.ID, actor.Name, auditAction, "user"); err != nil {
		return campaigns.Campaign{}, err
	}
	return updated, nil
}

func (c *Campaigns) AddNote(ctx context.Context, id, text string, actor Actor) (campaigns.AuditEvent, error) {
	if strings.TrimSpace(text) == "" {
		return campaigns.AuditEvent{}, ErrValidation
	}
	if _, err := c.Store.GetCampaign(ctx, id); err != nil {
		return campaigns.AuditEvent{}, err
	}
	return c.Store.CreateAuditEvent(ctx, id, actor.ID, actor.Name, text, "user")
}

func (c *Campaigns) Anomalies(ctx context.Context) ([]campaigns.Campaign, error) {
	return c.Store.ListFlaggedCampaigns(ctx)
}

// RowOf recomputes derived fields for a single campaign against the full
// current campaign set — handlers use this to return an enriched row right
// after a write (decision/pause/note) without duplicating enrich's logic.
func (c *Campaigns) RowOf(ctx context.Context, camp campaigns.Campaign) campaigns.CampaignRow {
	all, err := c.Store.ListCampaigns(ctx)
	if err != nil {
		return campaigns.CampaignRow{Campaign: camp}
	}
	rowsByID, _, _, _ := c.enrich(ctx, all)
	if row, ok := rowsByID[camp.ID]; ok {
		return row
	}
	return campaigns.CampaignRow{Campaign: camp}
}

// --- capabilities published to sibling services -------------------------
//
// Analytics, creators and reports each declare the narrow interface they
// need; these are the methods that satisfy them. Keeping them together
// makes it obvious what this service exposes beyond its own transport.

// All returns every campaign, the starting point for anything that has to
// aggregate across the fleet.
func (c *Campaigns) All(ctx context.Context) ([]campaigns.Campaign, error) {
	return c.Store.ListCampaigns(ctx)
}

// Enrich derives pace, CPM and the two indices for a set of campaigns.
// Siblings need the derived view, not the raw rows.
func (c *Campaigns) Enrich(ctx context.Context, all []campaigns.Campaign) map[string]campaigns.CampaignRow {
	rows, _, _, _ := c.enrich(ctx, all)
	return rows
}

func (c *Campaigns) PendingCount(ctx context.Context) (int64, error) {
	return c.Store.CountPendingApprovals(ctx)
}

func (c *Campaigns) PendingSpend(ctx context.Context) (int64, error) {
	return c.Store.TotalPendingSpend(ctx)
}

func (c *Campaigns) Flagged(ctx context.Context) ([]campaigns.Campaign, error) {
	return c.Store.ListFlaggedCampaigns(ctx)
}

func (c *Campaigns) CreativesFor(ctx context.Context, campaignID string) ([]campaigns.Creative, error) {
	return c.Store.ListCreativesByCampaign(ctx, campaignID)
}
