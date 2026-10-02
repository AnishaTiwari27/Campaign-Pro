package store

import (
	"encoding/json"

	"campaigntrackerpro/db/gen"
	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/services/reports"
)

func toDomainReport(r gen.Report) reports.Report {
	var scope map[string]any
	_ = json.Unmarshal(r.ScopeFilters, &scope)
	return reports.Report{
		ID:           database.UuidToString(r.ID),
		Name:         r.Name,
		Enabled:      r.Enabled,
		Cadence:      string(r.Cadence),
		Recipients:   r.Recipients,
		ScopeFilters: scope,
		ScopeLabel:   r.ScopeLabel,
		Columns:      r.Columns,
		CreatedAt:    database.TimeOf(r.CreatedAt),
		UpdatedAt:    database.TimeOf(r.UpdatedAt),
	}
}

func toDomainReports(rs []gen.Report) []reports.Report {
	out := make([]reports.Report, len(rs))
	for i, r := range rs {
		out[i] = toDomainReport(r)
	}
	return out
}

func toDomainReportRun(r gen.ReportRun) reports.ReportRun {
	return reports.ReportRun{
		ID:       r.ID,
		ReportID: database.UuidToString(r.ReportID),
		RanAt:    database.TimeOf(r.RanAt),
		Result:   string(r.Result),
		RowCount: int(r.RowCount),
	}
}

func toDomainReportRuns(rs []gen.ReportRun) []reports.ReportRun {
	out := make([]reports.ReportRun, len(rs))
	for i, r := range rs {
		out[i] = toDomainReportRun(r)
	}
	return out
}
