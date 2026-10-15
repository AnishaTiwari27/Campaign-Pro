package ingest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Google sunset developer tokens on 9 September 2026. Access now
// attaches to the Google Cloud project behind the OAuth credentials,
// and a token cannot be applied for any more — so requiring one would
// mean nobody setting this up today could ever configure the provider.
func TestGoogleAdsConfiguredWithoutDeveloperToken(t *testing.T) {
	p := GoogleAdsProvider{Config: GoogleAdsConfig{
		ClientID: "cid", ClientSecret: "secret", RefreshToken: "refresh", CustomerID: "123",
	}}
	if !p.Configured() {
		t.Fatal("a config with OAuth credentials and a customer id reads as unconfigured — " +
			"the sunset developer token is still being required")
	}

	// The credentials that do still matter are still required.
	for _, missing := range []GoogleAdsConfig{
		{ClientSecret: "s", RefreshToken: "r", CustomerID: "c"},
		{ClientID: "i", RefreshToken: "r", CustomerID: "c"},
		{ClientID: "i", ClientSecret: "s", CustomerID: "c"},
		{ClientID: "i", ClientSecret: "s", RefreshToken: "r"},
	} {
		if (GoogleAdsProvider{Config: missing}).Configured() {
			t.Errorf("%+v reads as configured", missing)
		}
	}
}

// An empty header is a claim to carry a credential that is not there.
func TestGoogleAdsOmitsTheDeveloperTokenHeaderWhenUnset(t *testing.T) {
	var present bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Write([]byte(`{"access_token":"minted","expires_in":3600}`))
			return
		}
		_, present = r.Header["Developer-Token"]
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	p := GoogleAdsProvider{Config: GoogleAdsConfig{
		ClientID: "cid", ClientSecret: "secret", RefreshToken: "refresh",
		CustomerID: "1234567890", BaseURL: srv.URL, HTTP: srv.Client(),
	}}
	p.tokens = testTokenSource(srv)

	if _, err := p.Fetch(t.Context(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if present {
		t.Error("developer-token header was sent with no token configured")
	}
}

// The pinned default has to be a version Google still serves. v18, the
// previous pin, is no longer listed as supported.
func TestGoogleAdsDefaultAPIVersionIsCurrent(t *testing.T) {
	got := GoogleAdsProvider{}.apiVersion()
	supported := map[string]bool{"v24": true, "v25": true, "v26": true}
	if !supported[got] {
		t.Errorf("default API version is %q, which Google does not list as supported "+
			"(v24, v25, v26 as of October 2026) — the first live call would fail on the path", got)
	}
}
