// Package data seeds the in-memory store with the same synthetic dataset the
// React dashboard's mock (CAMPAIGNS/BRANDS in the original frontend) used, so
// the UI renders identically once wired to this API. This is a stand-in for
// the real ingestion pipeline (Phase 3, Google Ads / Meta / LinkedIn feeds) —
// swap GenerateCampaigns for a repository-backed loader once Postgres lands.
package data

import (
	"math"
	"sort"
	"time"

	"campaigntrackerpro/internal/models"
)

var Brands = []models.Brand{
	{Name: "Zepto", Category: "E-commerce"},
	{Name: "Myntra", Category: "Fashion"},
	{Name: "Swiggy", Category: "Food Delivery"},
	{Name: "Tata CLiQ", Category: "E-commerce"},
	{Name: "boAt", Category: "Technology"},
	{Name: "Nykaa", Category: "Fashion"},
	{Name: "CRED", Category: "Fintech"},
	{Name: "Flipkart", Category: "E-commerce"},
	{Name: "Dream11", Category: "Technology"},
	{Name: "Amul", Category: "FMCG"},
	{Name: "Lenskart", Category: "Fashion"},
	{Name: "PhonePe", Category: "Fintech"},
}

var Territories = []string{"Mumbai", "Delhi NCR", "Bengaluru", "South Zone", "West Zone", "Pan-India"}

var AdTypes = []string{"Social Media", "Influencer", "Google Ads", "Display", "Video", "Performance"}

var PlatformForAdType = map[string]string{
	"Social Media": "Instagram",
	"Influencer":   "YouTube Creators",
	"Google Ads":   "Google Search",
	"Display":      "Google Display",
	"Video":        "YouTube",
	"Performance":  "Meta Ads",
}

var AdTypeColor = map[string]string{
	"Social Media": "#C97A1A",
	"Performance":  "#1F6F6F",
	"Video":        "#5A6ACF",
	"Google Ads":   "#B23A2A",
	"Display":      "#8A6D3B",
	"Influencer":   "#3B7A57",
}

// Categories are the distinct brand categories, in first-seen order — same
// derivation as the frontend's `[...new Set(BRANDS.map(b => b.category))]`.
func Categories() []string {
	seen := map[string]bool{}
	var out []string
	for _, b := range Brands {
		if !seen[b.Category] {
			seen[b.Category] = true
			out = append(out, b.Category)
		}
	}
	return out
}

// GenerateCampaigns reproduces the frontend's seedCampaigns() deterministically:
// same index-based pseudo-randomness, so backend and (old) frontend mocks
// agree while the ingestion pipeline isn't live yet.
func GenerateCampaigns(today time.Time) []models.Campaign {
	rows := make([]models.Campaign, 0, 48)
	for i := 0; i < 48; i++ {
		brand := Brands[i%len(Brands)]
		adType := AdTypes[(i*3)%len(AdTypes)]
		territory := Territories[(i*5)%len(Territories)]

		daysAgo := (i * 137) % 150
		start := today.AddDate(0, 0, -daysAgo)
		end := start.AddDate(0, 0, 14+(i%21))

		reach := int64(40000 + (i*91117)%900000)
		factor := 0.8 + float64((i*13)%7)/10.0
		spend := int64(math.Round(float64(reach) / 22.0 * factor))

		status := "Completed"
		if end.After(today) {
			status = "Live"
		}

		rows = append(rows, models.Campaign{
			ID: i + 1, Brand: brand.Name, Category: brand.Category, Territory: territory,
			AdType: adType, Platform: PlatformForAdType[adType],
			Start: start, End: end, Reach: reach, Spend: spend, Status: status,
		})
	}
	sort.SliceStable(rows, func(a, b int) bool { return rows[a].Start.After(rows[b].Start) })
	return rows
}
