package creators_test

import (
	"encoding/csv"
	"strings"
	"testing"

	"campaigntrackerpro/services/creators"
)

// parse reads the rendered CSV back, so the tests assert on fields a
// spreadsheet would see rather than on a string that merely looks right.
func parse(t *testing.T, b []byte) [][]string {
	t.Helper()
	recs, err := csv.NewReader(strings.NewReader(string(b))).ReadAll()
	if err != nil {
		t.Fatalf("rendered CSV does not parse: %v", err)
	}
	return recs
}

func full() creators.CreatorPerformance {
	return creators.CreatorPerformance{
		Creator: creators.Creator{
			Name: "Prajakta Koli", Role: "Creator", Category: "Comedy",
			Region: "Maharashtra", Tier: creators.TierMacro, Followers: 7_200_000,
			PrimaryPlatform: "Instagram", Languages: []string{"Hindi", "English"},
		},
		Campaigns: 6, LiveCampaigns: 2,
		AvgReach: 18.4, TierMedian: 12.0, TierIndex: 1.53,
		CostPerLakh: 42_000, CostIndex: 0.88, TierCostPeers: 9,
		Consistency: 0.74, ConsistencyN: 6,
		AudienceReachPct: 25.6, Flagged: 1,
		Signals: creators.Signals{
			Scanned: true, ScanAt: scanAt, BrandPosts: 4,
			EngagementRate: 1.4, HasRate: true, PostsScanned: 25,
		},
	}
}

func TestEveryColumnRendersForACompleteRow(t *testing.T) {
	recs := parse(t, creators.RosterCSV([]creators.CreatorPerformance{full()}, creators.RosterColumns))
	if len(recs) != 2 {
		t.Fatalf("want a header and one row, got %d records", len(recs))
	}
	if len(recs[0]) != len(creators.RosterColumns) {
		t.Fatalf("header has %d columns, want %d", len(recs[0]), len(creators.RosterColumns))
	}
	for i, col := range creators.RosterColumns {
		if recs[1][i] == "" {
			t.Errorf("column %q came out blank for a row that has every figure", col)
		}
	}
}

// The rule the whole file exists for. The screen withholds these figures
// because they do not mean anything; a CSV that fills them in undoes
// that, and a spreadsheet is where the caveat gets lost.
func TestFiguresTheScreenWouldWithholdComeOutBlank(t *testing.T) {
	for _, tc := range []struct {
		name   string
		column string
		row    creators.CreatorPerformance
	}{
		{
			name: "consistency from a single campaign is arithmetic, not evidence",
			// A perfect 1.0 off one campaign is the dangerous value: it
			// is the most flattering number in the file and the least
			// supported.
			column: "Consistency",
			row: func() creators.CreatorPerformance {
				r := full()
				r.Consistency, r.ConsistencyN = 1.0, 1
				return r
			}(),
		},
		{
			name:   "a cost index with no priced peers to compare against",
			column: "Cost index",
			row: func() creators.CreatorPerformance {
				r := full()
				r.CostIndex, r.TierCostPeers = 1.0, 1
				return r
			}(),
		},
		{
			name:   "average reach for a creator with no campaigns",
			column: "Avg reach",
			row: func() creators.CreatorPerformance {
				r := full()
				r.Campaigns, r.AvgReach = 0, 0
				return r
			}(),
		},
		{
			name:   "a tier index with no campaigns behind it",
			column: "Tier index",
			row: func() creators.CreatorPerformance {
				r := full()
				r.Campaigns = 0
				return r
			}(),
		},
		{
			name:   "an unpriced creator has no cost per lakh",
			column: "Cost per lakh",
			row: func() creators.CreatorPerformance {
				r := full()
				r.CostPerLakh = 0
				return r
			}(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Name rides along because encoding/csv writes a record of
			// one empty field as a blank line, and its reader skips
			// blank lines — so a single-column assertion would be
			// testing the reader, not the renderer. Every real export
			// carries a name anyway.
			recs := parse(t, creators.RosterCSV([]creators.CreatorPerformance{tc.row}, []string{"Name", tc.column}))
			if len(recs) != 2 {
				t.Fatalf("want a header and one row, got %d records", len(recs))
			}
			if got := recs[1][1]; got != "" {
				t.Errorf("%s: got %q, want an empty cell", tc.column, got)
			}
		})
	}
}

// Followers is the one figure that is always real — it comes from the
// platform, not from our campaign history — so it is never withheld.
func TestFollowersSurviveACreatorWithNoCampaigns(t *testing.T) {
	r := full()
	r.Campaigns, r.LiveCampaigns = 0, 0
	recs := parse(t, creators.RosterCSV([]creators.CreatorPerformance{r}, []string{"Followers", "Campaigns"}))
	if recs[1][0] != "7200000" {
		t.Errorf("followers = %q, want the raw count", recs[1][0])
	}
	if recs[1][1] != "0" {
		t.Errorf("campaigns = %q, want an explicit zero rather than a blank", recs[1][1])
	}
}

// Raw digits, not "72L": the point of an export is the next tool, and a
// formatted string has to be undone before it can be summed.
func TestFollowersExportUnformatted(t *testing.T) {
	recs := parse(t, creators.RosterCSV([]creators.CreatorPerformance{full()}, []string{"Followers"}))
	if strings.ContainsAny(recs[1][0], "L,₹ ") {
		t.Errorf("followers = %q — an export has to be machine-readable", recs[1][0])
	}
}

// A comma inside a quoted field is legal CSV and still splits the column
// in several spreadsheet importers, shifting every column after it.
func TestLanguagesDoNotUseTheDelimiter(t *testing.T) {
	recs := parse(t, creators.RosterCSV([]creators.CreatorPerformance{full()}, []string{"Languages", "Campaigns"}))
	if strings.Contains(recs[1][0], ",") {
		t.Errorf("languages = %q — must not contain the delimiter", recs[1][0])
	}
	if recs[1][0] != "Hindi; English" {
		t.Errorf("languages = %q", recs[1][0])
	}
}

// An address is held under a lawful basis for a stated purpose, and a
// spreadsheet leaves the audit trail behind. Adding a contact column
// here would route around the consent screening entirely.
func TestContactDetailsAreNotExportable(t *testing.T) {
	for _, col := range creators.RosterColumns {
		low := strings.ToLower(col)
		for _, banned := range []string{"email", "phone", "contact", "address", "handle"} {
			if strings.Contains(low, banned) {
				t.Errorf("column %q exposes contact data — that export belongs with the consent records", col)
			}
		}
	}
}

func TestUnknownColumnsRenderBlankRatherThanFailing(t *testing.T) {
	recs := parse(t, creators.RosterCSV([]creators.CreatorPerformance{full()}, []string{"Name", "Nonsense"}))
	if recs[1][0] == "" {
		t.Error("a valid column alongside an invalid one must still render")
	}
	if recs[1][1] != "" {
		t.Errorf("unknown column rendered %q", recs[1][1])
	}
	if creators.IsValidRosterColumn("Nonsense") {
		t.Error("IsValidRosterColumn accepted a column that does not exist")
	}
	for _, col := range creators.RosterColumns {
		if !creators.IsValidRosterColumn(col) {
			t.Errorf("IsValidRosterColumn rejected %q, which is in RosterColumns", col)
		}
	}
}

func TestNoRowsStillRendersAHeader(t *testing.T) {
	recs := parse(t, creators.RosterCSV(nil, creators.RosterColumns))
	if len(recs) != 1 {
		t.Fatalf("want the header alone, got %d records", len(recs))
	}
}

// An unscanned creator must not export a zero brand-post count: that
// is our backlog presented as a finding about them.
func TestUnscannedSignalsExportBlank(t *testing.T) {
	r := full()
	r.Signals = creators.Signals{} // never scanned
	recs := parse(t, creators.RosterCSV([]creators.CreatorPerformance{r},
		[]string{"Name", "Brand posts", "Engagement rate", "Last scanned", "Stage"}))
	for i, col := range []string{"Brand posts", "Engagement rate", "Last scanned"} {
		if got := recs[1][i+1]; got != "" {
			t.Errorf("%s = %q for an unscanned creator, want blank", col, got)
		}
	}
	if recs[1][4] != "Not yet checked" {
		t.Errorf("stage = %q, want it to say nobody has looked", recs[1][4])
	}
}

// Scanned with nothing found is a real finding and exports as zero,
// not as a blank.
func TestAScannedCreatorWithNoBrandWorkExportsZero(t *testing.T) {
	r := full()
	r.Signals = creators.Signals{Scanned: true, ScanAt: scanAt, BrandPosts: 0, EngagementRate: 1.4, HasRate: true}
	recs := parse(t, creators.RosterCSV([]creators.CreatorPerformance{r},
		[]string{"Name", "Brand posts", "Engagement rate", "Stage"}))
	if recs[1][1] != "0" {
		t.Errorf("brand posts = %q, want an explicit 0", recs[1][1])
	}
	if recs[1][2] != "1.40%" {
		t.Errorf("engagement = %q", recs[1][2])
	}
	if recs[1][3] != "None in recent posts" {
		t.Errorf("stage = %q", recs[1][3])
	}
}
