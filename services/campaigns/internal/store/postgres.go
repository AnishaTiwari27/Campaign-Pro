// Package store is campaigns-service's only door into Postgres — every
// query is qualified against "campaigns.campaigns", the only table this
// service (and only this service) can see.
package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"campaigntrackerpro/services/campaigns/internal/models"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Filter mirrors the shared query params documented in docs/API_CONTRACT.md.
// Empty string / zero fields mean "no filter", same convention the
// original in-memory store used.
type Filter struct {
	Category, Region, AdType, SubjectType, Subject, Query string
	Days                                                  int
	Today                                                 time.Time
}

// whereClause builds a parameterized WHERE clause (and its args) from f,
// shared by List, Export, and Count so filter semantics can't drift between
// the paginated and unpaginated read paths.
func whereClause(f Filter) (string, []any) {
	clauses := []string{"1=1"}
	var args []any

	add := func(column, value string) {
		if value == "" || value == "All" {
			return
		}
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	add("subject_category", f.Category)
	add("region", f.Region)
	add("ad_type", f.AdType)
	add("subject_type", f.SubjectType)
	add("subject_name", f.Subject)

	if f.Query != "" {
		args = append(args, "%"+strings.ToLower(f.Query)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(subject_name) LIKE $%d", len(args)))
	}
	if f.Days > 0 {
		today := f.Today
		if today.IsZero() {
			today = time.Now()
		}
		cutoff := today.AddDate(0, 0, -f.Days)
		args = append(args, cutoff)
		clauses = append(clauses, fmt.Sprintf("start_date >= $%d", len(args)))
	}
	return strings.Join(clauses, " AND "), args
}

// List returns page/limit rows matching f (start_date desc) plus the total
// count of matching rows before pagination. today drives each row's
// derived Status ("Live"/"Completed") — passed in rather than read from
// time.Now() here so a single request sees one consistent "now" across
// every row, and so tests can pin it.
func (s *Store) List(ctx context.Context, f Filter, page, limit int, today time.Time) ([]models.Campaign, int, error) {
	where, args := whereClause(f)

	var total int
	countSQL := "SELECT count(*) FROM campaigns.campaigns WHERE " + where
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	listArgs := append(append([]any{}, args...), limit, offset)
	listSQL := fmt.Sprintf(`SELECT %s FROM campaigns.campaigns WHERE %s
		ORDER BY start_date DESC LIMIT $%d OFFSET $%d`,
		selectColumns, where, len(listArgs)-1, len(listArgs))

	rows, err := s.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	campaigns, err := scanAll(rows, today)
	return campaigns, total, err
}

// Export returns every row matching f, unpaginated — backs both the CSV
// export endpoint and analytics-service's aggregation calls.
func (s *Store) Export(ctx context.Context, f Filter, today time.Time) ([]models.Campaign, error) {
	where, args := whereClause(f)
	sql := fmt.Sprintf(`SELECT %s FROM campaigns.campaigns WHERE %s ORDER BY start_date DESC`, selectColumns, where)

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAll(rows, today)
}

func (s *Store) ByID(ctx context.Context, id int64, today time.Time) (models.Campaign, bool, error) {
	sql := fmt.Sprintf(`SELECT %s FROM campaigns.campaigns WHERE id = $1`, selectColumns)
	rows, err := s.pool.Query(ctx, sql, id)
	if err != nil {
		return models.Campaign{}, false, err
	}
	defer rows.Close()

	campaigns, err := scanAll(rows, today)
	if err != nil || len(campaigns) == 0 {
		return models.Campaign{}, false, err
	}
	return campaigns[0], true, nil
}

// Create inserts a new campaign row and returns it, status/pacing computed
// relative to today.
func (s *Store) Create(ctx context.Context, c models.Campaign, today time.Time) (models.Campaign, error) {
	// A caller that never sets ApprovalStatus (the demo ticker, the seed
	// script — both call Create directly, bypassing handleCreateCampaign)
	// gets the same 'approved' a real admin-created campaign eventually
	// reaches, so nothing already running changes behavior. Only
	// handleCreateCampaign explicitly passes "pending".
	approvalStatus := c.ApprovalStatus
	if approvalStatus == "" {
		approvalStatus = "approved"
	}

	sql := `INSERT INTO campaigns.campaigns
		(subject_name, subject_type, subject_category, region, ad_type, platform,
		 start_date, end_date, reach, spend_paise, budget_paise, approval_status, source)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id`

	var id int64
	err := s.pool.QueryRow(ctx, sql,
		c.Subject, c.SubjectType, c.Category, c.Region, c.AdType, c.Platform,
		c.Start, c.End, c.Reach, c.Spend*100, rupeesToPaise(c.Budget), approvalStatus, "manual",
	).Scan(&id)
	if err != nil {
		return models.Campaign{}, err
	}
	c.ID = id
	c.Status = statusOf(c.End, today)
	c.Pacing = models.PacingOf(c.Budget, c.Spend, c.Start, c.End, today)
	c.ApprovalStatus = approvalStatus
	return c, nil
}

// UpdateApprovalStatus sets a campaign's review status — the only other
// mutable field on an existing campaign besides budget (see UpdateBudget).
// A separate method/route rather than folding into UpdateBudget, same
// reasoning as models.UpdateApprovalStatusRequest's doc comment.
func (s *Store) UpdateApprovalStatus(ctx context.Context, id int64, status string, today time.Time) (models.Campaign, bool, error) {
	sql := fmt.Sprintf(`UPDATE campaigns.campaigns SET approval_status = $1, updated_at = now()
		WHERE id = $2 RETURNING %s`, selectColumns)

	rows, err := s.pool.Query(ctx, sql, status, id)
	if err != nil {
		return models.Campaign{}, false, err
	}
	defer rows.Close()

	campaigns, err := scanAll(rows, today)
	if err != nil || len(campaigns) == 0 {
		return models.Campaign{}, false, err
	}
	return campaigns[0], true, nil
}

// UpdateBudget sets (or clears, if budgetRupees is nil) a campaign's
// planned budget — the only mutable field on an existing campaign right
// now (see models.UpdateBudgetRequest). Returns (_, false, nil) if id
// doesn't exist, the same "not found is not an error" shape ByID uses.
func (s *Store) UpdateBudget(ctx context.Context, id int64, budgetRupees *int64, today time.Time) (models.Campaign, bool, error) {
	sql := fmt.Sprintf(`UPDATE campaigns.campaigns SET budget_paise = $1, updated_at = now()
		WHERE id = $2 RETURNING %s`, selectColumns)

	rows, err := s.pool.Query(ctx, sql, rupeesToPaise(budgetRupees), id)
	if err != nil {
		return models.Campaign{}, false, err
	}
	defer rows.Close()

	campaigns, err := scanAll(rows, today)
	if err != nil || len(campaigns) == 0 {
		return models.Campaign{}, false, err
	}
	return campaigns[0], true, nil
}

// rupeesToPaise converts the API's rupee amounts to the paise this table
// stores, preserving nil (no budget set) rather than turning it into 0 —
// a real, deliberate zero-rupee budget and "no budget was ever set" are
// different things (see models.PacingOf, which treats both as "").
func rupeesToPaise(rupees *int64) *int64 {
	if rupees == nil {
		return nil
	}
	paise := *rupees * 100
	return &paise
}

const selectColumns = `id, subject_name, subject_type, subject_category, region, ad_type, platform,
	start_date, end_date, reach, spend_paise, budget_paise, approval_status`

func scanAll(rows pgx.Rows, today time.Time) ([]models.Campaign, error) {
	var out []models.Campaign
	for rows.Next() {
		var c models.Campaign
		var spendPaise int64
		var budgetPaise *int64
		if err := rows.Scan(&c.ID, &c.Subject, &c.SubjectType, &c.Category, &c.Region, &c.AdType,
			&c.Platform, &c.Start, &c.End, &c.Reach, &spendPaise, &budgetPaise, &c.ApprovalStatus); err != nil {
			return nil, err
		}
		c.Spend = spendPaise / 100
		if budgetPaise != nil {
			budgetRupees := *budgetPaise / 100
			c.Budget = &budgetRupees
		}
		c.Status = statusOf(c.End, today)
		c.Pacing = models.PacingOf(c.Budget, c.Spend, c.Start, c.End, today)
		out = append(out, c)
	}
	return out, rows.Err()
}

func statusOf(end, today time.Time) string {
	if end.After(today) {
		return "Live"
	}
	return "Completed"
}
