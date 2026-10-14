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

type orgKey struct{}

// Org is the tenant boundary: a hard wall that applies to every account,
// admins included. It is a different thing from Campaigns above, which
// narrows what one client account sees *inside* an organisation.
//
// System is the escape hatch for callers that are not requests — the
// report worker, the seeder, the CLI. They legitimately operate across
// organisations, and the alternative (inventing a request context for
// the worker) would mean the worker could only ever serve one tenant.
//
// The zero value is neither: no id and not system, which every query
// below reads as "matches nothing". That is deliberate. A request whose
// organisation could not be resolved must return an empty screen, not
// somebody else's book of business — the failure mode of forgetting to
// set this has to be a blank page, never a leak.
type Org struct {
	ID     string
	System bool
}

// WithOrg attaches the tenant to the context.
func WithOrg(ctx context.Context, o Org) context.Context {
	return context.WithValue(ctx, orgKey{}, o)
}

// DefaultOrg is the organisation every existing row was backfilled into
// by the tenancy migration, and the one a CLI or seeder writes to unless
// told otherwise.
const DefaultOrg = "default"

// SystemOrg marks a context as running outside any request, so its reads
// are not confined to one tenant. Used by the report worker, which runs
// every organisation's due reports.
//
// It names no organisation, which is right for a pure reader and wrong
// for anything that inserts: a write has to say which tenant owns the
// row, and an insert under this context fails the foreign key rather
// than landing somewhere arbitrary. Writers use SystemOrgIn.
func SystemOrg(ctx context.Context) context.Context {
	return WithOrg(ctx, Org{System: true})
}

// SystemOrgIn is SystemOrg for a caller that also writes: reads span
// organisations, and anything inserted belongs to the one named.
func SystemOrgIn(ctx context.Context, orgID string) context.Context {
	return WithOrg(ctx, Org{ID: orgID, System: true})
}

// OrgFrom reads the tenant back. An absent org is the zero value, which
// matches nothing — see the note on Org.
func OrgFrom(ctx context.Context) Org {
	o, _ := ctx.Value(orgKey{}).(Org)
	return o
}
