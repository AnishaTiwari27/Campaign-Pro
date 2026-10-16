package billing_test

import (
	"testing"
	"time"

	"campaigntrackerpro/services/billing"
)

func TestAllows(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	cases := []struct {
		name string
		sub  billing.Subscription
		want bool
		why  string
	}{
		{
			"no row at all", billing.Subscription{Status: billing.StatusNone}, false,
			"an organisation that never subscribed must not be let in",
		},
		{
			"grandfathered — active with no end",
			billing.Subscription{Status: billing.StatusActive}, true,
			"the migration writes these; a zero period end means it does not expire, " +
				"and reading it as 'expired at the epoch' would lock out every existing tenant",
		},
		{
			"active, paid up",
			billing.Subscription{Status: billing.StatusActive, CurrentPeriodEnd: now.Add(10 * day)}, true, "",
		},
		{
			"active but the period has passed",
			billing.Subscription{Status: billing.StatusActive, CurrentPeriodEnd: now.Add(-time.Minute)}, false,
			"a status left stale by a missed webhook must not outlive the period it was for",
		},
		{
			"trialing, days left",
			billing.Subscription{Status: billing.StatusTrialing, TrialEndsAt: now.Add(3 * day)}, true, "",
		},
		{
			"trial over",
			billing.Subscription{Status: billing.StatusTrialing, TrialEndsAt: now.Add(-time.Second)}, false, "",
		},
		{
			"past due, inside grace",
			billing.Subscription{Status: billing.StatusPastDue, CurrentPeriodEnd: now.Add(-2 * day)}, true,
			"a failed card should not lock someone out of live campaigns on day one",
		},
		{
			"past due, grace exhausted",
			billing.Subscription{Status: billing.StatusPastDue, CurrentPeriodEnd: now.Add(-8 * day)}, false, "",
		},
		{
			"cancelled",
			billing.Subscription{Status: billing.StatusCancelled, CurrentPeriodEnd: now.Add(10 * day)}, false,
			"cancelled is a decision, not an expiry — the date on it is not access",
		},
	}

	for _, c := range cases {
		if got := c.sub.Allows(now); got != c.want {
			t.Errorf("%s: Allows = %v, want %v%s", c.name, got, c.want, note(c.why))
		}
	}
}

func note(why string) string {
	if why == "" {
		return ""
	}
	return " — " + why
}

func TestDaysLeft(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

	// Rounded up: an hour left is a day left, not zero days.
	trial := billing.Subscription{Status: billing.StatusTrialing, TrialEndsAt: now.Add(time.Hour)}
	if got := trial.DaysLeft(now); got != 1 {
		t.Errorf("an hour of trial reads as %d days, want 1", got)
	}

	perpetual := billing.Subscription{Status: billing.StatusActive}
	if got := perpetual.DaysLeft(now); got != -1 {
		t.Errorf("a subscription with no end reads as %d, want -1", got)
	}

	done := billing.Subscription{Status: billing.StatusTrialing, TrialEndsAt: now.Add(-time.Hour)}
	if got := done.DaysLeft(now); got != 0 {
		t.Errorf("an expired trial reads as %d days, want 0", got)
	}
}

// The id arrives in a request body, so an unknown one must not resolve.
func TestPlanByID(t *testing.T) {
	if _, ok := billing.PlanByID("growth"); !ok {
		t.Error("growth is a published plan and did not resolve")
	}
	for _, bad := range []string{"", "free", "GROWTH", "../growth"} {
		if _, ok := billing.PlanByID(bad); ok {
			t.Errorf("%q resolved to a plan", bad)
		}
	}
	for _, p := range billing.Plans {
		if p.AmountPaise <= 0 {
			t.Errorf("plan %s costs %d paise", p.ID, p.AmountPaise)
		}
	}
}
