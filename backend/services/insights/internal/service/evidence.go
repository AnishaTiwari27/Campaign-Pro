package service

import (
	"context"
	"fmt"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/platform/units"
	"campaigntrackerpro/services/campaigns"
	"campaigntrackerpro/services/insights"
)

// evidence gathers what the model is allowed to know about a campaign.
//
// Every figure is rendered here, by the same formatters the UI uses, and
// handed over as a finished string. The model never sees a raw number,
// which means it never has to format one — and a number it did not
// format is a number it cannot quietly change. It is also what makes
// the verification in Verify possible at all.
func (s *Insights) evidence(ctx context.Context, campaignID string) (insights.Evidence, error) {
	all, err := s.campaigns.All(ctx)
	if err != nil {
		return insights.Evidence{}, err
	}
	var target campaigns.Campaign
	var found bool
	for _, c := range all {
		if c.ID == campaignID {
			target, found = c, true
			break
		}
	}
	if !found {
		// Scoped reads mean a campaign outside the caller's visibility
		// simply is not in All(), so this covers "no such campaign" and
		// "not yours" with the same answer, as every other read does.
		return insights.Evidence{}, httpx.ErrNotFound
	}

	rows := s.campaigns.Enrich(ctx, all)
	row := rows[campaignID]

	ev := insights.Evidence{
		CampaignID: target.ID,
		Name:       target.Name,
		Category:   target.Category,
		Region:     target.Region,
		Status:     string(target.Status),
		FlagReason: target.FlagReason,
	}

	add := func(label, display string) {
		ev.Facts = append(ev.Facts, insights.Fact{Label: label, Display: display})
	}
	add("Reach", units.FormatReach(target.Reach))
	add("Spend", units.FormatMoney(target.Spend))
	add("Budget", units.FormatMoney(target.Budget))
	add("Category median reach", units.FormatReach(campaigns.MedianReach(
		campaigns.Running(sameCategory(all, target.Category)))))
	add("Reach against category median", units.FormatIndex(row.CategoryIndex))
	// Pace and ExpectedPace are already percentages — campaigns.Pace
	// multiplies by 100 itself. Multiplying again here reported a
	// campaign as having spent 8200% of its budget, which is the same
	// unit mistake the benchmark cost-per-lakh made, caught the same
	// way: by reading the output instead of trusting the field name.
	add("Budget spent", fmt.Sprintf("%.0f%%", row.Pace))
	if row.ExpectedPace > 0 {
		add("Budget the plan expects by now", fmt.Sprintf("%.0f%%", row.ExpectedPace))
	}
	if row.CPM > 0 {
		add("Cost per thousand", units.FormatCPM(row.CPM))
	}
	add("Days running", fmt.Sprintf("%d", target.DaysRunning))
	if target.FlightDays > 0 {
		add("Days planned", fmt.Sprintf("%d", target.FlightDays))
	}
	return ev, nil
}

func sameCategory(all []campaigns.Campaign, category string) []campaigns.Campaign {
	out := make([]campaigns.Campaign, 0, len(all))
	for _, c := range all {
		if c.Category == category {
			out = append(out, c)
		}
	}
	return out
}
