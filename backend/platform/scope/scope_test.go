package scope_test

import (
	"context"
	"testing"

	"campaigntrackerpro/platform/scope"
)

// The whole safety argument for the tenant boundary rests on this: a
// context that nobody set an organisation on must match nothing, not
// everything. If this ever inverts, every unscoped code path silently
// starts returning other tenants' data.
func TestAnUnsetOrgMatchesNothing(t *testing.T) {
	o := scope.OrgFrom(context.Background())
	if o.System {
		t.Error("an unset org must not be a system context")
	}
	if o.ID != "" {
		t.Errorf("an unset org must have no id, got %q", o.ID)
	}
}

func TestSystemOrgIsNotConfinedToATenant(t *testing.T) {
	o := scope.OrgFrom(scope.SystemOrg(context.Background()))
	if !o.System {
		t.Error("SystemOrg must mark the context as system")
	}
}

func TestOrgRoundTrips(t *testing.T) {
	ctx := scope.WithOrg(context.Background(), scope.Org{ID: "acme"})
	o := scope.OrgFrom(ctx)
	if o.ID != "acme" || o.System {
		t.Errorf("got %+v, want {acme false}", o)
	}
}

// Campaigns scope and Org scope are independent: narrowing a client to
// two campaigns must not disturb which tenant they are in, and vice
// versa. They are stored under different keys for exactly this reason.
func TestOrgAndCampaignScopeDoNotOverwriteEachOther(t *testing.T) {
	ctx := scope.WithOrg(context.Background(), scope.Org{ID: "acme"})
	ctx = scope.WithCampaigns(ctx, scope.Campaigns{Restricted: true, IDs: []string{"a", "b"}})

	if got := scope.OrgFrom(ctx).ID; got != "acme" {
		t.Errorf("org lost when campaign scope was set: %q", got)
	}
	cs := scope.CampaignsFrom(ctx)
	if !cs.Restricted || len(cs.IDs) != 2 {
		t.Errorf("campaign scope lost: %+v", cs)
	}
}
