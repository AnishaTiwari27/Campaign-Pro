package ingest

import (
	"context"
	"strings"
	"testing"
	"time"
)

const header = "external_id,name,category,region,platform,status,ad_type,reach_lakh,spend,budget,days_running,frequency\n"

func fetch(t *testing.T, csv string) []ExternalCampaign {
	t.Helper()
	rows, err := CSVProvider{Reader: strings.NewReader(csv)}.Fetch(context.Background(), time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return rows
}

func TestCSVMapsARowOntoTheProductsVocabulary(t *testing.T) {
	rows := fetch(t, header+"g-123,Blinkit Diwali,E-commerce,Delhi NCR,YouTube,ENABLED,Social,42.5,250000,500000,12,1.8\n")
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	r := rows[0]
	if r.ExternalID != "g-123" || r.Name != "Blinkit Diwali" {
		t.Errorf("identity wrong: %+v", r)
	}
	if r.Status != "live" {
		t.Errorf("status %q — ENABLED should normalise to live", r.Status)
	}
	if r.ReachLakh != 42.5 || r.Spend != 250000 || r.Budget != 500000 {
		t.Errorf("figures wrong: %+v", r)
	}
}

// Platforms each have their own word for the same state. Anything
// unrecognised becomes paused rather than live: a campaign wrongly shown
// as paused gets noticed and corrected, one wrongly shown as live
// quietly accrues spend nobody is watching.
func TestUnknownStatusesFailSafeToPaused(t *testing.T) {
	cases := map[string]string{
		"ENABLED": "live", "serving": "live", "Active": "live",
		"REMOVED": "ended", "completed": "ended",
		"DRAFT": "scheduled", "upcoming": "scheduled",
		"":                "paused",
		"SOMETHING_ELSE":  "paused",
		"under_review_42": "paused",
	}
	for in, want := range cases {
		if got := normaliseStatus(in); got != want {
			t.Errorf("normaliseStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

// A spreadsheet that has been through a human has had its columns
// reordered, so the reader matches on name.
func TestColumnOrderDoesNotMatter(t *testing.T) {
	reordered := "spend,name,external_id,budget,region,category,platform,status,ad_type,reach_lakh\n" +
		"9999,Reordered,g-9,5000,Mumbai,Beauty,Instagram,live,Influencer,3.5\n"
	rows := fetch(t, reordered)
	if len(rows) != 1 || rows[0].ExternalID != "g-9" || rows[0].Spend != 9999 {
		t.Fatalf("reordered columns were not matched by name: %+v", rows)
	}
}

// A missing column has to name itself. The alternative is an import that
// silently yields zero rows and a user with no way to tell why.
func TestMissingColumnsAreNamed(t *testing.T) {
	_, err := CSVProvider{Reader: strings.NewReader("external_id,name\ng-1,X\n")}.Fetch(context.Background(), time.Time{})
	if err == nil {
		t.Fatal("expected an error for a file missing most columns")
	}
	for _, want := range []string{"category", "region", "spend"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not name the missing column %q: %v", want, err)
		}
	}
}

// Without an external id there is nothing to match the row on, so it
// would create a duplicate on every sync. Refused outright.
func TestARowWithNoExternalIDIsRefused(t *testing.T) {
	_, err := CSVProvider{Reader: strings.NewReader(header + ",No Id,Beauty,Mumbai,Instagram,live,Social,1,2,3,4,5\n")}.
		Fetch(context.Background(), time.Time{})
	if err == nil || !strings.Contains(err.Error(), "external_id") {
		t.Fatalf("expected a refusal naming external_id, got %v", err)
	}
}

// A malformed number is visibly wrong in the UI where somebody can fix
// it. Rejecting the whole import instead would leave them with nothing.
func TestAnUnparseableNumberBecomesZeroRatherThanFailingTheImport(t *testing.T) {
	rows := fetch(t, header+"g-5,Odd,Beauty,Mumbai,Instagram,live,Social,not-a-number,1'2'3,500,x,y\n")
	if len(rows) != 1 {
		t.Fatalf("row was dropped: %+v", rows)
	}
	if rows[0].ReachLakh != 0 || rows[0].Spend != 0 {
		t.Errorf("expected zeros for unparseable cells, got %+v", rows[0])
	}
	if rows[0].Budget != 500 {
		t.Errorf("a good column was lost alongside the bad ones: %+v", rows[0])
	}
}

// Thousands separators survive a trip through a spreadsheet.
func TestCommasInNumbersAreTolerated(t *testing.T) {
	rows := fetch(t, header+"g-6,Commas,FMCG,Mumbai,TV,live,Video,12.5,\"1,250,000\",\"2,000,000\",30,2\n")
	if rows[0].Spend != 1250000 || rows[0].Budget != 2000000 {
		t.Errorf("thousands separators not handled: %+v", rows[0])
	}
}

// An unconfigured provider says so specifically, because "you have not
// set this up" and "the platform is down" need different responses.
func TestGoogleAdsReportsMissingCredentialsDistinctly(t *testing.T) {
	p := GoogleAdsProvider{}
	if p.Configured() {
		t.Error("an empty config should not report as configured")
	}
	_, err := p.Fetch(context.Background(), time.Time{})
	if err != ErrNotConfigured {
		t.Errorf("got %v, want ErrNotConfigured", err)
	}
}

// Ad type needs the same normalisation as status, and did not have it at
// first: a Google Ads export says "Search", which is not one of the six
// values the column stores, so the row was rejected by the enum and the
// sync quietly reported a failure instead of ingesting it.
func TestAdTypesAreMappedOntoTheStoredVocabulary(t *testing.T) {
	cases := map[string]string{
		"Search": "Google Ads", "SHOPPING": "Google Ads", "sem": "Google Ads",
		"Display": "Display", "banner-ish": "Display", "": "Display",
		"Video": "Video", "YouTube": "Video",
		"Performance Max": "Performance", "pmax": "Performance",
		"Paid Social": "Social", "instagram": "Social",
		"Creator": "Influencer",
	}
	for in, want := range cases {
		if got := normaliseAdType(in); got != want {
			t.Errorf("normaliseAdType(%q) = %q, want %q", in, got, want)
		}
	}
}
