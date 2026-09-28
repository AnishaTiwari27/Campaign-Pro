package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/campaigns/internal/models"
	"campaigntrackerpro/services/campaigns/internal/store"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.Ping(r.Context()); err != nil {
		platform.Upstream(w, "database unreachable")
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "instance": s.InstanceID})
}

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	today := time.Now()
	f := parseFilter(r, today)
	page, limit := parsePagination(r)

	rows, total, err := s.Store.List(r.Context(), f, page, limit, today)
	if err != nil {
		slog.Error("list campaigns", "err", err)
		platform.Internal(w, "failed to list campaigns")
		return
	}

	out := make([]any, len(rows))
	for i, c := range rows {
		out[i] = c.AsJSON()
	}
	platform.WriteJSON(w, http.StatusOK, map[string]any{"data": out, "page": page, "limit": limit, "total": total})
}

// handleAggregateKPIs, handleAggregateTrend, handleAggregateRegions, and
// handleAggregateBenchmark are analytics-service's only way of reading
// campaign facts: pre-aggregated in SQL, not the full filtered row set —
// see store/aggregates.go and docs/ROADMAP.md's Phase A for why. Not
// routed through the gateway's public path, same as every other
// /internal/* route.
func (s *Server) handleAggregateKPIs(w http.ResponseWriter, r *http.Request) {
	today := time.Now()
	f := parseFilter(r, today)

	totals, err := s.Store.KPITotals(r.Context(), f)
	if err != nil {
		slog.Error("aggregate kpis: totals", "err", err)
		platform.Internal(w, "failed to compute kpis")
		return
	}
	weekly, err := s.Store.WeeklyBuckets(r.Context(), f, today)
	if err != nil {
		slog.Error("aggregate kpis: weekly", "err", err)
		platform.Internal(w, "failed to compute kpis")
		return
	}
	platform.WriteJSON(w, http.StatusOK, models.KPIRawData{Totals: totals, Weekly: weekly})
}

func (s *Server) handleAggregateTrend(w http.ResponseWriter, r *http.Request) {
	today := time.Now()
	buckets, err := s.Store.TrendBuckets(r.Context(), parseFilter(r, today))
	if err != nil {
		slog.Error("aggregate trend", "err", err)
		platform.Internal(w, "failed to compute trend")
		return
	}
	platform.WriteJSON(w, http.StatusOK, buckets)
}

func (s *Server) handleAggregateRegions(w http.ResponseWriter, r *http.Request) {
	today := time.Now()
	counts, err := s.Store.RegionCounts(r.Context(), parseFilter(r, today))
	if err != nil {
		slog.Error("aggregate regions", "err", err)
		platform.Internal(w, "failed to compute regions")
		return
	}
	platform.WriteJSON(w, http.StatusOK, counts)
}

func (s *Server) handleAggregateBenchmark(w http.ResponseWriter, r *http.Request) {
	today := time.Now()
	q := r.URL.Query()
	rows, err := s.Store.BenchmarkRows(r.Context(), parseFilter(r, today), q.Get("sort"), q.Get("dir") == "asc")
	if err != nil {
		slog.Error("aggregate benchmark", "err", err)
		platform.Internal(w, "failed to compute benchmark")
		return
	}
	platform.WriteJSON(w, http.StatusOK, rows)
}

func (s *Server) handleGetCampaign(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		platform.BadRequest(w, "id must be an integer")
		return
	}
	today := time.Now()
	c, ok, err := s.Store.ByID(r.Context(), id, today)
	if err != nil {
		slog.Error("get campaign", "id", id, "err", err)
		platform.Internal(w, "failed to load campaign")
		return
	}
	if !ok {
		platform.NotFound(w, fmt.Sprintf("no campaign with id %d", id))
		return
	}
	platform.WriteJSON(w, http.StatusOK, c.AsJSON())
}

func (s *Server) handleExportCampaigns(w http.ResponseWriter, r *http.Request) {
	today := time.Now()
	rows, err := s.Store.Export(r.Context(), parseFilter(r, today), today)
	if err != nil {
		slog.Error("export campaigns", "err", err)
		platform.Internal(w, "failed to export campaigns")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="campaign-report.csv"`)
	w.WriteHeader(http.StatusOK)

	writeCampaignsCSV(w, rows)
}

// writeCampaignsCSV is shared by the CSV-download export route and the
// emailed-report route (handleEmailReport) so the two can never drift
// apart — no CSV library, hand-rolled writes + csvEscape, same as before.
func writeCampaignsCSV(w io.Writer, rows []models.Campaign) {
	fmt.Fprint(w, "Subject,Subject Type,Category,Region,Ad Type,Platform,Start,Reach,Spend,Budget,Approval\n")
	for _, c := range rows {
		budget := ""
		if c.Budget != nil {
			budget = strconv.FormatInt(*c.Budget, 10)
		}
		fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s,%s,%d,%d,%s,%s\n",
			csvEscape(c.Subject), csvEscape(c.SubjectType), csvEscape(c.Category), csvEscape(c.Region),
			csvEscape(c.AdType), csvEscape(c.Platform), c.Start.Format("02-Jan-06"), c.Reach, c.Spend, budget,
			csvEscape(c.ApprovalStatus))
	}
}

// handleListCreatives backs GET /api/v1/campaigns/{id}/creatives — any
// authenticated role, same openness as reading the campaign itself.
func (s *Server) handleListCreatives(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		platform.BadRequest(w, "id must be an integer")
		return
	}

	if _, ok, err := s.Store.ByID(r.Context(), id, time.Now()); err != nil {
		slog.Error("list creatives: load campaign", "id", id, "err", err)
		platform.Internal(w, "failed to list creatives")
		return
	} else if !ok {
		platform.NotFound(w, fmt.Sprintf("no campaign with id %d", id))
		return
	}

	creatives, err := s.Store.ListCreatives(r.Context(), id)
	if err != nil {
		slog.Error("list creatives", "id", id, "err", err)
		platform.Internal(w, "failed to list creatives")
		return
	}
	out := make([]any, len(creatives))
	for i, c := range creatives {
		out[i] = c.AsJSON()
	}
	platform.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

// handleCreateCreative backs POST /api/v1/campaigns/{id}/creatives —
// editor or admin, same content-edit tier as creating the campaign
// itself (handleCreateCampaign). Checks the parent campaign exists first
// (via Store.ByID) rather than letting the creatives table's foreign key
// surface as a raw constraint-violation 500.
func (s *Server) handleCreateCreative(w http.ResponseWriter, r *http.Request) {
	if !platform.RequireRole(w, r, "only editors and admins can add a creative", platform.RoleEditor, platform.RoleAdmin) {
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		platform.BadRequest(w, "id must be an integer")
		return
	}

	if _, ok, err := s.Store.ByID(r.Context(), id, time.Now()); err != nil {
		slog.Error("create creative: load campaign", "id", id, "err", err)
		platform.Internal(w, "failed to create creative")
		return
	} else if !ok {
		platform.NotFound(w, fmt.Sprintf("no campaign with id %d", id))
		return
	}

	var req models.CreateCreativeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.BadRequest(w, "invalid JSON body")
		return
	}
	if req.Headline == "" {
		platform.BadRequest(w, "headline is required")
		return
	}
	if !models.CreativeTypeValid(req.CreativeType) {
		platform.BadRequest(w, `creativeType must be one of "image", "video", "carousel", "text"`)
		return
	}
	if req.Reach < 0 || req.Spend < 0 {
		platform.BadRequest(w, "reach and spend cannot be negative")
		return
	}

	created, err := s.Store.CreateCreative(r.Context(), id, req.Headline, req.CreativeType, req.Reach, req.Spend)
	if err != nil {
		slog.Error("create creative", "campaignId", id, "err", err)
		platform.Internal(w, "failed to create creative")
		return
	}
	platform.WriteJSON(w, http.StatusCreated, created.AsJSON())
}

// handleListAnomalies backs GET /api/v1/campaigns/anomalies — no admin
// gate, same openness as list/export (read-only). Deliberately ignores
// query-string filters (see models.AnomalyReport's doc comment): this is
// a global "what needs attention" signal, not a view of whatever's
// currently filtered.
func (s *Server) handleListAnomalies(w http.ResponseWriter, r *http.Request) {
	report, err := s.Store.AnomalyReport(r.Context(), time.Now())
	if err != nil {
		slog.Error("anomaly report", "err", err)
		platform.Internal(w, "failed to compute anomaly report")
		return
	}
	platform.WriteJSON(w, http.StatusOK, report)
}

// handleEmailReport backs POST /api/v1/campaigns/email-report — any
// authenticated role, same openness as export/anomalies (no admin gate).
// Filters come from the query string via the same parseFilter export
// uses, no request body — this is "the CSV export, delivered by email
// instead of a browser download." Sends to the caller's own account email
// (trusted X-User-Email, set by the gateway post-JWT-verification, same
// trust X-User-Role already gets), never a client-supplied recipient.
func (s *Server) handleEmailReport(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-User-Email")
	if email == "" {
		// Shouldn't happen behind the gateway (every /api/v1/* request is
		// authenticated first) — defensive, not a normal path.
		platform.WriteError(w, http.StatusUnauthorized, "unauthorized", "no authenticated user")
		return
	}

	sent, reason, err := s.GenerateAndSendReport(r.Context(), parseFilter(r, time.Now()), email)
	if err != nil {
		slog.Error("email report", "err", err)
		platform.Internal(w, "failed to build report")
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]any{"sent": sent, "reason": reason})
}

// GenerateAndSendReport builds the CSV for f and emails it to recipient —
// shared by handleEmailReport (on-demand, filtered to the caller's
// current view) and the scheduled digest ticker
// (cmd/server/main.go's runScheduledReports, unfiltered) so both send
// identically formatted reports through the same degrade-gracefully path.
// A nil err means the report was successfully *generated*; sent/reason
// separately say whether it was *delivered* — "not configured" is an
// expected state in an environment with no SMTP credentials (see
// platform/mail.go), not an error, so callers can report it plainly
// instead of claiming a success that didn't happen.
func (s *Server) GenerateAndSendReport(ctx context.Context, f store.Filter, recipient string) (sent bool, reason string, err error) {
	rows, err := s.Store.Export(ctx, f, time.Now())
	if err != nil {
		return false, "", fmt.Errorf("export: %w", err)
	}

	var csv bytes.Buffer
	writeCampaignsCSV(&csv, rows)

	subject := "Your campaign report"
	body := fmt.Sprintf("Attached: %d campaigns matching your filters.", len(rows))
	sendErr := s.Mailer.Send(recipient, subject, body, "campaign-report.csv", csv.Bytes())
	switch {
	case sendErr == nil:
		return true, "", nil
	case errors.Is(sendErr, platform.ErrMailerNotConfigured):
		return false, "email delivery is not configured in this environment", nil
	default:
		slog.Error("send report email", "to", recipient, "err", sendErr)
		return false, "email delivery failed", nil
	}
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// handleCreateCampaign validates the subject against catalog-service,
// snapshots its category, inserts the row, and publishes campaign.created —
// notifications-service fans that out over WebSocket and analytics-service's
// cache gets flushed by the same event.
//
// X-User-Role is set by the gateway after verifying the caller's JWT —
// see services/gateway/internal/api/authmw.go and docs/ARCHITECTURE.md.
// editor or admin: creating a campaign is a content-edit action, the same
// tier as setting its budget (handleUpdateCampaignBudget) — approving one
// is a separate tier (handleUpdateApprovalStatus), real separation of
// duties, see docs/ARCHITECTURE.md's RBAC section.
func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	if !platform.RequireRole(w, r, "only editors and admins can create campaigns", platform.RoleEditor, platform.RoleAdmin) {
		return
	}

	var req models.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.BadRequest(w, "invalid JSON body")
		return
	}
	if req.Subject == "" || req.SubjectType == "" || req.Region == "" || req.AdType == "" {
		platform.BadRequest(w, "subject, subjectType, region, and adType are required")
		return
	}

	subj, found, err := s.Catalog.Lookup(r.Context(), req.SubjectType, req.Subject)
	if err != nil {
		platform.Upstream(w, "could not validate subject: "+err.Error())
		return
	}
	if !found {
		platform.BadRequest(w, fmt.Sprintf("no such %s: %q", req.SubjectType, req.Subject))
		return
	}

	start, err := time.Parse("2006-01-02", req.Start)
	if err != nil {
		platform.BadRequest(w, "start must be YYYY-MM-DD")
		return
	}
	end, err := time.Parse("2006-01-02", req.End)
	if err != nil {
		platform.BadRequest(w, "end must be YYYY-MM-DD")
		return
	}

	today := time.Now()
	created, err := s.Store.Create(r.Context(), models.Campaign{
		Subject: subj.Name, SubjectType: subj.Type, Category: subj.Category,
		Region: req.Region, AdType: req.AdType, Platform: req.Platform,
		Start: start, End: end, Reach: req.Reach, Spend: req.Spend, Budget: req.Budget,
		// A real admin-submitted campaign starts as a review checkpoint,
		// not client-controlled — there's no request field that could set
		// this to "approved" directly, keeping the checkpoint meaningful.
		ApprovalStatus: "pending",
	}, today)
	if err != nil {
		slog.Error("create campaign", "err", err)
		platform.Internal(w, "failed to create campaign")
		return
	}

	if s.Events != nil {
		s.Events.Publish("campaign.created", created.AsJSON())
	}
	s.AuditPublish("campaign.created", r.Header.Get("X-User-Id"), r.Header.Get("X-User-Email"), r.Header.Get("X-User-Role"),
		created.ID, fmt.Sprintf("created %q", created.Subject))
	platform.WriteJSON(w, http.StatusCreated, created.AsJSON())
}

// handleUpdateCampaignBudget sets or clears a campaign's planned budget
// (see models.UpdateBudgetRequest for why this exists instead of a
// general edit form). Same editor-or-admin gate as handleCreateCampaign —
// setting a budget is the same content-edit tier as creating the
// campaign, not the separate approval tier handleUpdateApprovalStatus
// requires.
func (s *Server) handleUpdateCampaignBudget(w http.ResponseWriter, r *http.Request) {
	if !platform.RequireRole(w, r, "only editors and admins can change a campaign's budget", platform.RoleEditor, platform.RoleAdmin) {
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		platform.BadRequest(w, "id must be an integer")
		return
	}
	var req models.UpdateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.BadRequest(w, "invalid JSON body")
		return
	}
	if req.Budget != nil && *req.Budget < 0 {
		platform.BadRequest(w, "budget cannot be negative")
		return
	}

	today := time.Now()
	updated, ok, err := s.Store.UpdateBudget(r.Context(), id, req.Budget, today)
	if err != nil {
		slog.Error("update campaign budget", "id", id, "err", err)
		platform.Internal(w, "failed to update budget")
		return
	}
	if !ok {
		platform.NotFound(w, fmt.Sprintf("no campaign with id %d", id))
		return
	}
	details := "budget cleared"
	if req.Budget != nil {
		details = fmt.Sprintf("budget set to ₹%d", *req.Budget)
	}
	s.AuditPublish("campaign.budget_updated", r.Header.Get("X-User-Id"), r.Header.Get("X-User-Email"), r.Header.Get("X-User-Role"), id, details)
	platform.WriteJSON(w, http.StatusOK, updated.AsJSON())
}

// handleUpdateApprovalStatus moves a campaign between "pending",
// "approved", and "rejected" — a review checkpoint, not an access gate
// (see models.UpdateApprovalStatusRequest and db/init/02_campaigns.sql).
// A separate route from handleUpdateCampaignBudget on purpose: folding
// this into that PATCH's body would need to distinguish "field absent"
// from "field present" for two independent optional fields in one
// request (a real correctness risk — an approval-only PATCH could
// silently reset the budget), for no benefit over one small new route.
// approver or admin: reviewing a campaign is a separate tier from
// creating/budgeting one (handleCreateCampaign/handleUpdateCampaignBudget,
// gated to editor or admin) — real separation of duties between whoever
// proposes a campaign and whoever signs off on it.
func (s *Server) handleUpdateApprovalStatus(w http.ResponseWriter, r *http.Request) {
	if !platform.RequireRole(w, r, "only approvers and admins can change a campaign's approval status", platform.RoleApprover, platform.RoleAdmin) {
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		platform.BadRequest(w, "id must be an integer")
		return
	}
	var req models.UpdateApprovalStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.BadRequest(w, "invalid JSON body")
		return
	}
	if !models.ApprovalStatusValid(req.Status) {
		platform.BadRequest(w, `status must be one of "pending", "approved", "rejected"`)
		return
	}

	today := time.Now()
	updated, ok, err := s.Store.UpdateApprovalStatus(r.Context(), id, req.Status, today)
	if err != nil {
		slog.Error("update campaign approval status", "id", id, "err", err)
		platform.Internal(w, "failed to update approval status")
		return
	}
	if !ok {
		platform.NotFound(w, fmt.Sprintf("no campaign with id %d", id))
		return
	}
	s.AuditPublish("campaign.approval_updated", r.Header.Get("X-User-Id"), r.Header.Get("X-User-Email"), r.Header.Get("X-User-Role"),
		id, fmt.Sprintf("approval status set to %q", req.Status))
	platform.WriteJSON(w, http.StatusOK, updated.AsJSON())
}

// AuditPublish publishes a campaign.audit event — deliberately a separate
// NATS subject from campaign.created (published alongside it in
// handleCreateCampaign) so audit-service's new subscriber can't affect
// the existing ones on that subject (notifications-service,
// analytics-service's cache invalidation) in any way. Exported because
// cmd/server/main.go's demo ticker calls this too, with "system"/"system"
// in place of real X-User-* headers — a non-HTTP-triggered write has
// none — so its synthetic campaigns don't leave the audit trail looking
// mysteriously incomplete.
func (s *Server) AuditPublish(action, actorID, actorEmail, actorRole string, campaignID int64, details string) {
	if s.Events == nil {
		return
	}
	s.Events.Publish("campaign.audit", map[string]any{
		"action": action, "actorId": actorID, "actorEmail": actorEmail, "actorRole": actorRole,
		"campaignId": campaignID, "details": details,
	})
}
