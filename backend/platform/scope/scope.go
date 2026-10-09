// Package scope carries row-level visibility on the request context.
//
// It sits in platform because both ends need it and neither owns it: the
// HTTP layer decides what the caller may see, and the store is the one
// place narrow enough to apply that to every read. Threading a caller
// through All(), List(), Flagged() and four services' worth of call sites
// would be more explicit, and would also mean that forgetting one of them
// silently widens what a client can read — exactly the failure this is
// meant to prevent.
package scope

import "context"

type key struct{}

// Campaigns is what the caller may see. Restricted is the switch: an
// unrestricted scope sees everything, which is right for agency staff and
// for anything running outside a request (the worker, the seeder, the CLI).
//
// The pairing matters. Restricted with an empty IDs list means "sees
// nothing", not "sees everything" — a client whose grants have not been
// set yet must get an empty screen rather than the whole book of business.
type Campaigns struct {
	Restricted bool
	IDs        []string
}

// WithCampaigns attaches the caller's visibility to the context.
func WithCampaigns(ctx context.Context, c Campaigns) context.Context {
	return context.WithValue(ctx, key{}, c)
}

// CampaignsFrom reads it back. An absent scope is unrestricted, because
// most callers are not requests at all.
func CampaignsFrom(ctx context.Context) Campaigns {
	c, _ := ctx.Value(key{}).(Campaigns)
	return c
}

// Allows reports whether a campaign id is visible under this scope.
func (c Campaigns) Allows(id string) bool {
	if !c.Restricted {
		return true
	}
	for _, allowed := range c.IDs {
		if allowed == id {
			return true
		}
	}
	return false
}
