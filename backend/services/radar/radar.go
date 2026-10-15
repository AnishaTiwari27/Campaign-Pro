// Package radar watches for products and brands that have just
// launched, and estimates which of them look close to advertising.
//
// A word on what the estimate is, because the honest answer is not the
// flattering one. A forecast worth the name is fitted to outcomes: you
// observe many launches, record how long each took to start
// advertising, and learn the relationship. This system has no such
// history — it has never watched a launch through to its first
// campaign — so nothing here is fitted to anything.
//
// What it does instead is band launches by signals anyone can check:
// how recent, how much traction the source reports, whether there is a
// site to send traffic to. The rules are written out below rather than
// learned, every band states the signals it came from, and the word
// used throughout is "estimate" rather than "forecast".
//
// The gap is closed by collecting the missing data from today. Every
// estimate is stored, and when a campaign for that brand appears the
// real lag is recorded against it. Once enough of those accumulate the
// bands can be checked, and eventually replaced by something fitted.
// Until then this is a prioritised list with its reasoning shown, and
// calling it a prediction would be overselling it.
package radar

import "time"

// Readiness is how close a launch looks to advertising.
type Readiness string

const (
	// ReadinessSoon — launched recently, has traction and somewhere to
	// send traffic. The ones worth a conversation this week.
	ReadinessSoon Readiness = "soon"
	// ReadinessBuilding — launched, but missing one of the signals
	// that usually accompanies a push.
	ReadinessBuilding Readiness = "building"
	// ReadinessQuiet — launched a while ago with little traction.
	ReadinessQuiet Readiness = "quiet"
	// ReadinessUnclear — not enough signal to say anything, which is a
	// real answer and better than inventing a band.
	ReadinessUnclear Readiness = "unclear"
)

// Launch is one product or brand the radar has seen.
type Launch struct {
	ID         string    `json:"id"`
	Source     string    `json:"source"`
	ExternalID string    `json:"externalId"`
	Name       string    `json:"name"`
	Tagline    string    `json:"tagline"`
	URL        string    `json:"url"`
	Category   string    `json:"category"`
	LaunchedAt time.Time `json:"launchedAt"`
	FirstSeen  time.Time `json:"firstSeenAt"`

	// Observable signals the band is computed from.
	Traction int32 `json:"traction"`
	HasSite  bool  `json:"hasSite"`

	Readiness Readiness `json:"readiness"`
	// Basis names the signals behind the band, so the estimate can be
	// argued with rather than just believed.
	Basis string `json:"basis"`

	// Outcome, once a campaign for this brand turns up.
	FirstCampaignAt *time.Time `json:"firstCampaignAt,omitempty"`
	DaysToMarket    *int32     `json:"daysToMarket,omitempty"`
}

// Source is somewhere launches are announced.
type Source interface {
	Name() string
	// Fetch returns launches announced since the given time.
	Fetch(since time.Time) ([]Launch, error)
}
