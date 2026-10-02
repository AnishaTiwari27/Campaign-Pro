package service

import (
	"bytes"
	"campaigntrackerpro/services/campaigns"
	"encoding/csv"
	"fmt"
)

// ExportCSV renders /campaigns/export.csv's fixed, comprehensive column
// set — distinct from a Report's user-chosen Columns (see ReportCSV).
func ExportCSV(rows []campaigns.CampaignRow) []byte {
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
