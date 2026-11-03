// Roster CSV rendering lives in the creators contract because the rows it
// renders are creators: the caller chooses which columns, the shape of a
// creator row is owned here.
package creators

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"campaigntrackerpro/platform/units"
)

// RosterColumns are every column an export can include, in the order the
// Influencers screen presents them.
//
// Contact details are deliberately absent. An address is held under a
// lawful basis for a stated purpose, and a spreadsheet leaves the app's
// access control and audit trail behind — so exporting contacts is a
// different export with different rules, and it belongs with the consent
// data rather than here.
var RosterColumns = []string{
	"Name", "Role", "Category", "Region", "Tier", "Followers",
	"Primary platform", "Languages", "Campaigns", "Live campaigns",
	"Avg reach", "Tier median", "Tier index", "Cost per lakh",
	"Cost index", "Consistency", "Audience reach", "Flagged",
}

func IsValidRosterColumn(col string) bool {
	for _, c := range RosterColumns {
		if c == col {
			return true
		}
	}
	return false
}

// RosterCSV renders the chosen columns for the chosen creators.
func RosterCSV(rows []CreatorPerformance, columns []string) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write(columns)
	for _, r := range rows {
		rec := make([]string, len(columns))
		for i, col := range columns {
			rec[i] = rosterCell(r, col)
		}
		w.Write(rec)
	}
	w.Flush()
	return buf.Bytes()
}

// rosterCell renders one cell, and leaves a figure blank wherever the
// screen would say "unknown".
//
// This is the whole reason the file needs care. The Influencers screen
// withholds a consistency score drawn from one campaign and a cost index
// with no priced peers to compare against, because neither means
// anything. A CSV that printed 1.0 in those cells would undo that — and
// a spreadsheet is exactly where a caveat gets lost, because the number
// gets pasted into a deck without the screen that qualified it. An empty
// cell reads as "we do not know", which is true; 1.0 reads as "perfectly
// consistent", which is not.
func rosterCell(r CreatorPerformance, column string) string {
	switch column {
	case "Name":
		return r.Creator.Name
	case "Role":
		return r.Creator.Role
	case "Category":
		return r.Creator.Category
	case "Region":
		return r.Creator.Region
	case "Tier":
		return TierLabel(r.Creator.Tier)
	case "Followers":
		return fmt.Sprintf("%d", r.Creator.Followers)
	case "Primary platform":
		return r.Creator.PrimaryPlatform
	case "Languages":
		// Semicolons, not commas: a comma inside a quoted field is legal
		// CSV and still splits the column in several spreadsheet
		// importers, which silently shifts every column after it.
		return strings.Join(r.Creator.Languages, "; ")
	case "Campaigns":
		return fmt.Sprintf("%d", r.Campaigns)
	case "Live campaigns":
		return fmt.Sprintf("%d", r.LiveCampaigns)
	case "Avg reach":
		if r.Campaigns == 0 {
			return ""
		}
		return units.FormatReach(r.AvgReach)
	case "Tier median":
		return units.FormatReach(r.TierMedian)
	case "Tier index":
		if r.Campaigns == 0 || r.TierMedian == 0 {
			return ""
		}
		return units.FormatIndex(r.TierIndex)
	case "Cost per lakh":
		if r.CostPerLakh == 0 {
			return ""
		}
		return units.FormatMoney(int64(r.CostPerLakh))
	case "Cost index":
		// Alone in their tier means there is nothing to be an index of.
		if r.TierCostPeers <= 1 || r.CostIndex == 0 {
			return ""
		}
		return units.FormatIndex(r.CostIndex)
	case "Consistency":
		// Below two campaigns the score is arithmetic, not evidence.
		if r.ConsistencyN < 2 {
			return ""
		}
		return fmt.Sprintf("%.0f%%", r.Consistency*100)
	case "Audience reach":
		if r.Campaigns == 0 {
			return ""
		}
		return fmt.Sprintf("%.1f%%", r.AudienceReachPct)
	case "Flagged":
		return fmt.Sprintf("%d", r.Flagged)
	default:
		return ""
	}
}
