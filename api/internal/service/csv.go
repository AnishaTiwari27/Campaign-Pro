package service

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"campaigntrackerpro/internal/domain"
)

// ExportCSV renders /campaigns/export.csv's fixed, comprehensive column
// set — distinct from a Report's user-chosen Columns (see ReportCSV).
func ExportCSV(rows []CampaignRow) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write([]string{
		"ID", "Name", "Subject type", "Role", "Category", "Region", "Ad type", "Platform",
		"Status", "Days running", "Reach (L)", "Spend", "Budget", "Budget pace %", "CPM",
		"Frequency", "Index", "Approval", "Flag reason",
	})
	for _, r := range rows {
		w.Write([]string{
			r.ID, r.Name, string(r.SubjectType), r.Role, r.Category, r.Region, string(r.AdType), r.Platform,
			string(r.Status), fmt.Sprintf("%d", r.DaysRunning), fmt.Sprintf("%.2f", r.Reach),
			fmt.Sprintf("%d", r.Spend), fmt.Sprintf("%d", r.Budget), fmt.Sprintf("%.0f", r.Pace),
			fmt.Sprintf("%.0f", r.CPM), fmt.Sprintf("%.2f", r.Frequency), fmt.Sprintf("%.2f", r.Index),
			string(r.Approval), r.FlagReason,
		})
	}
	w.Flush()
	return buf.Bytes()
}

// ReportColumns are every column a scheduled report can be configured to
// include, in the order the Reports screen's checklist presents them.
var ReportColumns = []string{
	"Subject", "Subject type", "Category", "Region", "Ad type", "Platform",
	"Approval", "Reach", "Spend", "Budget pace", "CPM", "Waiting since",
	"Flag reason", "Category median", "Index",
}

func IsValidReportColumn(col string) bool {
	for _, c := range ReportColumns {
		if c == col {
			return true
		}
	}
	return false
}

// ReportCSV renders a scheduled report's chosen columns for its chosen rows.
func ReportCSV(rows []CampaignRow, benchmarks []domain.CategoryBenchmark, columns []string) []byte {
	medByCat := map[string]float64{}
	for _, b := range benchmarks {
		medByCat[b.Category] = b.MedianReach
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write(columns)
	for _, r := range rows {
		rec := make([]string, len(columns))
		for i, col := range columns {
			rec[i] = reportCell(r, col, medByCat[r.Category])
		}
		w.Write(rec)
	}
	w.Flush()
	return buf.Bytes()
}

func reportCell(r CampaignRow, column string, categoryMedian float64) string {
	switch column {
	case "Subject":
		return r.Name
	case "Subject type":
		return string(r.SubjectType)
	case "Category":
		return r.Category
	case "Region":
		return r.Region
	case "Ad type":
		return string(r.AdType)
	case "Platform":
		return r.Platform
	case "Approval":
		return string(r.Approval)
	case "Reach":
		return domain.FormatReach(r.Reach)
	case "Spend":
		return domain.FormatMoney(r.Spend)
	case "Budget pace":
		return fmt.Sprintf("%.0f%%", r.Pace)
	case "CPM":
		return domain.FormatCPM(r.CPM)
	case "Waiting since":
		if r.Approval == domain.ApprovalPending {
			return r.UpdatedAt.Format("2006-01-02")
		}
		return ""
	case "Flag reason":
		return r.FlagReason
	case "Category median":
		return domain.FormatReach(categoryMedian)
	case "Index":
		return domain.FormatIndex(r.Index)
	default:
		return ""
	}
}
