// Package reports is the public contract of the reports service: the
// scheduled-report types and the cadence calendar maths.
package reports

import "time"

type Report struct {
	ID           string
	Name         string
	Enabled      bool
	Cadence      string
	Recipients   []string
	ScopeFilters map[string]any
	ScopeLabel   string
	Columns      []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
type ReportRun struct {
	ID       int64
	ReportID string
	RanAt    time.Time
	Result   string
	RowCount int
}
