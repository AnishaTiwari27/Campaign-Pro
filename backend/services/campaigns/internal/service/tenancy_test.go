package service

import (
	"context"
	"testing"

	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/campaigns"
	"campaigntrackerpro/services/campaigns/internal/store"

	"github.com/stretchr/testify/require"
)

// mkOrg creates a second organisation and removes it afterwards.
func mkOrg(t *testing.T, s *store.Store, id string) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, s.Exec(ctx, `INSERT INTO organizations (id, name) VALUES ($1, $1) ON CONFLICT DO NOTHING`, id))
	t.Cleanup(func() { _ = s.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, id) })
}

// The tenant boundary, tested the only way that means anything: put a
// campaign in another organisation and prove it is invisible through
// every read, not just the one that was most obviously scoped.
//
// An admin is used deliberately. Role is not the boundary — org is —
// so the most privileged account inside one tenant must still see
// nothing of another's.
func TestOneTenantCannotSeeAnother(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := NewCampaigns(s)

	mkOrg(t, s, "svctest-other-org")

	mine := "svctest-tenant-mine"
	theirs := "svctest-tenant-theirs"
	cleanup(t, s, mine, theirs)

	ours := scope.WithOrg(ctx, scope.Org{ID: scope.DefaultOrg})
	foreign := scope.WithOrg(ctx, scope.Org{ID: "svctest-other-org"})

	_, err := s.CreateCampaign(ours, mkTestCampaign(mine, 10, 100, 1000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)

	// Snapshot our side, then create the other tenant's campaign. Our
	// figures must be byte-identical afterwards: an absolute threshold
	// would only be testing how much seed data happens to be present.
	beforeTotals, err := c.Totals(ours)
	require.NoError(t, err)
	beforePending, err := c.PendingSpend(ours)
	require.NoError(t, err)

	_, err = s.CreateCampaign(foreign, mkTestCampaign(theirs, 99999, 777777, 999999, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)

	// A direct fetch by id must not cross the wall, even though the id is
	// known and valid — guessing an id is the easiest attack there is.
	_, err = c.GetDetail(ours, theirs, campaigns.ListParams{})
	require.Error(t, err, "fetched another organisation's campaign by id")

	// Nor may it appear in any listing.
	all, err := c.All(ours)
	require.NoError(t, err)
	for _, got := range all {
		require.NotEqual(t, theirs, got.ID, "another organisation's campaign appeared in All()")
	}

	// Nor may its money appear in an aggregate, which is the leak that is
	// easiest to miss because no row is ever returned.
	afterTotals, err := c.Totals(ours)
	require.NoError(t, err)
	require.Equal(t, beforeTotals, afterTotals, "another organisation's campaign moved our totals")

	afterPending, err := c.PendingSpend(ours)
	require.NoError(t, err)
	require.Equal(t, beforePending, afterPending, "another organisation's spend moved our pending figure")

	// And the far side sees its own, so the predicate is a wall and not
	// a blanket filter that happens to hide everything.
	theirAll, err := c.All(foreign)
	require.NoError(t, err)
	var found bool
	for _, got := range theirAll {
		if got.ID == theirs {
			found = true
		}
		require.NotEqual(t, mine, got.ID, "our campaign leaked into the other organisation")
	}
	require.True(t, found, "the other organisation cannot see its own campaign")
}

// A context nobody set an organisation on must behave like a stranger,
// not like an administrator. This is the regression that would turn
// every missed call site into a cross-tenant leak instead of an empty
// screen.
func TestAContextWithNoOrganisationSeesNothing(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := NewCampaigns(s)

	id := "svctest-tenant-unset"
	cleanup(t, s, id)
	_, err := s.CreateCampaign(scope.WithOrg(ctx, scope.Org{ID: scope.DefaultOrg}),
		mkTestCampaign(id, 10, 100, 1000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)

	all, err := c.All(ctx) // no org on the context at all
	require.NoError(t, err)
	require.Empty(t, all, "an unscoped context saw campaigns — the boundary fails open")

	totals, err := c.Totals(ctx)
	require.NoError(t, err)
	require.Zero(t, totals.SpendWindow)
}
