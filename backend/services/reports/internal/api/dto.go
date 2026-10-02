package api

import (
	"campaigntrackerpro/services/reports"
	"time"
)

const rfc3339 = time.RFC3339

type ReportDTO struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Enabled      bool           `json:"enabled"`
	Cadence      string         `json:"cadence"`
	CadenceLabel string         `json:"cadenceLabel"`
	Recipients   []string       `json:"recipients"`
	ScopeFilters map[string]any `json:"scopeFilters"`
	ScopeLabel   string         `json:"scopeLabel"`
	Columns      []string       `json:"columns"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
	LastRun      *ReportRunDTO  `json:"lastRun,omitempty"`
}

func reportDTO(r reports.Report) ReportDTO {
	return ReportDTO{
		ID: r.ID, Name: r.Name, Enabled: r.Enabled, Cadence: r.Cadence,
		CadenceLabel: reports.CadenceLabel(reports.Cadence(r.Cadence)),
		Recipients:   r.Recipients, ScopeFilters: r.ScopeFilters, ScopeLabel: r.ScopeLabel,
		Columns: r.Columns, CreatedAt: r.CreatedAt.Format(rfc3339), UpdatedAt: r.UpdatedAt.Format(rfc3339),
	}
}

func reportDTOs(rs []reports.Report) []ReportDTO {
	out := make([]ReportDTO, len(rs))
	for i, r := range rs {
		out[i] = reportDTO(r)
	}
	return out
}

type ReportRunDTO struct {
	RanAt    string `json:"ranAt"`
	Result   string `json:"result"`
	RowCount int    `json:"rowCount"`
}

func reportRunDTO(r reports.ReportRun) ReportRunDTO {
	return ReportRunDTO{RanAt: r.RanAt.Format(rfc3339), Result: r.Result, RowCount: r.RowCount}
}

func reportRunDTOs(rs []reports.ReportRun) []ReportRunDTO {
	out := make([]ReportRunDTO, len(rs))
	for i, r := range rs {
		out[i] = reportRunDTO(r)
	}
	return out
}
