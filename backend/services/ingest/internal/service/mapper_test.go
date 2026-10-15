package service

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"campaigntrackerpro/services/ingest"
)

type mapperModel struct {
	reply string
	calls int
}

func (m *mapperModel) Complete(context.Context, string, string) (string, error) {
	m.calls++
	return m.reply, nil
}

func mapperSvc(m *mapperModel) *Sync {
	s := New(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if m != nil {
		s = s.WithMapper(m)
	}
	return s
}

// A Google Ads export uses its own words for everything.
var googleHeaders = []string{"Campaign ID", "Campaign", "Campaign state", "Campaign type", "Impr.", "Cost", "Budget"}
var googleRows = [][]string{
	{"1234", "Nykaa Festive", "Enabled", "Search", "4,200,000", "1850000", "3000000"},
	{"5678", "Lenskart Video", "Paused", "Video", "3,850,000", "420000", "1500000"},
}

// This product's own export already uses the field names, so the
// common case must not pay for a model call at all.
func TestAnExportUsingTheFieldNamesNeedsNoModel(t *testing.T) {
	m := &mapperModel{}
	headers := []string{"external_id", "name", "category", "region", "platform", "status", "ad_type", "reach_lakh", "spend", "budget", "days_running", "frequency"}
	insp := mapperSvc(m).ProposeMapping(context.Background(), headers, [][]string{})

	if m.calls != 0 {
		t.Errorf("called the model %d times for an exact match", m.calls)
	}
	if insp.Source != ingest.MappingSourceExact || !insp.Confident {
		t.Errorf("got %+v", insp)
	}
}

func TestAnUnfamiliarExportIsMappedByTheModel(t *testing.T) {
	m := &mapperModel{reply: `{"external_id":"Campaign ID","name":"Campaign","status":"Campaign state","ad_type":"Campaign type","reach":"Impr.","spend":"Cost","budget":"Budget"}`}
	insp := mapperSvc(m).ProposeMapping(context.Background(), googleHeaders, googleRows)

	if insp.Source != ingest.MappingSourceModel {
		t.Fatalf("source %q", insp.Source)
	}
	if insp.Mapping["external_id"] != "Campaign ID" || insp.Mapping["spend"] != "Cost" {
		t.Errorf("mapping wrong: %+v", insp.Mapping)
	}
	if !insp.Confident {
		t.Error("both required fields are mapped, so this should be confident")
	}
	// The preview is what lets a human check the guess before importing.
	if len(insp.Preview) == 0 || insp.Preview[0]["spend"] != "1850000" {
		t.Errorf("preview did not read through the mapping: %+v", insp.Preview)
	}
}

// The failure that matters. A column that is not in the file would map
// a field to nothing and read every row as blank, which looks like
// missing data rather than a bad mapping.
func TestAColumnThatDoesNotExistIsDiscarded(t *testing.T) {
	m := &mapperModel{reply: `{"external_id":"Campaign ID","spend":"Total Cost (INR)","name":"Campaign"}`}
	insp := mapperSvc(m).ProposeMapping(context.Background(), googleHeaders, googleRows)

	if _, ok := insp.Mapping["spend"]; ok {
		t.Errorf("kept a mapping to a column the file does not have: %+v", insp.Mapping)
	}
	if insp.Mapping["external_id"] != "Campaign ID" {
		t.Error("the valid part of the reply should survive")
	}
}

func TestAnInventedFieldNameIsDiscarded(t *testing.T) {
	m := &mapperModel{reply: `{"external_id":"Campaign ID","name":"Campaign","roi_score":"Cost"}`}
	insp := mapperSvc(m).ProposeMapping(context.Background(), googleHeaders, googleRows)
	if _, ok := insp.Mapping["roi_score"]; ok {
		t.Error("kept a field that is not in the schema")
	}
}

// Models fence JSON even when told not to.
func TestAFencedReplyIsStillRead(t *testing.T) {
	m := &mapperModel{reply: "```json\n{\"external_id\":\"Campaign ID\",\"name\":\"Campaign\"}\n```"}
	insp := mapperSvc(m).ProposeMapping(context.Background(), googleHeaders, googleRows)
	if insp.Mapping["external_id"] != "Campaign ID" {
		t.Errorf("a fenced reply was not parsed: %+v", insp.Mapping)
	}
}

// A reply that is not JSON at all leaves the exact matches standing
// rather than losing the import.
func TestAnUnparseableReplyFallsBackToExactMatches(t *testing.T) {
	m := &mapperModel{reply: "I think the first column is the id."}
	headers := append([]string{"external_id"}, googleHeaders...)
	insp := mapperSvc(m).ProposeMapping(context.Background(), headers, googleRows)

	if insp.Source != ingest.MappingSourceExact {
		t.Errorf("source %q, want the exact fallback", insp.Source)
	}
	if insp.Mapping["external_id"] != "external_id" {
		t.Errorf("the exact match was lost: %+v", insp.Mapping)
	}
}

// Two fields pointing at one column means one of them is wrong, and
// there is no way to tell which.
func TestOneColumnCannotFeedTwoFields(t *testing.T) {
	m := &mapperModel{reply: `{"external_id":"Campaign ID","name":"Campaign","spend":"Cost","budget":"Cost"}`}
	insp := mapperSvc(m).ProposeMapping(context.Background(), googleHeaders, googleRows)

	used := map[string]int{}
	for _, col := range insp.Mapping {
		used[col]++
	}
	for col, n := range used {
		if n > 1 {
			t.Errorf("column %q feeds %d fields", col, n)
		}
	}
}

// Without external_id a row cannot be matched on a later sync and
// would insert a duplicate every time.
func TestAMappingWithNoIdIsRefused(t *testing.T) {
	_, err := ingest.RowsToCampaigns(googleHeaders, googleRows, ingest.ColumnMapping{"name": "Campaign"}, ingest.ReachRaw)
	if err == nil || !strings.Contains(err.Error(), "external_id") {
		t.Fatalf("got %v", err)
	}
}

// A confirmed mapping reads the rows, normalising the platform's own
// vocabulary on the way through.
func TestAConfirmedMappingReadsTheRows(t *testing.T) {
	rows, err := ingest.RowsToCampaigns(googleHeaders, googleRows, ingest.ColumnMapping{
		"external_id": "Campaign ID", "name": "Campaign", "status": "Campaign state",
		"ad_type": "Campaign type", "reach": "Impr.", "spend": "Cost", "budget": "Budget",
	}, ingest.ReachRaw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows", len(rows))
	}
	if rows[0].Status != "live" || rows[0].AdType != "Google Ads" {
		t.Errorf("vocabulary not normalised: %+v", rows[0])
	}
	if rows[0].Spend != 1850000 {
		t.Errorf("spend %d", rows[0].Spend)
	}
	if rows[1].Status != "paused" {
		t.Errorf("status %q", rows[1].Status)
	}
	// 4,200,000 raw impressions is 42 lakh. Storing the raw figure
	// overflows the column, which is how the ambiguity was found.
	if rows[0].ReachLakh != 42 {
		t.Errorf("reach %.2f lakh, want 42 — the raw figure was not converted", rows[0].ReachLakh)
	}
}

// The unit is confirmed, not inferred. Reading a raw count as lakhs
// overstates reach by a hundred thousand.
func TestTheReachUnitIsApplied(t *testing.T) {
	m := ingest.ColumnMapping{"external_id": "Campaign ID", "reach": "Impr."}
	raw, err := ingest.RowsToCampaigns(googleHeaders, googleRows, m, ingest.ReachRaw)
	if err != nil {
		t.Fatal(err)
	}
	lakh, err := ingest.RowsToCampaigns(googleHeaders, googleRows, m, ingest.ReachLakh)
	if err != nil {
		t.Fatal(err)
	}
	if raw[0].ReachLakh != 42 {
		t.Errorf("raw: got %.2f, want 42", raw[0].ReachLakh)
	}
	if lakh[0].ReachLakh != 4_200_000 {
		t.Errorf("lakh: got %.2f, want the figure taken as given", lakh[0].ReachLakh)
	}
}
