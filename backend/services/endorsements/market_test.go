package endorsements_test

import (
	"strings"
	"testing"

	"campaigntrackerpro/services/endorsements"
)

// Every figure here is someone else's, so every one has to say whose.
// A published number with no source is indistinguishable from an
// invented one once it is on the screen.
func TestEveryMarketFactCitesItsSource(t *testing.T) {
	facts := endorsements.Market()
	if len(facts) == 0 {
		t.Fatal("no market facts")
	}
	for _, f := range facts {
		if strings.TrimSpace(f.Label) == "" || strings.TrimSpace(f.Value) == "" {
			t.Errorf("incomplete fact: %+v", f)
		}
		if strings.TrimSpace(f.Source) == "" {
			t.Errorf("%s: no source named", f.Label)
		}
		if !strings.HasPrefix(f.SourceURL, "https://") {
			t.Errorf("%s: source url %q is not a link", f.Label, f.SourceURL)
		}
	}
}

// The market size is reported two ways that differ about threefold,
// because most Indian influencer spend never reaches an organised
// channel. Showing one and calling it the number would be taking a
// side in a disagreement the reader should see.
func TestTheDisputedMarketSizeIsShownBothWays(t *testing.T) {
	var organised, total bool
	for _, f := range endorsements.Market() {
		if strings.Contains(f.Label, "organised channels") && f.Period == "2025" {
			organised = true
			if f.Note == "" {
				t.Error("the organised figure needs its qualification")
			}
		}
		if strings.Contains(f.Label, "total deployment") {
			total = true
			if !strings.Contains(strings.ToLower(f.Note), "direct") {
				t.Error("the total figure should explain why it differs")
			}
		}
	}
	if !organised || !total {
		t.Error("both readings of the market size must be present")
	}
}

func TestMarketReturnsACopy(t *testing.T) {
	a := endorsements.Market()
	if len(a) < 2 {
		t.Skip()
	}
	first := a[0].Label
	a[0], a[1] = a[1], a[0]
	if endorsements.Market()[0].Label != first {
		t.Error("Market() handed out the package's own slice")
	}
}
