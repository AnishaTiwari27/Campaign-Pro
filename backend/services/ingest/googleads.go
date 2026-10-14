package ingest

import (
	"context"
	"errors"
	"time"
)

// ErrNotConfigured is returned by a provider whose credentials are
// absent. Distinct from a transport failure on purpose: "you have not
// set this up" and "the platform is down" need different responses, and
// a sync that reports the second when it means the first sends people
// to look at the wrong thing.
var ErrNotConfigured = errors.New("provider is not configured")

// GoogleAdsConfig is what the Google Ads API needs before it will answer.
//
// All five are required and none can be defaulted: the developer token
// is issued per manager account after an application, and the OAuth
// client and refresh token belong to a specific Google Cloud project.
// There is no useful behaviour without them, which is why this provider
// refuses rather than guessing.
type GoogleAdsConfig struct {
	DeveloperToken string
	ClientID       string
	ClientSecret   string
	RefreshToken   string
	CustomerID     string
}

func (c GoogleAdsConfig) complete() bool {
	return c.DeveloperToken != "" && c.ClientID != "" && c.ClientSecret != "" &&
		c.RefreshToken != "" && c.CustomerID != ""
}

// GoogleAdsProvider pulls campaigns from a Google Ads account.
//
// The transport is not implemented here, and that is a deliberate
// boundary rather than an oversight. Reaching the API needs a developer
// token that Google issues per manager account on application, plus an
// OAuth client and a refresh token for a real account — none of which
// can be stood up from inside this repository, and none of which can be
// meaningfully faked. Writing a client that has never once been run
// against the real thing would be the worst outcome: it would look
// finished, and the first person to configure credentials would find out
// it was not.
//
// What is real is everything around it. The Provider interface, the
// mapping, the idempotent upsert, the run history and the failure
// counting are all exercised by the CSV provider, which is the same path
// end to end. Implementing Fetch against googleads.GoogleAdsClient is
// then the whole remaining job, and it has somewhere correct to put its
// rows.
type GoogleAdsProvider struct {
	Config GoogleAdsConfig
}

func (p GoogleAdsProvider) Source() string { return "google-ads" }

func (p GoogleAdsProvider) Fetch(_ context.Context, _ time.Time) ([]ExternalCampaign, error) {
	if !p.Config.complete() {
		return nil, ErrNotConfigured
	}
	// Credentials present but no transport: say exactly that, so the
	// person who just configured it is not left wondering whether their
	// token is wrong.
	return nil, errors.New("google ads transport is not implemented — credentials are configured but no client is wired up yet")
}

// Configured reports whether this provider could run at all, so the API
// can list it as available or not without attempting a sync.
func (p GoogleAdsProvider) Configured() bool { return p.Config.complete() }
