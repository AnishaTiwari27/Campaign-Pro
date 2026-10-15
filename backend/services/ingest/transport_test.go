package ingest

import (
	"campaigntrackerpro/platform/oauth"

	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// These exercise the request the client builds and the way it walks a
// paginated response — everything except whether the real API returns
// what its reference says it does, which only `make verify-ingest`
// against a live account can establish.

func TestGoogleSendsTheHeadersAndQueryTheAPIRequires(t *testing.T) {
	var gotAuth, gotDevToken, gotLoginCustomer, gotPath, gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/token") {
			json.NewEncoder(w).Encode(map[string]any{"access_token": "minted", "expires_in": 3600})
			return
		}
		gotAuth = r.Header.Get("Authorization")
		gotDevToken = r.Header.Get("developer-token")
		gotLoginCustomer = r.Header.Get("login-customer-id")
		gotPath = r.URL.Path
		var body gaqlRequest
		json.NewDecoder(r.Body).Decode(&body)
		gotQuery = body.Query
		w.Write([]byte(googleBody))
	}))
	defer srv.Close()

	p := GoogleAdsProvider{Config: GoogleAdsConfig{
		DeveloperToken: "dev-token", ClientID: "cid", ClientSecret: "secret",
		RefreshToken: "refresh", CustomerID: "123-456-7890", LoginCustomerID: "999-888-7777",
		BaseURL: srv.URL, APIVersion: "v18", HTTP: srv.Client(),
	}}
	p.tokens = testTokenSource(srv)

	rows, err := p.Fetch(t.Context(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows", len(rows))
	}
	if gotAuth != "Bearer minted" {
		t.Errorf("Authorization %q — the refreshed token was not sent", gotAuth)
	}
	if gotDevToken != "dev-token" {
		t.Errorf("developer-token %q", gotDevToken)
	}
	// Hyphens must be stripped: the API rejects a formatted customer id.
	if gotLoginCustomer != "9998887777" {
		t.Errorf("login-customer-id %q, want the hyphens stripped", gotLoginCustomer)
	}
	if !strings.Contains(gotPath, "/v18/customers/1234567890/googleAds:searchStream") {
		t.Errorf("path %q", gotPath)
	}
	if !strings.Contains(gotQuery, "segments.date >= '2026-01-01'") {
		t.Errorf("the since date did not reach the query: %q", gotQuery)
	}
}

func TestGoogleReportsAnHTTPFailureWithItsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/token") {
			json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": 3600})
			return
		}
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"message":"developer token not approved"}}`))
	}))
	defer srv.Close()

	p := GoogleAdsProvider{Config: GoogleAdsConfig{
		DeveloperToken: "d", ClientID: "c", ClientSecret: "s", RefreshToken: "r",
		CustomerID: "1", BaseURL: srv.URL, HTTP: srv.Client(),
	}}
	p.tokens = testTokenSource(srv)

	_, err := p.Fetch(t.Context(), time.Time{})
	if err == nil || !strings.Contains(err.Error(), "developer token not approved") {
		t.Fatalf("got %v — the platform's reason should survive", err)
	}
}

// Meta paginates with an absolute next URL. Following it is the
// difference between the first hundred campaigns and the account.
func TestMetaFollowsItsPagingCursor(t *testing.T) {
	var pages int
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		if pages == 1 {
			body := strings.Replace(metaBody,
				"https://graph.facebook.com/v21.0/act_1/campaigns?after=CURSOR",
				srv.URL+"/next-page", 1)
			w.Write([]byte(body))
			return
		}
		w.Write([]byte(`{"data":[{"id":"999","name":"Page Two","status":"ACTIVE","objective":"OUTCOME_SALES","insights":{"data":[]}}],"paging":{}}`))
	}))
	defer srv.Close()

	p := MetaProvider{Config: MetaConfig{
		AccessToken: "tok", AdAccountID: "1", BaseURL: srv.URL, HTTP: srv.Client(),
	}}
	rows, err := p.Fetch(t.Context(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if pages != 2 {
		t.Errorf("fetched %d pages, want 2 — the cursor was not followed", pages)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d campaigns across both pages, want 3", len(rows))
	}
	if rows[2].ExternalID != "999" {
		t.Errorf("the second page's campaign is missing: %+v", rows)
	}
}

// The account id gets its act_ prefix whether or not it was supplied.
func TestMetaPrefixesTheAccountID(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Write([]byte(`{"data":[],"paging":{}}`))
	}))
	defer srv.Close()

	for _, id := range []string{"12345", "act_12345"} {
		p := MetaProvider{Config: MetaConfig{AccessToken: "t", AdAccountID: id, BaseURL: srv.URL, HTTP: srv.Client()}}
		if _, err := p.Fetch(t.Context(), time.Time{}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(path, "/act_12345/campaigns") {
			t.Errorf("account %q produced path %q", id, path)
		}
	}
}

// A cursor that loops must not spin forever holding the sync open.
func TestMetaStopsAtThePageLimit(t *testing.T) {
	var pages int
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		json.NewEncoder(w).Encode(map[string]any{
			"data":   []any{map[string]any{"id": "1", "name": "Loop", "status": "ACTIVE"}},
			"paging": map[string]string{"next": srv.URL + "/again"},
		})
	}))
	defer srv.Close()

	p := MetaProvider{Config: MetaConfig{AccessToken: "t", AdAccountID: "1", BaseURL: srv.URL, HTTP: srv.Client()}}
	if _, err := p.Fetch(t.Context(), time.Time{}); err != nil {
		t.Fatal(err)
	}
	if pages != maxPages {
		t.Errorf("followed %d pages, want the %d limit to stop it", pages, maxPages)
	}
}

// testTokenSource points the OAuth exchange at the fake server too, so
// these cover the token refresh without reaching Google.
func testTokenSource(srv *httptest.Server) *oauth.RefreshTokenSource {
	return &oauth.RefreshTokenSource{
		TokenURL: srv.URL + "/token", ClientID: "c", ClientSecret: "s",
		RefreshToken: "r", HTTP: srv.Client(),
	}
}
