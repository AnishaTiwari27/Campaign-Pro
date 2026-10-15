package campaigns

import (
	"math"
	"testing"
)

func running(reach float64, spend, budget int64, days, flight int, shape CurveShape) Campaign {
	return Campaign{
		Reach: reach, Spend: spend, Budget: budget,
		DaysRunning: days, FlightDays: flight, CurveShape: shape,
	}
}

// The projection inverts the curve the campaign is already assumed to
// follow, so a campaign exactly on its curve projects to exactly its
// implied total. This is the property the whole thing rests on.
func TestACampaignOnItsCurveProjectsToItsImpliedTotal(t *testing.T) {
	const total, flight, age = 100.0, 30, 18
	for _, shape := range []CurveShape{CurveSteady, CurveFast, CurveSlow} {
		delivered := ValueAtAge(total, shape, flight, age)
		got := Project(running(delivered, 0, 0, age, flight, shape))
		if !got.Available {
			t.Fatalf("%s: no projection at %d/%d", shape, age, flight)
		}
		if math.Abs(got.ProjectedReach-total) > 0.001 {
			t.Errorf("%s: projected %.3f, want %.3f", shape, got.ProjectedReach, total)
		}
	}
}

// Dividing by the delivered share is what makes this work and what
// makes it dangerous early: at day 1 of 30 a single day's noise is
// multiplied thirtyfold. Refusing is the honest answer.
func TestItRefusesToProjectTooEarly(t *testing.T) {
	got := Project(running(10, 50_000, 500_000, 2, 30, CurveSteady))
	if got.Available {
		t.Error("projected from 2 days of a 30-day flight")
	}
	if got.ElapsedShare == 0 {
		t.Error("the elapsed share should still be reported, so the UI can say why")
	}
}

// A finished campaign has actuals. Putting a guess beside a fact is
// worse than showing nothing.
func TestAFinishedCampaignIsNotProjected(t *testing.T) {
	if Project(running(90, 500_000, 600_000, 30, 30, CurveSteady)).Available {
		t.Error("projected a campaign that has already finished")
	}
	if Project(running(90, 500_000, 600_000, 44, 30, CurveSteady)).Available {
		t.Error("projected a campaign past its flight")
	}
}

func TestAFlightWithNoLengthCannotBeProjected(t *testing.T) {
	if Project(running(10, 100, 1000, 5, 0, CurveSteady)).Available {
		t.Error("projected without a flight length")
	}
}

// The question the page actually needs: will this run out, and when.
func TestOverspendIsProjectedWithTheDayItRunsOut(t *testing.T) {
	// Half the flight gone, and already 70% of budget spent on a
	// steady curve — that lands well over.
	c := running(50, 700_000, 1_000_000, 15, 30, CurveSteady)
	got := Project(c)
	if !got.Available {
		t.Fatal("no projection")
	}
	if got.OverspendBy <= 0 {
		t.Fatalf("projected spend %d against a budget of %d — expected an overspend",
			got.ProjectedSpend, c.Budget)
	}
	if got.ExhaustsOnDay <= 0 || got.ExhaustsOnDay > c.FlightDays {
		t.Errorf("exhaustion day %d is outside the flight", got.ExhaustsOnDay)
	}
	// It must run out before the end, or it would not be an overspend.
	if got.ExhaustsOnDay >= c.FlightDays {
		t.Errorf("exhausts on day %d of %d — that is not early", got.ExhaustsOnDay, c.FlightDays)
	}
}

// A campaign tracking under budget must not be reported as running out.
func TestAnUnderspendingCampaignHasNoExhaustionDay(t *testing.T) {
	got := Project(running(50, 300_000, 1_000_000, 15, 30, CurveSteady))
	if !got.Available {
		t.Fatal("no projection")
	}
	if got.OverspendBy != 0 || got.ExhaustsOnDay != 0 {
		t.Errorf("reported an overspend for a campaign inside budget: %+v", got)
	}
}

// Elapsed share is the confidence signal, and the only honest one: it
// is what governs how far the arithmetic is extrapolating.
func TestElapsedShareTracksProgressThroughTheFlight(t *testing.T) {
	for _, tc := range []struct {
		days, flight int
		want         float64
	}{{15, 30, 0.5}, {24, 30, 0.8}, {9, 30, 0.3}} {
		got := Project(running(50, 100, 1000, tc.days, tc.flight, CurveSteady))
		if math.Abs(got.ElapsedShare-tc.want) > 0.001 {
			t.Errorf("%d/%d: elapsed %.3f, want %.3f", tc.days, tc.flight, got.ElapsedShare, tc.want)
		}
	}
}

// A fast campaign has most of its delivery behind it at the
// same age, so it projects to less than a slow burn showing the same
// numbers. If that ever inverts, the curve is being read backwards.
func TestCurveShapeChangesTheProjection(t *testing.T) {
	front := Project(running(50, 0, 0, 15, 30, CurveFast))
	slow := Project(running(50, 0, 0, 15, 30, CurveSlow))
	if !front.Available || !slow.Available {
		t.Fatal("no projection")
	}
	if front.ProjectedReach >= slow.ProjectedReach {
		t.Errorf("fast projected %.1f, slow burn %.1f — a fast campaign has more of its delivery already in, so it should project lower",
			front.ProjectedReach, slow.ProjectedReach)
	}
}

// A creator-fronted campaign is not judged "under-delivering" against
// the all-campaign median, because that median mixes creator tiers
// that are not comparable: a four-million-follower creator fails it by
// arithmetic rather than by performing badly. Kusha Kapila at 9 lakh
// was flagged against a median pulled up by Virat Kohli, which is the
// comparison the Influencers page explicitly refuses to make.
func TestCreatorCampaignsAreNotFlaggedUnderDeliveringOnRawMedian(t *testing.T) {
	low := Campaign{Reach: 9, Spend: 100, Budget: 1000, DaysRunning: 10, CreatorID: "kusha-kapila"}
	if got := DetectAnomaly(low, 0.2, 0.22); got.Flagged {
		t.Errorf("flagged a creator campaign as %q on a raw-median comparison", got.Reason)
	}
	// A brand campaign with the same numbers still flags: brands are
	// comparable with each other, and that judgement is about a line of
	// spend rather than somebody's standing.
	brand := Campaign{Reach: 9, Spend: 100, Budget: 1000, DaysRunning: 10}
	if got := DetectAnomaly(brand, 0.2, 0.22); !got.Flagged || got.Reason != "Under-delivering" {
		t.Errorf("brand campaign should still flag, got %+v", got)
	}
}

// Budget rules are about money, not comparability, so they apply to
// everyone. A creator campaign that is over budget must still say so.
func TestBudgetRulesStillApplyToCreatorCampaigns(t *testing.T) {
	over := Campaign{Reach: 50, Spend: 1200, Budget: 1000, DaysRunning: 10, CreatorID: "someone"}
	if got := DetectAnomaly(over, 1.0, 1.0); !got.Flagged || got.Reason != "Over budget pace" {
		t.Errorf("got %+v, want an over-budget flag", got)
	}
}
