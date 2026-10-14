package service

import (
	"math"
	"testing"

	"campaigntrackerpro/services/campaigns"
)

func camp(category string, reachLakh float64, spend int64) campaigns.Campaign {
	return campaigns.Campaign{Category: category, Reach: reachLakh, Spend: spend}
}

// Campaign.Reach is denominated in lakhs, not in people. The first
// version of this divided by a hundred thousand as well, and reported a
// cost per lakh of seven billion rupees — a number large enough to be
// obviously wrong, which is the only reason it was caught. A plausible
// wrong answer would have shipped.
func TestMedianCostPerLakhIsInRupeesPerLakhOfReach(t *testing.T) {
	// 50 lakh of reach for ₹25,00,000 is ₹50,000 per lakh.
	got := medianCostPerLakh([]campaigns.Campaign{camp("Fintech", 50, 2_500_000)}, "Fintech")
	if math.Abs(got-50_000) > 0.01 {
		t.Errorf("got %.2f, want 50000 — check the unit of Reach", got)
	}
}

// The median, not the mean: one very large campaign should describe
// itself rather than redefining the category's cost.
func TestMedianCostPerLakhResistsOneHugeCampaign(t *testing.T) {
	rows := []campaigns.Campaign{
		camp("Beauty", 10, 500_000),       // 50k per lakh
		camp("Beauty", 10, 600_000),       // 60k per lakh
		camp("Beauty", 1000, 900_000_000), // 900k per lakh, and enormous
	}
	got := medianCostPerLakh(rows, "Beauty")
	if got != 60_000 {
		t.Errorf("got %.0f, want 60000 — an outlier is setting the category's cost", got)
	}
}

// A campaign that has not delivered yet has no defined cost per lakh.
// Counting it as zero would drag the median down and make a category
// look cheapest exactly when it is delivering least.
func TestCampaignsWithNoReachYetAreExcluded(t *testing.T) {
	rows := []campaigns.Campaign{
		camp("Sports", 0, 1_000_000), // launched, nothing delivered
		camp("Sports", 20, 1_000_000),
	}
	got := medianCostPerLakh(rows, "Sports")
	if got != 50_000 {
		t.Errorf("got %.0f, want 50000 — an undelivered campaign is in the median", got)
	}
}

// An even number of values averages the middle two, and a category with
// nothing in it is zero rather than a panic on an empty slice.
func TestMedianCostPerLakhEdges(t *testing.T) {
	if got := medianCostPerLakh(nil, "Anything"); got != 0 {
		t.Errorf("empty input: got %.0f, want 0", got)
	}
	rows := []campaigns.Campaign{
		camp("FMCG", 10, 400_000), // 40k
		camp("FMCG", 10, 600_000), // 60k
	}
	if got := medianCostPerLakh(rows, "FMCG"); got != 50_000 {
		t.Errorf("even count: got %.0f, want 50000", got)
	}
}

// Other categories must not bleed into the figure.
func TestMedianCostPerLakhFiltersByCategory(t *testing.T) {
	rows := []campaigns.Campaign{
		camp("Fintech", 10, 500_000),
		camp("Beauty", 10, 9_000_000),
	}
	if got := medianCostPerLakh(rows, "Fintech"); got != 50_000 {
		t.Errorf("got %.0f, want 50000 — another category is included", got)
	}
}
