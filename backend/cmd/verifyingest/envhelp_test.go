package main

import (
	"strings"
	"testing"
)

// This string is what someone reads at the moment they are setting the
// provider up, so it is the most expensive place in the codebase for a
// stale instruction. It used to lead with GOOGLE_ADS_DEVELOPER_TOKEN
// "applied for, issued per manager account" — which, since Google
// sunset tokens on 9 September 2026, sent people off to apply for
// something that cannot be applied for.
func TestGoogleAdsEnvHelpDoesNotSendPeopleAfterADeveloperToken(t *testing.T) {
	help := envHelp("google-ads")

	// Every credential that is genuinely required must be named.
	for _, required := range []string{
		"GOOGLE_ADS_CLIENT_ID",
		"GOOGLE_ADS_CLIENT_SECRET",
		"GOOGLE_ADS_REFRESH_TOKEN",
		"GOOGLE_ADS_CUSTOMER_ID",
	} {
		if !strings.Contains(help, required) {
			t.Errorf("%s is required but not named in the setup help", required)
		}
	}

	// The token may be mentioned — it still works for anyone holding
	// one — but never as something to go and obtain.
	for _, chasing := range []string{"applied for", "apply for", "issued per manager account"} {
		if strings.Contains(strings.ToLower(help), chasing) {
			t.Errorf("setup help says %q, pointing at a developer token that can no longer be obtained", chasing)
		}
	}
	if !strings.Contains(help, "sunset") {
		t.Error("setup help mentions the developer token without saying it was sunset, " +
			"which reads as a credential still worth chasing")
	}
}

// Meta's token has no refresh exchange, so naming the kind of token
// matters: a short-lived user token looks like it works and then stops.
func TestMetaEnvHelpNamesTheTokenKind(t *testing.T) {
	help := envHelp("meta")
	if !strings.Contains(help, "META_ACCESS_TOKEN") || !strings.Contains(help, "ads_read") {
		t.Errorf("meta setup help does not name the token and its scope:\n%s", help)
	}
}
