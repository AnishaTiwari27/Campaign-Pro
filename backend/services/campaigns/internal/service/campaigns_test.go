// Service tests run against a real local Postgres — this machine has no
// Docker, so testcontainers isn't available; DATABASE_URL (defaulting to
// the same dev database docker-compose would otherwise provide) stands in
// for it. Every test uses its own "svctest-" prefixed campaign IDs and
// cleans them up, so runs are safe alongside seeded demo data.
package service

import (
	"campaigntrackerpro/services/campaigns"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/campaigns/internal/store"

	"github.com/stretchr/testify/require"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://campaign_tracker_pro:campaign_tracker_pro_dev_pw@localhost:5432/campaign_tracker_pro?sslmode=disable"
	}
	pool, err := database.Connect(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping: no local Postgres available (%v)", err)
	}
	t.Cleanup(pool.Close)
	return store.New(database.New(pool))
}

func mkTestCampaign(id string, reach float64, spend, budget int64, approval campaigns.Approval, status campaigns.Status) campaigns.Campaign {
	return campaigns.Campaign{
		ID: id, Name: "Test " + id, SubjectType: campaigns.SubjectBrand, Initials: "TC",
		Category: "Fintech", Region: "Delhi NCR", AdType: campaigns.AdSocial, Platform: "Instagram",
		Status: status, DaysRunning: 10, Reach: reach, Spend: spend, Budget: budget,
		Frequency: 1.5, Approval: approval, CurveShape: campaigns.CurveSteady,
	}
}

func cleanup(t *testing.T, s *store.Store, ids ...string) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		for _, id := range ids {
			_ = s.Exec(ctx, `DELETE FROM campaigns WHERE id = $1`, id)
		}
	})
}

// testActor creates a real user row, because audit_events.user_id is a
// foreign key — a made-up id would make the attribution assertions pass
// against a write the database would have rejected in production.
func testActor(t *testing.T, s *store.Store) Actor {
	t.Helper()
	ctx := context.Background()
	email := fmt.Sprintf("svctest-%s-%d@example.test", t.Name(), time.Now().UnixNano())
	u, err := s.CreateUser(ctx, email, "Svctest Reviewer", "approver", true)
	require.NoError(t, err)
	t.Cleanup(func() {
		// The audit rows first: user_id has no ON DELETE, so the user row
		// cannot go while anything still points at it. Not all of those rows
		// hang off a campaign this test cleaned up.
		_ = s.Exec(ctx, `DELETE FROM audit_events WHERE user_id = $1`, u.ID)
		_ = s.Exec(ctx, `DELETE FROM users WHERE id = $1`, u.ID)
	})
	return Actor{ID: u.ID, Name: u.Name}
}

func TestCampaignsListFilterSortPage(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := NewCampaigns(s)

	ids := []string{"svctest-a", "svctest-b", "svctest-c"}
	cleanup(t, s, ids...)

	_, err := s.CreateCampaign(ctx, mkTestCampaign(ids[0], 10, 100, 1000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)
	_, err = s.CreateCampaign(ctx, mkTestCampaign(ids[1], 50, 100, 1000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)
	_, err = s.CreateCampaign(ctx, mkTestCampaign(ids[2], 30, 100, 1000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)

	result, err := c.List(ctx, campaigns.ListParams{
		Category: "Fintech", Region: "Delhi NCR", Sort: "reach", Dir: "desc", Page: 1, Per: 10,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Items), 3)

	// Find our three test rows in the result and confirm they're ordered by
	// reach descending relative to each other.
	var positions []int
	for i, row := range result.Items {
		for _, id := range ids {
			if row.ID == id {
				positions = append(positions, i)
			}
		}
	}
	require.Len(t, positions, 3)

	byID := map[string]int{}
	for i, row := range result.Items {
		byID[row.ID] = i
	}
	require.Less(t, byID[ids[1]], byID[ids[2]], "50L campaign should sort before 30L campaign (desc)")
	require.Less(t, byID[ids[2]], byID[ids[0]], "30L campaign should sort before 10L campaign (desc)")

	// Pagination: per=1 should return exactly 1 item and report total pages.
	paged, err := c.List(ctx, campaigns.ListParams{Category: "Fintech", Region: "Delhi NCR", Page: 1, Per: 1})
	require.NoError(t, err)
	require.Len(t, paged.Items, 1)
	require.GreaterOrEqual(t, paged.Pages, 3)
}

func TestCampaignsDecisionApproveRejectReopen(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := NewCampaigns(s)

	id := "svctest-decision"
	cleanup(t, s, id)
	_, err := s.CreateCampaign(ctx, mkTestCampaign(id, 20, 100, 1000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)

	actor := testActor(t, s)

	// Forbidden without can_approve.
	_, err = c.Decision(ctx, id, "approve", actor, false)
	require.ErrorIs(t, err, ErrForbidden)

	// Invalid action.
	_, err = c.Decision(ctx, id, "not-a-real-action", actor, true)
	require.ErrorIs(t, err, ErrValidation)

	camp, err := c.Decision(ctx, id, "approve", actor, true)
	require.NoError(t, err)
	require.Equal(t, campaigns.ApprovalApproved, camp.Approval)

	camp, err = c.Decision(ctx, id, "reject", actor, true)
	require.NoError(t, err)
	require.Equal(t, campaigns.ApprovalRejected, camp.Approval)

	camp, err = c.Decision(ctx, id, "reopen", actor, true)
	require.NoError(t, err)
	require.Equal(t, campaigns.ApprovalPending, camp.Approval)

	events, err := s.ListAuditEventsByCampaign(ctx, id)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(events), 3, "approve, reject and reopen should each write an audit event")

	// The point of the trail: every decision is attributable to an account,
	// not just to whatever display name was current at the time.
	for _, e := range events {
		require.Equal(t, actor.ID, e.UserID, "decision %q must record the acting user id", e.Action)
		require.Equal(t, actor.Name, e.Actor)
	}
}

// TestDecisionBlockedOverBudget covers the approval guard: the budget flags
// refuse an approval, while reject and reopen stay available because they are
// how an approver responds to an over-budget campaign.
func TestDecisionBlockedOverBudget(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := NewCampaigns(s)
	actor := testActor(t, s)

	// 1000 of 1000 spent: pace 100.
	id := "svctest-blocked-over"
	cleanup(t, s, id)
	_, err := s.CreateCampaign(ctx, mkTestCampaign(id, 20, 1000, 1000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)

	_, err = c.Decision(ctx, id, "approve", actor, true)
	var blocked ApprovalBlockedError
	require.ErrorAs(t, err, &blocked)
	require.Contains(t, blocked.Blocked, id)
	require.Contains(t, blocked.Reason(), "100%")
	require.ErrorIs(t, err, httpx.ErrConflict, "a missed check must still answer 409, not 500")

	// The refusal wrote nothing.
	after, err := s.GetCampaign(ctx, id)
	require.NoError(t, err)
	require.Equal(t, campaigns.ApprovalPending, after.Approval)

	// Rejecting it is still allowed.
	rejected, err := c.Decision(ctx, id, "reject", actor, true)
	require.NoError(t, err)
	require.Equal(t, campaigns.ApprovalRejected, rejected.Approval)

	// 960 of 1000: pace 96, inside the "nearly exhausted" band.
	nearID := "svctest-blocked-near"
	cleanup(t, s, nearID)
	_, err = s.CreateCampaign(ctx, mkTestCampaign(nearID, 20, 960, 1000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)
	_, err = c.Decision(ctx, nearID, "approve", actor, true)
	require.ErrorAs(t, err, &blocked)

	// A scheduled campaign has spent nothing, so its pre-flight sign-off —
	// the case the queue exists for — is never blocked.
	schedID := "svctest-blocked-sched"
	cleanup(t, s, schedID)
	_, err = s.CreateCampaign(ctx, mkTestCampaign(schedID, 0, 0, 1000, campaigns.ApprovalPending, campaigns.StatusScheduled))
	require.NoError(t, err)
	approved, err := c.Decision(ctx, schedID, "approve", actor, true)
	require.NoError(t, err)
	require.Equal(t, campaigns.ApprovalApproved, approved.Approval)
}

// TestBulkDecisionBlockedIsAllOrNothing proves one over-budget campaign in a
// selection approves none of them, so the caller is never left guessing which
// of their selection went through.
func TestBulkDecisionBlockedIsAllOrNothing(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := NewCampaigns(s)
	actor := testActor(t, s)

	okID, badID := "svctest-bulkblock-ok", "svctest-bulkblock-bad"
	cleanup(t, s, okID, badID)
	_, err := s.CreateCampaign(ctx, mkTestCampaign(okID, 20, 100, 1000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)
	_, err = s.CreateCampaign(ctx, mkTestCampaign(badID, 20, 1100, 1000, campaigns.ApprovalPending, campaigns.StatusLive))
	require.NoError(t, err)

	_, err = c.BulkDecision(ctx, []string{okID, badID}, "approve", actor, true)
	var blocked ApprovalBlockedError
	require.ErrorAs(t, err, &blocked)
	require.Contains(t, blocked.Blocked, badID)
	require.NotContains(t, blocked.Blocked, okID)

	// Neither was touched — not even the one that would have passed.
	for _, id := range []string{okID, badID} {
		camp, gerr := s.GetCampaign(ctx, id)
		require.NoError(t, gerr)
		require.Equal(t, campaigns.ApprovalPending, camp.Approval, "%s should be untouched", id)
	}
}

func TestCampaignsPauseToggle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := NewCampaigns(s)

	id := "svctest-pause"
	cleanup(t, s, id)
	_, err := s.CreateCampaign(ctx, mkTestCampaign(id, 20, 100, 1000, campaigns.ApprovalApproved, campaigns.StatusLive))
	require.NoError(t, err)

	actor := testActor(t, s)

	camp, err := c.Pause(ctx, id, actor)
	require.NoError(t, err)
	require.Equal(t, campaigns.StatusPaused, camp.Status)

	camp, err = c.Pause(ctx, id, actor)
	require.NoError(t, err)
	require.Equal(t, campaigns.StatusLive, camp.Status)

	// Ended campaigns can't be paused.
	endedID := "svctest-pause-ended"
	cleanup(t, s, endedID)
	_, err = s.CreateCampaign(ctx, mkTestCampaign(endedID, 20, 100, 1000, campaigns.ApprovalApproved, campaigns.StatusEnded))
	require.NoError(t, err)
	_, err = c.Pause(ctx, endedID, actor)
	require.ErrorIs(t, err, ErrValidation)
}

func TestCampaignsBulkDecision(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := NewCampaigns(s)

	ids := []string{"svctest-bulk-1", "svctest-bulk-2"}
	cleanup(t, s, ids...)
	for _, id := range ids {
		_, err := s.CreateCampaign(ctx, mkTestCampaign(id, 20, 100, 1000, campaigns.ApprovalPending, campaigns.StatusLive))
		require.NoError(t, err)
	}

	actor := testActor(t, s)

	camps, err := c.BulkDecision(ctx, ids, "approve", actor, true)
	require.NoError(t, err)
	require.Len(t, camps, 2)
	for _, camp := range camps {
		require.Equal(t, campaigns.ApprovalApproved, camp.Approval)
	}

	_, err = c.BulkDecision(ctx, ids, "approve", actor, false)
	require.ErrorIs(t, err, ErrForbidden)

	for _, id := range ids {
		events, eerr := s.ListAuditEventsByCampaign(ctx, id)
		require.NoError(t, eerr)
		require.NotEmpty(t, events)
		require.Equal(t, actor.ID, events[0].UserID, "a bulk decision is attributable too")
	}
}

func TestAnomalyDetectorFlagsAndClears(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := NewCampaigns(s)
	detector := NewAnomalyDetector(c, slog.New(slog.NewTextHandler(io.Discard, nil)))

	id := "svctest-anomaly"
	cleanup(t, s, id)
	// Over-budget: spend > budget -> should flag "Over budget pace".
	_, err := s.CreateCampaign(ctx, mkTestCampaign(id, 20, 1100, 1000, campaigns.ApprovalApproved, campaigns.StatusLive))
	require.NoError(t, err)

	_, err = detector.Run(ctx)
	require.NoError(t, err)

	camp, err := s.GetCampaign(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "Over budget pace", camp.FlagReason)

	// Fix the spend and re-run: the flag should clear.
	_, err = s.UpdateCampaignFlag(ctx, id, camp.FlagReason) // no-op, just confirming current state readable
	require.NoError(t, err)
	err = s.Exec(ctx, `UPDATE campaigns SET spend = 100 WHERE id = $1`, id)
	require.NoError(t, err)

	_, err = detector.Run(ctx)
	require.NoError(t, err)
	camp, err = s.GetCampaign(ctx, id)
	require.NoError(t, err)
	require.Empty(t, camp.FlagReason)
}
