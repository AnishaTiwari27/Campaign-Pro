package service

import (
	"strings"
	"testing"

	"campaigntrackerpro/services/insights"
)

func ev(facts ...[2]string) insights.Evidence {
	e := insights.Evidence{Name: "Blinkit", FlagReason: "Reach outlier"}
	for _, f := range facts {
		e.Facts = append(e.Facts, insights.Fact{Label: f[0], Display: f[1]})
	}
	return e
}

var sample = ev(
	[2]string{"Reach", "1.80Cr"},
	[2]string{"Spend", "₹98.0L"},
	[2]string{"Category median reach", "41.0L"},
	[2]string{"Reach against category median", "4.4x"},
	[2]string{"Budget spent", "82%"},
	[2]string{"Days running", "18"},
)

// The case the guard exists for. A model writing about money produces
// a plausible figure very easily, and a wrong one here would sit beside
// real figures and be believed.
func TestAFigureNotInTheEvidenceIsRejected(t *testing.T) {
	bad, ok := Verify("Reach of 1.80Cr is well ahead, and spend is tracking at ₹97.5L.", sample)
	if ok {
		t.Fatal("accepted a spend figure that was never supplied")
	}
	if len(bad) != 1 || !strings.Contains(bad[0], "97.5") {
		t.Errorf("expected the invented figure to be named, got %v", bad)
	}
}

func TestFiguresQuotedFromTheEvidencePass(t *testing.T) {
	text := "At 1.80Cr this is 4.4x the category median of 41.0L, on ₹98.0L of spend and 82% of budget."
	if bad, ok := Verify(text, sample); !ok {
		t.Errorf("rejected figures that were all supplied: %v", bad)
	}
}

// A model paraphrasing a number is one step from changing it, so the
// comparison is on the rendered string and a reworded figure fails.
func TestAParaphrasedFigureIsRejected(t *testing.T) {
	if _, ok := Verify("Spend has reached 98 lakh so far.", sample); ok {
		t.Error("accepted a figure rewritten in words — only the rendered form is allowed")
	}
}

// Ordinary prose has small integers in it. Banning them outright makes
// the feature unusable, and they cannot misstate a monetary figure.
func TestSmallCountingNumbersAreAllowed(t *testing.T) {
	for _, s := range []string{
		"There are 2 options here.",
		"On day 18 of the flight this is unusual.",
		"The top 3 creatives account for most of it.",
	} {
		if bad, ok := Verify(s, sample); !ok {
			t.Errorf("rejected ordinary prose %q: %v", s, bad)
		}
	}
}

// A large bare number is not a counting word — it is a figure with its
// unit dropped, which is exactly how a misstatement would look.
func TestALargeBareNumberIsRejected(t *testing.T) {
	if _, ok := Verify("Spend is around 4500000 against the plan.", sample); ok {
		t.Error("accepted a large bare number")
	}
}

// Formatting differences a model might introduce must not smuggle a
// figure through: separators and case are normalised, so these are the
// same figure and are allowed.
func TestHarmlessFormattingDifferencesAreTolerated(t *testing.T) {
	if bad, ok := Verify("Spend stands at ₹98.0l today.", sample); !ok {
		t.Errorf("rejected a case difference: %v", bad)
	}
}

// Prose with no figures at all is trivially safe.
func TestTextWithNoFiguresPasses(t *testing.T) {
	if _, ok := Verify("Reach is running well ahead of the category baseline.", sample); !ok {
		t.Error("rejected text containing no figures")
	}
}

// The fallback is assembled from the evidence, so it must pass its own
// guard — if it ever did not, the safe path would be the unsafe one.
func TestTheFallbackPassesItsOwnVerification(t *testing.T) {
	text := fallback(sample)
	if bad, ok := Verify(text, sample); !ok {
		t.Errorf("the fallback produced figures it could not justify: %v\n%s", bad, text)
	}
	if !strings.Contains(text, "1.80Cr") {
		t.Errorf("the fallback dropped the evidence: %q", text)
	}
}
