package models

import (
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestPacingOf_NoBudgetIsEmpty(t *testing.T) {
	start, end, today := day(2026, 6, 1), day(2026, 6, 30), day(2026, 6, 15)
	if got := PacingOf(nil, 5000, start, end, today); got != "" {
		t.Errorf("nil budget: pacing = %q, want empty", got)
	}
	zero := int64(0)
	if got := PacingOf(&zero, 5000, start, end, today); got != "" {
		t.Errorf("zero budget: pacing = %q, want empty (treated as unset)", got)
	}
}

func TestPacingOf_OnPaceAtHalfway(t *testing.T) {
	budget := int64(10000)
	start, end := day(2026, 6, 1), day(2026, 6, 31) // 30-day flight
	today := day(2026, 6, 16)                       // 15 days in = halfway
	// Half the budget spent at the halfway mark — squarely on pace.
	if got := PacingOf(&budget, 5000, start, end, today); got != "on" {
		t.Errorf("pacing = %q, want \"on\"", got)
	}
}

func TestPacingOf_OverPace(t *testing.T) {
	budget := int64(10000)
	start, end := day(2026, 6, 1), day(2026, 6, 31)
	today := day(2026, 6, 16) // halfway — expected spend is ~5000
	if got := PacingOf(&budget, 9000, start, end, today); got != "over" {
		t.Errorf("pacing = %q, want \"over\" (9000 spent vs ~5000 expected)", got)
	}
}

func TestPacingOf_UnderPace(t *testing.T) {
	budget := int64(10000)
	start, end := day(2026, 6, 1), day(2026, 6, 31)
	today := day(2026, 6, 16) // halfway — expected spend is ~5000
	if got := PacingOf(&budget, 1000, start, end, today); got != "under" {
		t.Errorf("pacing = %q, want \"under\" (1000 spent vs ~5000 expected)", got)
	}
}

func TestPacingOf_CompletedCampaignComparesAgainstFullBudget(t *testing.T) {
	budget := int64(10000)
	start, end := day(2026, 1, 1), day(2026, 1, 31)
	today := day(2026, 6, 1) // long after the flight ended — elapsed clamps to 1

	if got := PacingOf(&budget, 12000, start, end, today); got != "over" {
		t.Errorf("finished 20%% over budget: pacing = %q, want \"over\"", got)
	}
	if got := PacingOf(&budget, 9800, start, end, today); got != "on" {
		t.Errorf("finished 2%% under budget: pacing = %q, want \"on\" (within tolerance)", got)
	}
}

func TestPacingOf_HasNotStartedYet(t *testing.T) {
	budget := int64(10000)
	start, end := day(2026, 7, 1), day(2026, 7, 31)
	today := day(2026, 6, 1) // before the flight even begins — elapsed clamps to 0

	if got := PacingOf(&budget, 0, start, end, today); got != "on" {
		t.Errorf("no spend yet, hasn't started: pacing = %q, want \"on\"", got)
	}
	if got := PacingOf(&budget, 500, start, end, today); got != "over" {
		t.Errorf("any spend before the flight starts: pacing = %q, want \"over\"", got)
	}
}

func TestPacingOf_ZeroLengthFlightDoesNotPanic(t *testing.T) {
	budget := int64(10000)
	same := day(2026, 6, 15)
	// start == end == today: the divide-by-zero guard clamps totalDays to
	// 1, and elapsed-since-start is exactly 0 either way — this just
	// confirms it resolves to a value (any spend at all on a same-day
	// flight reads as ahead of schedule) instead of panicking or NaN-ing.
	got := PacingOf(&budget, 12000, same, same, same)
	if got != "over" {
		t.Errorf("pacing = %q, want \"over\"", got)
	}
}
