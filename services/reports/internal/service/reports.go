package service

import (
	"campaigntrackerpro/platform/httpx"
	platformmail "campaigntrackerpro/platform/mail"
	"campaigntrackerpro/services/campaigns"
	"campaigntrackerpro/services/reports"
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"campaigntrackerpro/services/reports/internal/store"
)

var (
	ErrNotFound   = httpx.ErrNotFound
	ErrValidation = httpx.ErrValidation
)

// CampaignReader is the slice of campaign data a report needs to resolve
// its saved scope into rows. Narrow on purpose: reports never writes
// campaigns, so it cannot.
type CampaignReader interface {
	All(ctx context.Context) ([]campaigns.Campaign, error)
	List(ctx context.Context, p campaigns.ListParams) (campaigns.ListResult, error)
	Enrich(ctx context.Context, all []campaigns.Campaign) map[string]campaigns.CampaignRow
	Flagged(ctx context.Context) ([]campaigns.Campaign, error)
}

type Reports struct {
	Store     *store.Store
	Campaigns CampaignReader
	Mailer    platformmail.Mailer
}

func NewReports(s *store.Store, c CampaignReader, mailer platformmail.Mailer) *Reports {
	return &Reports{Store: s, Campaigns: c, Mailer: mailer}
}

func (r *Reports) List(ctx context.Context) ([]reports.Report, error) {
	return r.Store.ListReports(ctx)
}

func (r *Reports) Get(ctx context.Context, id string) (reports.Report, error) {
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
func (r *Reports) Create(ctx context.Context, in CreateReportInput) (reports.Report, error) {
	if strings.TrimSpace(in.Name) == "" {
		return reports.Report{}, ErrValidation
	}
	if !reports.IsValidCadence(in.Cadence) {
		return reports.Report{}, ErrValidation
	}
	for _, e := range in.Recipients {
		if !validEmail(e) {
			return reports.Report{}, ErrValidation
		}
	}
	columns := in.Columns
	if len(columns) == 0 {
		columns = campaigns.ReportColumns
	}
	for _, c := range columns {
		if !campaigns.IsValidReportColumn(c) {
			return reports.Report{}, ErrValidation
		}
	}
	return r.Store.CreateReport(ctx, reports.Report{
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

func (r *Reports) Update(ctx context.Context, id string, in UpdateReportInput) (reports.Report, error) {
	if in.Cadence != nil && !reports.IsValidCadence(*in.Cadence) {
		return reports.Report{}, ErrValidation
	}
	for _, e := range in.Recipients {
		if !validEmail(e) {
			return reports.Report{}, ErrValidation
		}
	}
	for _, c := range in.Columns {
		if !campaigns.IsValidReportColumn(c) {
			return reports.Report{}, ErrValidation
		}
	}
	return r.Store.UpdateReport(ctx, id, store.ReportUpdate{
		Name: in.Name, Enabled: in.Enabled, Cadence: in.Cadence,
		Recipients: in.Recipients, Columns: in.Columns,
	})
}

// scopeToParams turns a report's saved Campaigns-filter snapshot back into
// list params so Run/Test re-apply exactly the filters it was created from.
func scopeToParams(scope map[string]any) campaigns.ListParams {
	get := func(k string) string {
		v, ok := scope[k]
		if !ok || v == nil {
			return ""
		}
		s, _ := v.(string)
		return s
	}
	return campaigns.ListParams{
		Search: get("q"), SubjectType: get("type"), Category: get("category"),
		Region: get("region"), AdType: get("adType"), Status: get("status"),
		Approval: get("approval"), Range: get("range"), Sort: get("sort"), Dir: get("dir"),
	}
}

func (r *Reports) rowsForReport(ctx context.Context, rep reports.Report) ([]campaigns.CampaignRow, []campaigns.CategoryBenchmark, error) {
	all, err := r.Campaigns.All(ctx)
	if err != nil {
		return nil, nil, err
	}
	running := campaigns.Running(all)
	benchmarks := campaigns.CategoryBenchmarks(running)

	// on_flag reports are inherently about whatever's currently flagged,
	// not a saved filter snapshot — there's no "flagged" key in campaigns.ListParams
	// to scope by, so pull straight from the flagged set instead.
	if reports.Cadence(rep.Cadence) == reports.CadenceOnFlag {
		flagged, err := r.Campaigns.Flagged(ctx)
		if err != nil {
			return nil, nil, err
		}
		rowsByID := r.Campaigns.Enrich(ctx, all)
		rows := make([]campaigns.CampaignRow, 0, len(flagged))
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
func (r *Reports) Run(ctx context.Context, id string) (reports.ReportRun, error) {
	rep, err := r.Store.GetReport(ctx, id)
	if err != nil {
		return reports.ReportRun{}, err
	}
	rows, benchmarks, err := r.rowsForReport(ctx, rep)
	if err != nil {
		return reports.ReportRun{}, err
	}
	csvBytes := campaigns.ReportCSV(rows, benchmarks, rep.Columns)

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
	csvBytes := campaigns.ReportCSV(rows, benchmarks, rep.Columns)
	filename := fmt.Sprintf("%s-test.csv", slugify(rep.Name))
	return r.Mailer.Send(ctx, []string{testEmail}, "[Test] "+rep.Name, filename, csvBytes)
}

func (r *Reports) ListRuns(ctx context.Context, id string, limit int32) ([]reports.ReportRun, error) {
	return r.Store.ListReportRuns(ctx, id, limit)
}

// DueScheduledReports returns enabled, non-on_flag reports whose next
// calendar occurrence (computed from their last run, or creation time if
// they've never run) has arrived.
func (r *Reports) DueScheduledReports(ctx context.Context, now time.Time) ([]reports.Report, error) {
	all, err := r.Store.ListReports(ctx)
	if err != nil {
		return nil, err
	}
	var due []reports.Report
	for _, rep := range all {
		if !rep.Enabled || reports.Cadence(rep.Cadence) == reports.CadenceOnFlag {
			continue
		}
		last := rep.CreatedAt
		if run, err := r.Store.GetLastReportRun(ctx, rep.ID); err == nil {
			last = run.RanAt
		}
		next := reports.NextOccurrence(reports.Cadence(rep.Cadence), last)
		if !next.IsZero() && !next.After(now) {
			due = append(due, rep)
		}
	}
	return due, nil
}

func (r *Reports) OnFlagReports(ctx context.Context) ([]reports.Report, error) {
	all, err := r.Store.ListEnabledReportsByCadence(ctx, string(reports.CadenceOnFlag))
	if err != nil {
		return nil, err
	}
	return all, nil
}
