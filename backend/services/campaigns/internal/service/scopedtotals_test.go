package service

import (
	"context"
	"testing"

	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/campaigns"

	"github.com/stretchr/testify/require"
)

// The overview's headline figures are aggregates, and an aggregate cannot
// be filtered after the fact: by the time it returns there are no rows
// left to drop. Before this, PendingCount and PendingSpend ran without a
// scope predicate, so a client account's overview showed the whole
// agency's pending queue — three campaigns and ₹21.5L against two
// campaigns they could actually open.
//
// Every assertion here is about a number a client must not be able to
// see, which is why it tests the restricted case rather than the happy
// one.
func TestAggregatesAreScopedToWhatTheCallerMaySee(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := NewCampaigns(s)

	mine := "svctest-scope-mine"
	theirs := "svctest-scope-theirs"
	cleanup(t, s, mine, theirs)

	// One campaign the client holds, pending, and one they do not.
	_, err := s.CreateCampaign(ctx, mkTestCampaign(mine, 100, 500, 2000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)
	_, err = s.CreateCampaign(ctx, mkTestCampaign(theirs, 900, 7000, 9000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)

	visible := scope.WithCampaigns(ctx, scope.Campaigns{Restricted: true, IDs: []string{mine}})

	n, err := c.PendingCount(visible)
	require.NoError(t, err)
	require.Equal(t, int64(1), n, "pending count leaked a campaign the client cannot open")

	spend, err := c.PendingSpend(visible)
	require.NoError(t, err)
	require.Equal(t, int64(500), spend, "pending spend leaked the other campaign's money")

	totals, err := c.Totals(visible)
	require.NoError(t, err)
	require.Equal(t, int64(1), totals.LiveCount)
	require.InDelta(t, 100.0, totals.ReachLive, 0.001)
	require.Equal(t, int64(500), totals.SpendWindow)
}

// The fail-closed case, and the one most likely to regress: a client
// whose grants have not been set yet has an empty id list. That has to
// mean "sees nothing". Treating it as unrestricted — the easy mistake,
// since an empty filter usually means no filter — would hand a brand new
// client account the entire agency's totals on their first login.
func TestARestrictedCallerWithNoGrantsSeesNothing(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := NewCampaigns(s)

	id := "svctest-scope-nogrants"
	cleanup(t, s, id)
	_, err := s.CreateCampaign(ctx, mkTestCampaign(id, 500, 4000, 8000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)

	none := scope.WithCampaigns(ctx, scope.Campaigns{Restricted: true, IDs: nil})

	n, err := c.PendingCount(none)
	require.NoError(t, err)
	require.Zero(t, n)

	spend, err := c.PendingSpend(none)
	require.NoError(t, err)
	require.Zero(t, spend)

	totals, err := c.Totals(none)
	require.NoError(t, err)
	require.Zero(t, totals.LiveCount)
	require.Zero(t, totals.SpendWindow)
	require.Zero(t, totals.ApprovedBudget)
}

// Agency staff are unrestricted and must still see everything, so the
// predicate cannot be a blanket filter.
func TestUnrestrictedCallerSeesTheWholeBook(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := NewCampaigns(s)

	id := "svctest-scope-agency"
	cleanup(t, s, id)
	_, err := s.CreateCampaign(ctx, mkTestCampaign(id, 250, 1234, 5000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)

	before, err := c.PendingCount(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, before, int64(1), "an unrestricted caller should see the campaign just created")

	totals, err := c.Totals(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, totals.SpendWindow, int64(1234))
}
