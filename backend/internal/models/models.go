// Package models holds the domain types shared across the store and API
// layers. Field names/JSON tags mirror the shape the React dashboard already
// expects (see the CAMPAIGNS mock in the frontend) so the client needs no
// remapping once it's pointed at this API.
package models

import "time"

// Campaign is one tracked paid-ad campaign.
type Campaign struct {
	ID        int       `json:"id"`
	Brand     string    `json:"brand"`
	Category  string    `json:"category"`
	Territory string    `json:"territory"`
	AdType    string    `json:"adType"`
	Platform  string    `json:"platform"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Reach     int64     `json:"reach"`
	Spend     int64     `json:"spend"` // rupees
	Status    string    `json:"status"`
}

// MarshalJSON dates as YYYY-MM-DD (the API contract's date format) rather
// than full RFC3339 timestamps — campaigns don't carry a time-of-day.
type campaignJSON struct {
	ID        int    `json:"id"`
	Brand     string `json:"brand"`
	Category  string `json:"category"`
	Territory string `json:"territory"`
	AdType    string `json:"adType"`
	Platform  string `json:"platform"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Reach     int64  `json:"reach"`
	Spend     int64  `json:"spend"`
	Status    string `json:"status"`
}

const dateLayout = "2006-01-02"

func (c Campaign) AsJSON() any {
	return campaignJSON{
		ID: c.ID, Brand: c.Brand, Category: c.Category, Territory: c.Territory,
		AdType: c.AdType, Platform: c.Platform,
		Start: c.Start.Format(dateLayout), End: c.End.Format(dateLayout),
		Reach: c.Reach, Spend: c.Spend, Status: c.Status,
	}
}

// Brand is a tracked advertiser and its industry category.
type Brand struct {
	Name     string
	Category string
}

// KPI is one ticker tile: value + trend + an 8-week sparkline.
type KPI struct {
	Value int64   `json:"value"`
	Delta string  `json:"delta"`
	Up    bool    `json:"up"`
	Spark []int64 `json:"spark"`
}

// KPISet is the full /kpis response.
type KPISet struct {
	ActiveCampaigns KPI `json:"activeCampaigns"`
	BrandsTracked   KPI `json:"brandsTracked"`
	EstimatedReach  KPI `json:"estimatedReach"`
	EstimatedSpend  KPI `json:"estimatedSpend"`
}

// TerritoryCount is one row of the /territories breakdown.
type TerritoryCount struct {
	Territory string `json:"territory"`
	Campaigns int    `json:"campaigns"`
}

// BenchmarkRow is one row of the /benchmark table.
type BenchmarkRow struct {
	Brand     string   `json:"brand"`
	Category  string   `json:"category"`
	Count     int      `json:"count"`
	Reach     int64    `json:"reach"`
	Platforms []string `json:"platforms"`
	Last      string   `json:"last"` // YYYY-MM-DD
}

// Meta is the static filter-dropdown vocabulary.
type Meta struct {
	Categories  []string `json:"categories"`
	Territories []string `json:"territories"`
	AdTypes     []string `json:"adTypes"`
}

// APIError is the JSON body returned on every non-2xx response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}
