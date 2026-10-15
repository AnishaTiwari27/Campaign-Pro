// Package ingest pulls campaign data from the ad platforms that actually
// ran the work.
//
// Everything else in this product was seeded, and seeded data gets to
// choose its own ids. A real platform chooses its own, and that id is
// the only stable handle on "the same campaign as last time" — names are
// edited and budgets change, so matching on either would create a
// duplicate the first time somebody renamed something in the ad account.
//
// A Provider is deliberately a narrow interface over "give me rows since
// this time". That keeps the transport — OAuth, pagination, a vendor
// query language — out of the part that has to be correct, which is the
// matching and the upsert. The CSV provider is not a toy for tests: it
// is how an agency without API access gets its data in, and it exercises
// exactly the same path as every other source.
package ingest

import (
	"context"
	"time"
)

// ExternalCampaign is one row as a platform reports it, already mapped
// out of that platform's vocabulary.
//
// Deliberately not the campaigns domain type. A platform knows nothing
// about approval state, flags or pacing, and a struct that can carry
// them is a struct somebody will eventually let a sync overwrite them
// with.
type ExternalCampaign struct {
	// ExternalID is the platform's own identifier. The match key.
	ExternalID string
	Name       string
	Category   string
	Region     string
	Platform   string
	// Status as this platform reports it, already normalised to the
	// product's vocabulary by the provider.
	Status      string
	AdType      string
	DaysRunning int32
	// ReachLakh keeps the product's unit. Providers convert; nothing
	// downstream should have to ask which unit a number is in.
	ReachLakh float64
	Spend     int64
	Budget    int64
	Frequency float64
}

// Provider is one ad platform.
type Provider interface {
	// Source names the platform, and is stored on every row it writes.
	// It is part of the match key, so changing it orphans that
	// platform's history.
	Source() string
	// Fetch returns rows changed since the given time. A provider that
	// cannot filter by time may return everything; the upsert is
	// idempotent, so the cost is bandwidth rather than correctness.
	Fetch(ctx context.Context, since time.Time) ([]ExternalCampaign, error)
}

// Result is what one sync did. Created and updated are counted
// separately because "47 seen, 47 created" on a second run means the
// matching is broken, and a single total would hide that.
type Result struct {
	Source  string
	Seen    int
	Created int
	Updated int
	Failed  int
	Err     error
}

// FixedProvider carries rows that have already been read and mapped —
// the output of a confirmed column mapping. It exists so a reviewed
// import runs the identical matching, upsert and run-recording path as
// every other source, rather than a second one written beside it.
type FixedProvider struct {
	SourceName string
	Rows       []ExternalCampaign
}

func (p FixedProvider) Source() string { return p.SourceName }

func (p FixedProvider) Fetch(context.Context, time.Time) ([]ExternalCampaign, error) {
	return p.Rows, nil
}
