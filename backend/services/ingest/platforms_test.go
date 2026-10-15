package ingest

import (
	"math"
	"strings"
	"testing"
	"time"
)

// These test the only part of a platform client that can be proven
// without a credential: the mapping from the documented response shape
// onto this product's model. The transport around them is unverified
// until `make verify-ingest` has been run against a real account.

// Recorded from Google's searchStream reference: an array of chunks,
// each with results, and a campaign repeated once per date segment.
const googleBody = `[
 {"results":[
  {"campaign":{"id":"1234","name":"Nykaa Festive Search","status":"ENABLED","advertisingChannelType":"SEARCH"},
   "campaignBudget":{"amountMicros":"3000000000"},
   "metrics":{"impressions":"4200000","costMicros":"900000000","averageCpm":"214000"}},
  {"campaign":{"id":"1234","name":"Nykaa Festive Search","status":"ENABLED","advertisingChannelType":"SEARCH"},
   "campaignBudget":{"amountMicros":"3000000000"},
   "metrics":{"impressions":"2000000","costMicros":"950000000","averageCpm":"210000"}}
 ]},
 {"results":[
  {"campaign":{"id":"5678","name":"Lenskart Discovery","status":"PAUSED","advertisingChannelType":"VIDEO"},
   "campaignBudget":{"amountMicros":"1500000000"},
   "metrics":{"impressions":"3850000","costMicros":"420000000","averageCpm":"109000"}}
 ]}
]`

// A campaign appears once per date segment. Without folding, a
// thirty-day window reports thirty copies of each campaign, each
// carrying one day's spend — which would then be ingested as thirty
// separate campaigns on first sync and look like a working import.
func TestGoogleFoldsTheDateSegmentsOfOneCampaign(t *testing.T) {
	got, err := parseGoogleAds([]byte(googleBody))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d campaigns, want 2 — segments were not folded", len(got))
	}
	first := got[0]
	if first.ExternalID != "1234" {
		t.Fatalf("unexpected order: %+v", got)
	}
	// 900000000 + 950000000 micros = 1850 rupees.
	if first.Spend != 1850 {
		t.Errorf("spend %d, want 1850 — the two segments should sum", first.Spend)
	}
	// 4.2M + 2.0M impressions = 6.2M = 62 lakh.
	if math.Abs(first.ReachLakh-62) > 0.001 {
		t.Errorf("reach %.3f lakh, want 62", first.ReachLakh)
	}
	// Budget is the same on every segment row, not summed.
	if first.Budget != 3000 {
		t.Errorf("budget %d, want 3000 — it should not accumulate across segments", first.Budget)
	}
}

// Micros are Google's unit for money. Reading them as rupees would
// overstate every figure by a million.
func TestGoogleConvertsMicrosToRupees(t *testing.T) {
	got, _ := parseGoogleAds([]byte(googleBody))
	if got[1].Spend != 420 {
		t.Errorf("spend %d, want 420 (420000000 micros)", got[1].Spend)
	}
}

func TestGoogleMapsItsOwnVocabulary(t *testing.T) {
	got, _ := parseGoogleAds([]byte(googleBody))
	if got[0].Status != "live" {
		t.Errorf("ENABLED became %q, want live", got[0].Status)
	}
	if got[0].AdType != "Google Ads" {
		t.Errorf("SEARCH became %q, want Google Ads", got[0].AdType)
	}
	if got[1].Status != "paused" {
		t.Errorf("PAUSED became %q, want paused", got[1].Status)
	}
	if got[1].AdType != "Video" {
		t.Errorf("VIDEO became %q, want Video", got[1].AdType)
	}
}

// The API reports failures in the body with a 200 on some paths, so a
// successful-looking response can still be an error.
func TestGoogleSurfacesAnErrorInTheBody(t *testing.T) {
	_, err := parseGoogleAds([]byte(`{"error":{"status":"PERMISSION_DENIED","message":"The caller does not have permission"}}`))
	if err == nil || !strings.Contains(err.Error(), "PERMISSION_DENIED") {
		t.Fatalf("got %v, want the API's own error", err)
	}
}

// Recorded from Meta's campaigns edge: insights nested as their own
// single-element list, money as strings in minor units.
const metaBody = `{
 "data":[
  {"id":"23850","name":"Mamaearth Always-On","status":"ACTIVE","objective":"OUTCOME_AWARENESS",
   "lifetime_budget":"250000000",
   "insights":{"data":[{"spend":"1842500","impressions":"9100000","reach":"5400000"}]}},
  {"id":"23851","name":"Nykaa Retargeting","status":"PAUSED","objective":"OUTCOME_SALES",
   "daily_budget":"500000",
   "insights":{"data":[]}}
 ],
 "paging":{"next":"https://graph.facebook.com/v21.0/act_1/campaigns?after=CURSOR"}
}`

func TestMetaMapsCampaignsAndReturnsItsCursor(t *testing.T) {
	got, next, err := parseMeta([]byte(metaBody))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d campaigns, want 2", len(got))
	}
	if next == "" || !strings.Contains(next, "after=CURSOR") {
		t.Errorf("the paging cursor was dropped: %q", next)
	}
	if got[0].Status != "live" || got[1].Status != "paused" {
		t.Errorf("status mapping wrong: %q, %q", got[0].Status, got[1].Status)
	}
}

// Meta reports reach directly — unique people, which is the measure
// this product stores — so it must be preferred over impressions.
func TestMetaPrefersReachOverImpressions(t *testing.T) {
	got, _, _ := parseMeta([]byte(metaBody))
	if math.Abs(got[0].ReachLakh-54) > 0.001 {
		t.Errorf("reach %.3f lakh, want 54 (5.4M reach, not 9.1M impressions)", got[0].ReachLakh)
	}
}

// Budgets arrive in paise as strings, and a lifetime cap bounds the
// campaign where both are present.
func TestMetaConvertsMinorUnitsAndPrefersTheLifetimeCap(t *testing.T) {
	got, _, _ := parseMeta([]byte(metaBody))
	if got[0].Budget != 2_500_000 {
		t.Errorf("budget %d, want 2500000 (250000000 paise)", got[0].Budget)
	}
	if got[1].Budget != 5_000 {
		t.Errorf("daily budget %d, want 5000", got[1].Budget)
	}
}

// A campaign with no insights yet is still a campaign; it must come
// through with zeroes rather than being dropped.
func TestMetaKeepsACampaignWithNoInsightsYet(t *testing.T) {
	got, _, _ := parseMeta([]byte(metaBody))
	if got[1].ExternalID != "23851" {
		t.Fatal("a campaign with an empty insights list was dropped")
	}
	if got[1].Spend != 0 || got[1].ReachLakh != 0 {
		t.Errorf("expected zeroes, got spend %d reach %.2f", got[1].Spend, got[1].ReachLakh)
	}
}

func TestMetaSurfacesItsOwnError(t *testing.T) {
	_, _, err := parseMeta([]byte(`{"error":{"message":"Invalid OAuth access token","type":"OAuthException","code":190}}`))
	if err == nil || !strings.Contains(err.Error(), "OAuthException") {
		t.Fatalf("got %v, want the API's own error", err)
	}
}

// Both providers refuse distinctly when nobody has set them up.
func TestUnconfiguredProvidersReportThatSpecifically(t *testing.T) {
	for _, p := range []Provider{GoogleAdsProvider{}, MetaProvider{}} {
		if _, err := p.Fetch(t.Context(), time.Time{}); err != ErrNotConfigured {
			t.Errorf("%s: got %v, want ErrNotConfigured", p.Source(), err)
		}
	}
}
