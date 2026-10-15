package store

import "testing"

// Matching is deliberately crude. The alternative is fuzzy matching,
// which pairs "Theorem" with "Theorem Health" and quietly records an
// outcome against the wrong brand — corrupting the only measurement
// this feature has. A missed match costs a row that stays pending; a
// wrong one costs the data, and nobody would notice.
func TestBrandMatchingIsStrictAboutWhatItAccepts(t *testing.T) {
	cases := []struct {
		campaign, launch string
		want             bool
	}{
		{"Theorem × Aanya Verma", "Theorem", true},
		{"theorem festive push", "Theorem", true},
		{"Nykaa Festive Search", "Nykaa", true},
		{"Blinkit", "Theorem", false},
		// Too short to be distinctive: "Ito" would match "Capito",
		// "Mito", "Ito" in any longer word.
		{"Capitol Hill Campaign", "Ito", false},
		{"anything", "", false},
	}
	for _, tc := range cases {
		if got := MatchesBrand(tc.campaign, tc.launch); got != tc.want {
			t.Errorf("MatchesBrand(%q, %q) = %v, want %v", tc.campaign, tc.launch, got, tc.want)
		}
	}
}
