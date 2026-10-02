// Service tests run against a real local Postgres — this machine has no
// Docker, so testcontainers isn't available; DATABASE_URL (defaulting to
// the same dev database docker-compose would otherwise provide) stands in
// for it. Every test uses its own "svctest-" prefixed campaign IDs and
// cleans them up, so runs are safe alongside seeded demo data.
package service

import (
	"campaigntrackerpro/services/campaigns"
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"campaigntrackerpro/platform/database"
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

	// Forbidden without can_approve.
	_, err = c.Decision(ctx, id, "approve", "Someone", false)
	require.ErrorIs(t, err, ErrForbidden)

	// Invalid action.
	_, err = c.Decision(ctx, id, "not-a-real-action", "Someone", true)
	require.ErrorIs(t, err, ErrValidation)

	camp, err := c.Decision(ctx, id, "approve", "Reviewer", true)
	require.NoError(t, err)
	require.Equal(t, campaigns.ApprovalApproved, camp.Approval)

	camp, err = c.Decision(ctx, id, "reject", "Reviewer", true)
	require.NoError(t, err)
	require.Equal(t, campaigns.ApprovalRejected, camp.Approval)

	camp, err = c.Decision(ctx, id, "reopen", "Reviewer", true)
	require.NoError(t, err)
	require.Equal(t, campaigns.ApprovalPending, camp.Approval)

	events, err := s.ListAuditEventsByCampaign(ctx, id)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(events), 3, "approve, reject and reopen should each write an audit event")
}

func TestCampaignsPauseToggle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := NewCampaigns(s)

	id := "svctest-pause"
	cleanup(t, s, id)
	_, err := s.CreateCampaign(ctx, mkTestCampaign(id, 20, 100, 1000, campaigns.ApprovalApproved, campaigns.StatusLive))
	require.NoError(t, err)

	camp, err := c.Pause(ctx, id, "Reviewer")
	require.NoError(t, err)
	require.Equal(t, campaigns.StatusPaused, camp.Status)

	camp, err = c.Pause(ctx, id, "Reviewer")
	require.NoError(t, err)
	require.Equal(t, campaigns.StatusLive, camp.Status)

	// Ended campaigns can't be paused.
	endedID := "svctest-pause-ended"
	cleanup(t, s, endedID)
	_, err = s.CreateCampaign(ctx, mkTestCampaign(endedID, 20, 100, 1000, campaigns.ApprovalApproved, campaigns.StatusEnded))
	require.NoError(t, err)
	_, err = c.Pause(ctx, endedID, "Reviewer")
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

	camps, err := c.BulkDecision(ctx, ids, "approve", "Reviewer", true)
	require.NoError(t, err)
	require.Len(t, camps, 2)
	for _, camp := range camps {
		require.Equal(t, campaigns.ApprovalApproved, camp.Approval)
	}

	_, err = c.BulkDecision(ctx, ids, "approve", "Reviewer", false)
	require.ErrorIs(t, err, ErrForbidden)
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
