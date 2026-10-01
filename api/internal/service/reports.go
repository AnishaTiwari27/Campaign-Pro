package service

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"campaigntrackerpro/internal/domain"
	"campaigntrackerpro/internal/store"
)

type Reports struct {
	Store     *store.Store
	Campaigns *Campaigns
	Mailer    Mailer
}

func NewReports(s *store.Store, c *Campaigns, mailer Mailer) *Reports {
	return &Reports{Store: s, Campaigns: c, Mailer: mailer}
}

func (r *Reports) List(ctx context.Context) ([]domain.Report, error) { return r.Store.ListReports(ctx) }

func (r *Reports) Get(ctx context.Context, id string) (domain.Report, error) {
	return r.Store.GetReport(ctx, id)
}

func validEmail(e string) bool {
	_, err := mail.ParseAddress(e)
	return err == nil
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = slugPattern.ReplaceAllString(strings.ToLower(s), "-")
	return strings.Trim(s, "-")
}

type CreateReportInput struct {
	Name         string
	Cadence      string
	Recipients   []string
	ScopeFilters map[string]any
	ScopeLabel   string
	Columns      []string
}

// Create validates cadence/recipients/columns and defaults Columns to every
// available column when the caller doesn't choose a subset.
func (r *Reports) Create(ctx context.Context, in CreateReportInput) (domain.Report, error) {
	if strings.TrimSpace(in.Name) == "" {
		return domain.Report{}, ErrValidation
	}
	if !domain.IsValidCadence(in.Cadence) {
		return domain.Report{}, ErrValidation
	}
	for _, e := range in.Recipients {
		if !validEmail(e) {
			return domain.Report{}, ErrValidation
		}
	}
	columns := in.Columns
	if len(columns) == 0 {
		columns = ReportColumns
	}
	for _, c := range columns {
		if !IsValidReportColumn(c) {
			return domain.Report{}, ErrValidation
		}
	}
	return r.Store.CreateReport(ctx, domain.Report{
		Name: in.Name, Enabled: true, Cadence: in.Cadence, Recipients: in.Recipients,
		ScopeFilters: in.ScopeFilters, ScopeLabel: in.ScopeLabel, Columns: columns,
	})
}

type UpdateReportInput struct {
	Name       *string
	Enabled    *bool
	Cadence    *string
	Recipients []string
	Columns    []string
}

func (r *Reports) Update(ctx context.Context, id string, in UpdateReportInput) (domain.Report, error) {
	if in.Cadence != nil && !domain.IsValidCadence(*in.Cadence) {
		return domain.Report{}, ErrValidation
	}
	for _, e := range in.Recipients {
		if !validEmail(e) {
			return domain.Report{}, ErrValidation
		}
	}
	for _, c := range in.Columns {
		if !IsValidReportColumn(c) {
			return domain.Report{}, ErrValidation
		}
	}
	return r.Store.UpdateReport(ctx, id, store.ReportUpdate{
		Name: in.Name, Enabled: in.Enabled, Cadence: in.Cadence,
		Recipients: in.Recipients, Columns: in.Columns,
	})
}

// scopeToParams turns a report's saved Campaigns-filter snapshot back into
// list params so Run/Test re-apply exactly the filters it was created from.
func scopeToParams(scope map[string]any) ListParams {
	get := func(k string) string {
		v, ok := scope[k]
		if !ok || v == nil {
			return ""
		}
		s, _ := v.(string)
		return s
	}
	return ListParams{
		Search: get("q"), SubjectType: get("type"), Category: get("category"),
		Region: get("region"), AdType: get("adType"), Status: get("status"),
		Approval: get("approval"), Range: get("range"), Sort: get("sort"), Dir: get("dir"),
	}
}

func (r *Reports) rowsForReport(ctx context.Context, rep domain.Report) ([]CampaignRow, []domain.CategoryBenchmark, error) {
	all, err := r.Campaigns.Store.ListCampaigns(ctx)
	if err != nil {
		return nil, nil, err
	}
	running := domain.Running(all)
	benchmarks := domain.CategoryBenchmarks(running)

	// on_flag reports are inherently about whatever's currently flagged,
	// not a saved filter snapshot — there's no "flagged" key in ListParams
	// to scope by, so pull straight from the flagged set instead.
	if domain.Cadence(rep.Cadence) == domain.CadenceOnFlag {
		flagged, err := r.Campaigns.Store.ListFlaggedCampaigns(ctx)
		if err != nil {
			return nil, nil, err
		}
		rowsByID, _, _, _ := r.Campaigns.enrich(ctx, all)
		rows := make([]CampaignRow, 0, len(flagged))
		for _, c := range flagged {
			rows = append(rows, rowsByID[c.ID])
		}
		return rows, benchmarks, nil
	}

	params := scopeToParams(rep.ScopeFilters)
	params.Page = 1
	params.Per = len(all) + 1
	listResult, err := r.Campaigns.List(ctx, params)
	if err != nil {
		return nil, nil, err
	}
	return listResult.Items, benchmarks, nil
}

// Run executes a report against its saved scope, emails its real
// recipients, and always writes a report_runs row — this is both "Run now"
// and what the scheduler calls when a cadence's next occurrence arrives.
func (r *Reports) Run(ctx context.Context, id string) (domain.ReportRun, error) {
	rep, err := r.Store.GetReport(ctx, id)
	if err != nil {
		return domain.ReportRun{}, err
	}
	rows, benchmarks, err := r.rowsForReport(ctx, rep)
	if err != nil {
		return domain.ReportRun{}, err
	}
	csvBytes := ReportCSV(rows, benchmarks, rep.Columns)

	result := "Not sending"
	if len(rep.Recipients) > 0 {
		filename := fmt.Sprintf("%s-%s.csv", slugify(rep.Name), time.Now().Format("2006-01-02"))
		if err := r.Mailer.Send(ctx, rep.Recipients, rep.Name, filename, csvBytes); err != nil {
			result = "Failed"
		} else {
			result = "Delivered"
		}
	}
	return r.Store.CreateReportRun(ctx, id, result, len(rows))
}

// Test sends a one-off copy to the requesting user only; it doesn't touch
// the report's recipients or run history, since it isn't a real scheduled
// or manual "run."
func (r *Reports) Test(ctx context.Context, id, testEmail string) error {
	rep, err := r.Store.GetReport(ctx, id)
	if err != nil {
		return err
	}
	rows, benchmarks, err := r.rowsForReport(ctx, rep)
	if err != nil {
		return err
	}
	csvBytes := ReportCSV(rows, benchmarks, rep.Columns)
	filename := fmt.Sprintf("%s-test.csv", slugify(rep.Name))
	return r.Mailer.Send(ctx, []string{testEmail}, "[Test] "+rep.Name, filename, csvBytes)
}

func (r *Reports) ListRuns(ctx context.Context, id string, limit int32) ([]domain.ReportRun, error) {
	return r.Store.ListReportRuns(ctx, id, limit)
}

// DueScheduledReports returns enabled, non-on_flag reports whose next
// calendar occurrence (computed from their last run, or creation time if
// they've never run) has arrived.
func (r *Reports) DueScheduledReports(ctx context.Context, now time.Time) ([]domain.Report, error) {
	all, err := r.Store.ListReports(ctx)
	if err != nil {
		return nil, err
	}
	var due []domain.Report
	for _, rep := range all {
		if !rep.Enabled || domain.Cadence(rep.Cadence) == domain.CadenceOnFlag {
			continue
		}
		last := rep.CreatedAt
		if run, err := r.Store.GetLastReportRun(ctx, rep.ID); err == nil {
			last = run.RanAt
		}
		next := domain.NextOccurrence(domain.Cadence(rep.Cadence), last)
		if !next.IsZero() && !next.After(now) {
			due = append(due, rep)
		}
	}
	return due, nil
}

func (r *Reports) OnFlagReports(ctx context.Context) ([]domain.Report, error) {
	all, err := r.Store.ListEnabledReportsByCadence(ctx, string(domain.CadenceOnFlag))
	if err != nil {
		return nil, err
	}
	return all, nil
}
