package radar

import (
	"strings"
	"testing"
	"time"
)

func at(daysAgo int) time.Time { return time.Now().AddDate(0, 0, -daysAgo) }

// The band is a pure function of the stored signals, so the same
// launch always lands in the same place and the estimate can be
// recomputed and argued with rather than just believed.
func TestBandingFollowsTheStatedSignals(t *testing.T) {
	cases := []struct {
		name string
		in   Launch
		want Readiness
	}{
		{"fresh, noticed, has a site", Launch{LaunchedAt: at(3), Traction: 180, HasSite: true}, ReadinessSoon},
		{"fresh but nowhere to send traffic", Launch{LaunchedAt: at(3), Traction: 180}, ReadinessBuilding},
		{"fresh but nobody noticed", Launch{LaunchedAt: at(5), Traction: 2, HasSite: true}, ReadinessBuilding},
		{"older but still has attention", Launch{LaunchedAt: at(60), Traction: 400, HasSite: true}, ReadinessBuilding},
		{"older and quiet", Launch{LaunchedAt: at(60), Traction: 3, HasSite: true}, ReadinessQuiet},
		{"long past a launch push", Launch{LaunchedAt: at(300), Traction: 900, HasSite: true}, ReadinessQuiet},
	}
	for _, tc := range cases {
		got, basis := Assess(tc.in, time.Now())
		if got != tc.want {
			t.Errorf("%s: got %q, want %q (basis: %s)", tc.name, got, tc.want, basis)
		}
		if basis == "" {
			t.Errorf("%s: banded with no stated basis", tc.name)
		}
	}
}

// Without a launch date the main signal is missing, and a band
// computed without it would be a guess wearing a label.
func TestNoLaunchDateIsUnclearRatherThanGuessed(t *testing.T) {
	got, basis := Assess(Launch{Traction: 500, HasSite: true}, time.Now())
	if got != ReadinessUnclear {
		t.Errorf("got %q, want unclear", got)
	}
	if !strings.Contains(basis, "no launch date") {
		t.Errorf("basis should say what is missing: %q", basis)
	}
}

// Every band has to name the signals it came from, or the estimate
// cannot be argued with.
func TestTheBasisNamesTheSignals(t *testing.T) {
	_, basis := Assess(Launch{Source: "hacker-news", LaunchedAt: at(2), Traction: 140, HasSite: true}, time.Now())
	for _, want := range []string{"launched", "140", "has a site"} {
		if !strings.Contains(basis, want) {
			t.Errorf("basis %q is missing %q", basis, want)
		}
	}
}

const hnBody = `{"hits":[
 {"objectID":"111","title":"Show HN: Theorem – Offline PDF reader that writes highlights","url":"https://theorem.app","points":180,"num_comments":42,"created_at":"2026-10-14T09:00:00.000Z"},
 {"objectID":"222","title":"Show HN: A free sail routing tool","url":"https://sail.example","points":12,"num_comments":3,"created_at":"2026-10-13T09:00:00.000Z"},
 {"objectID":"333","title":"Show HN: No date here","url":"https://x.example","points":5,"num_comments":1,"created_at":"not-a-date"}
]}`

// "Show HN: Theorem – An offline reader" is a name and a tagline.
// Storing the lot as the brand name makes every entry begin "Show HN:".
func TestShowHNTitlesSplitIntoNameAndTagline(t *testing.T) {
	got, err := parseHN([]byte(hnBody))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d launches, want 2 — the undated one should be skipped", len(got))
	}
	if got[0].Name != "Theorem" {
		t.Errorf("name %q, want Theorem", got[0].Name)
	}
	if !strings.Contains(got[0].Tagline, "Offline PDF reader") {
		t.Errorf("tagline %q", got[0].Tagline)
	}
	// A title with no separator is all name and no tagline.
	if got[1].Name != "A free sail routing tool" || got[1].Tagline != "" {
		t.Errorf("unseparated title became %q / %q", got[1].Name, got[1].Tagline)
	}
}

func TestTractionCombinesPointsAndComments(t *testing.T) {
	got, _ := parseHN([]byte(hnBody))
	if got[0].Traction != 222 {
		t.Errorf("traction %d, want 222 (180 points + 42 comments)", got[0].Traction)
	}
}

// A launch with no date cannot be banded, so it is skipped at the
// source rather than stored as unclear forever.
func TestAnUndatedLaunchIsSkipped(t *testing.T) {
	got, _ := parseHN([]byte(hnBody))
	for _, l := range got {
		if l.ExternalID == "333" {
			t.Error("an undated launch was kept")
		}
	}
}

func TestSplitHandlesTheSeparatorsThatActuallyAppear(t *testing.T) {
	cases := map[string][2]string{
		"Show HN: Acme – does things": {"Acme", "does things"},
		"Show HN: Acme — does things": {"Acme", "does things"},
		"Show HN: Acme - does things": {"Acme", "does things"},
		"Show HN: Acme: does things":  {"Acme", "does things"},
		"Show HN: JustAName":          {"JustAName", ""},
	}
	for title, want := range cases {
		name, tagline := splitShowHN(title)
		if name != want[0] || tagline != want[1] {
			t.Errorf("%q -> %q / %q, want %q / %q", title, name, tagline, want[0], want[1])
		}
	}
}
