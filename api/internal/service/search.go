package service

import (
	"context"
	"fmt"
	"strings"

	"campaigntrackerpro/internal/domain"
)

type SearchResult struct {
	Kind     string // "section" | "campaign" | "region" | "action"
	ID       string
	Title    string
	Subtitle string
}

var paletteSections = []struct{ id, title string }{
	{"overview", "Overview"}, {"campaigns", "Campaigns"}, {"approvals", "Approvals"},
	{"regions", "Regions"}, {"benchmarks", "Benchmarks"}, {"reports", "Reports"}, {"settings", "Settings"},
}

// Search backs the command palette: "Go to" sections always show; campaigns,
// regions and actions only show once the caller has typed something.
func (c *Campaigns) Search(ctx context.Context, q string) ([]SearchResult, error) {
	needle := strings.ToLower(strings.TrimSpace(q))
	var results []SearchResult

	for _, s := range paletteSections {
		if needle == "" || strings.Contains(strings.ToLower(s.title), needle) {
			results = append(results, SearchResult{Kind: "section", ID: s.id, Title: s.title})
		}
	}

	if needle == "" {
		return results, nil
	}

	all, err := c.Store.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	for _, camp := range all {
		if len(results) >= 40 {
			break
		}
		if strings.Contains(strings.ToLower(camp.Name), needle) ||
			strings.Contains(strings.ToLower(camp.Category), needle) ||
			strings.Contains(strings.ToLower(camp.Region), needle) {
			results = append(results, SearchResult{
				Kind: "campaign", ID: camp.ID, Title: camp.Name,
				Subtitle: fmt.Sprintf("%s · %s", camp.Region, domain.FormatReach(camp.Reach)),
			})
		}
	}

	regions, err := c.Regions(ctx)
	if err == nil {
		for _, r := range regions {
			if len(results) >= 40 {
				break
			}
			if strings.Contains(strings.ToLower(r.Region), needle) {
				results = append(results, SearchResult{Kind: "region", ID: r.Region, Title: r.Region})
			}
		}
	}

	if pending, err := c.Store.CountPendingApprovals(ctx); err == nil && pending > 0 {
		if strings.Contains("review pending approvals", needle) {
			results = append(results, SearchResult{Kind: "action", ID: "approvals", Title: "Review pending approvals"})
		}
	}

	if len(results) > 40 {
		results = results[:40]
	}
	return results, nil
}
