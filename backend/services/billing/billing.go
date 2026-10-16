// Package billing is the public contract of the subscription service:
// what an organisation is paying for, and whether it may use the app.
//
// Access is a property of the tenant, not of the person. An agency pays
// once and everyone in it is let in or kept out together — a per-seat
// rule would mean the approver can work and the analyst cannot, which
// is not a thing anyone buys.
package billing

import "time"

// Status mirrors the provider's vocabulary rather than inventing one,
// so a webhook can be written down without translation.
type Status string

const (
	// StatusNone is an organisation with no subscription row at all.
	// Distinct from cancelled: never started, rather than stopped.
	StatusNone     Status = "none"
	StatusTrialing Status = "trialing"
	StatusActive   Status = "active"
	// StatusPastDue is a renewal that failed. Access continues for
	// GracePeriod, because the usual cause is an expired card rather
	// than a decision to leave, and locking someone out of live
	// campaign data over a card is how you lose the renewal you were
	// trying to collect.
	StatusPastDue   Status = "past_due"
	StatusCancelled Status = "cancelled"
)

// GracePeriod is how long a past_due subscription keeps working.
const GracePeriod = 7 * 24 * time.Hour

// TrialPeriod is what a new organisation gets before paying.
const TrialPeriod = 14 * 24 * time.Hour

// Plan is one thing an organisation can buy.
//
// Prices are in paise, the smallest unit, because that is what
// Razorpay's API takes and because money in a float is a bug waiting
// for a rounding error.
type Plan struct {
	ID          string
	Name        string
	AmountPaise int64
	Interval    string
	Blurb       string
	Includes    []string
}

// Plans are the published tiers.
//
// The figures are a starting point, not researched pricing: nobody has
// tested what this is worth to an agency. They are here so the flow is
// real end to end, and they are one edit away from whatever you decide.
var Plans = []Plan{
	{
		ID: "starter", Name: "Starter", AmountPaise: 499_900, Interval: "monthly",
		Blurb: "One brand's campaigns, and the people fronting them.",
		Includes: []string{
			"Up to 25 live campaigns",
			"Approvals with a full audit trail",
			"Anomaly detection and benchmarks",
		},
	},
	{
		ID: "growth", Name: "Growth", AmountPaise: 1_499_900, Interval: "monthly",
		Blurb: "A roster, and the ad platforms it runs on.",
		Includes: []string{
			"Unlimited campaigns and creators",
			"Google Ads and Meta ingestion",
			"Scheduled reports by email",
			"Launch radar",
		},
	},
	{
		ID: "scale", Name: "Scale", AmountPaise: 4_999_900, Interval: "monthly",
		Blurb: "Several agencies' books under one roof.",
		Includes: []string{
			"Everything in Growth",
			"Client accounts with scoped access",
			"Single sign-on",
			"Written signal explanations",
		},
	},
}

// PlanByID finds a plan, reporting whether it is one that exists —
// which matters because the id arrives from a request body.
func PlanByID(id string) (Plan, bool) {
	for _, p := range Plans {
		if p.ID == id {
			return p, true
		}
	}
	return Plan{}, false
}

// Subscription is one organisation's standing.
type Subscription struct {
	OrgID  string
	Plan   string
	Status Status
	// TrialEndsAt is zero where there was never a trial.
	TrialEndsAt time.Time
	// CurrentPeriodEnd is zero where the subscription does not expire,
	// which is what the organisations grandfathered in by the migration
	// are. Zero means "no end", not "ended at the epoch".
	CurrentPeriodEnd       time.Time
	ProviderSubscriptionID string
}

// Allows reports whether this subscription lets the organisation use
// the app right now.
//
// The one rule the gate asks, kept here beside the statuses rather than
// in the middleware, so there is a single answer to "are they in" that
// a test can reach without a request.
func (s Subscription) Allows(now time.Time) bool {
	switch s.Status {
	case StatusActive:
		// A zero period end does not expire. Grandfathered rows.
		return s.CurrentPeriodEnd.IsZero() || now.Before(s.CurrentPeriodEnd)
	case StatusTrialing:
		return now.Before(s.TrialEndsAt)
	case StatusPastDue:
		// Grace runs from the end of the period that failed to renew.
		return s.CurrentPeriodEnd.IsZero() || now.Before(s.CurrentPeriodEnd.Add(GracePeriod))
	default:
		return false
	}
}

// Expires is when access runs out, or the zero time if it does not.
func (s Subscription) Expires() time.Time {
	switch s.Status {
	case StatusTrialing:
		return s.TrialEndsAt
	case StatusPastDue:
		if s.CurrentPeriodEnd.IsZero() {
			return time.Time{}
		}
		return s.CurrentPeriodEnd.Add(GracePeriod)
	default:
		return s.CurrentPeriodEnd
	}
}

// DaysLeft is how long access lasts, rounded up so a subscription with
// an hour on it reads as "1 day" rather than "0 days" — and -1 where
// there is no end at all.
func (s Subscription) DaysLeft(now time.Time) int {
	end := s.Expires()
	if end.IsZero() {
		return -1
	}
	d := end.Sub(now)
	if d <= 0 {
		return 0
	}
	return int((d + 24*time.Hour - time.Nanosecond) / (24 * time.Hour))
}
