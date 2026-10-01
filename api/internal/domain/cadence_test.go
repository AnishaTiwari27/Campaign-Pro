package domain

import (
	"testing"
	"time"
)

func mustIST(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.ParseInLocation("2006-01-02 15:04", s, istLocation)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return tm
}

func TestNextOccurrenceWeeklyMon9(t *testing.T) {
	// Wednesday -> the following Monday.
	from := mustIST(t, "2026-01-07 10:00") // a Wednesday
	got := NextOccurrence(CadenceWeeklyMon9, from)
	want := mustIST(t, "2026-01-12 09:00") // the next Monday
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	// Monday before 9am -> today at 9am.
	from = mustIST(t, "2026-01-12 06:00")
	got = NextOccurrence(CadenceWeeklyMon9, from)
	want = mustIST(t, "2026-01-12 09:00")
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	// Monday after 9am -> next Monday.
	from = mustIST(t, "2026-01-12 09:01")
	got = NextOccurrence(CadenceWeeklyMon9, from)
	want = mustIST(t, "2026-01-19 09:00")
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNextOccurrenceWeekday830SkipsWeekend(t *testing.T) {
	// Friday after 8:30am -> Monday (skips Sat/Sun).
	from := mustIST(t, "2026-01-09 09:00") // a Friday
	got := NextOccurrence(CadenceWeekday830, from)
	want := mustIST(t, "2026-01-12 08:30")
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNextOccurrenceMonthly19(t *testing.T) {
	from := mustIST(t, "2026-01-15 12:00")
	got := NextOccurrence(CadenceMonthly19, from)
	want := mustIST(t, "2026-02-01 09:00")
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	// Before the 1st-of-month cutoff -> this month.
	from = mustIST(t, "2026-01-01 06:00")
	got = NextOccurrence(CadenceMonthly19, from)
	want = mustIST(t, "2026-01-01 09:00")
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	// December -> wraps to January of next year.
	from = mustIST(t, "2026-12-15 12:00")
	got = NextOccurrence(CadenceMonthly19, from)
	want = mustIST(t, "2027-01-01 09:00")
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNextOccurrenceOnFlagIsZero(t *testing.T) {
	got := NextOccurrence(CadenceOnFlag, time.Now())
	if !got.IsZero() {
		t.Fatalf("expected zero time for on_flag, got %v", got)
	}
}

func TestIsValidCadence(t *testing.T) {
	if !IsValidCadence("weekly_mon_9") {
		t.Fatalf("expected weekly_mon_9 to be valid")
	}
	if IsValidCadence("hourly") {
		t.Fatalf("expected hourly to be invalid")
	}
}
