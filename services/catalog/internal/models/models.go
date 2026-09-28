// Package models holds catalog-service's JSON-facing types — the taxonomy
// (categories/regions/ad types) and the two subject kinds (brands and
// people) that campaigns can be about.
package models

// Category is a brand- or person-side grouping. AppliesTo drives which
// dropdown(s) it shows up in on the frontend: "brand", "person", or "both".
type Category struct {
	Name      string `json:"name"`
	AppliesTo string `json:"appliesTo"`
}

// Subject is the flattened shape both Brand and Person collapse into for
// any endpoint that just needs "who is this campaign about" — the same
// shape campaigns-service snapshots onto a campaign row at write time.
type Subject struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // "brand" | "person"
	Category string `json:"category"`
	// Person-only, empty for brands.
	Occupation      string `json:"occupation,omitempty"`
	PrimaryPlatform string `json:"primaryPlatform,omitempty"`
}

// Meta is the /meta response — filter-dropdown vocabulary, split by which
// subject type each category applies to so the frontend's Type filter
// (Brand/Person) can show the right category list.
type Meta struct {
	Categories struct {
		Brand  []string `json:"brand"`
		Person []string `json:"person"`
	} `json:"categories"`
	Regions []string `json:"regions"`
	AdTypes []string `json:"adTypes"`
}
