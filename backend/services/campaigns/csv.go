// Report CSV rendering lives in the campaigns contract because the rows
// it renders are campaigns: the reports service chooses which columns and
// which scope, but the shape of a campaign row is owned here.
package campaigns

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"campaigntrackerpro/platform/units"
)

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
func ReportCSV(rows []CampaignRow, benchmarks []CategoryBenchmark, columns []string) []byte {
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
		return units.FormatReach(r.Reach)
	case "Spend":
		return units.FormatMoney(r.Spend)
	case "Budget pace":
		return fmt.Sprintf("%.0f%%", r.Pace)
	case "CPM":
		return units.FormatCPM(r.CPM)
	case "Waiting since":
		if r.Approval == ApprovalPending {
			return r.UpdatedAt.Format("2006-01-02")
		}
		return ""
	case "Flag reason":
		return r.FlagReason
	case "Category median":
		return units.FormatReach(categoryMedian)
	case "Index":
		return units.FormatIndex(r.Index)
	default:
		return ""
	}
}
