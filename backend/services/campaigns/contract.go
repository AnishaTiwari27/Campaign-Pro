// Public types that cross service boundaries. Analytics, creators and
// reports all read campaigns, so the shapes they receive are declared here
// in the contract rather than inside internal/, which they cannot import.
package campaigns

import "time"

// CampaignRow is a campaign plus every value derived from it that a list or
// detail view needs to render without recomputing formulas itself.
type CampaignRow struct {
	Campaign
	Pace          float64
	PaceClass     PaceClass
	CPM           float64
	Index         float64
	CategoryIndex float64
	// ExpectedPace is the budget share the plan implies by now, and
	// PaceVsPlan is actual pace against it. Both are 0 when the campaign
	// has no recorded flight length, which the UI shows as "—": raw pace
	// cannot distinguish mid-flight from underspending on its own.
	ExpectedPace float64
	PaceVsPlan   float64
}

// ListParams mirrors every GET /campaigns query parameter.
type ListParams struct {
	Search      string
	SubjectType string // "" | brand | person
	Category    string
	Region      string
	AdType      string
	Status      string
	Approval    string
	Range       string // "" | "7" | "30" (All time = no filter)
	Sort        string
	Dir         string // asc | desc
	Page        int
	Per         int
}

type ListResult struct {
	Items []CampaignRow
	Total int
	Page  int
	Pages int
}

// User is the acting account; auth loads one onto every request.
type User struct {
	ID         string
	Email      string
	Name       string
	Role       string
	CanApprove bool
	CreatedAt  time.Time
}

// FestivalStat is analysed-creative performance grouped by the festival a
// creative was cut for. Festival is "" for evergreen work, which is the
// baseline the tagged groups are measured against — the question is not
// "what did Diwali do", it is "what did Diwali do that ordinary work did
// not".
type FestivalStat struct {
	Festival  string
	Creatives int
	AvgReach  float64
	AvgCTR    float64
	// Lift is AvgReach against the evergreen baseline. 1.0 means a
	// festival cut performed exactly like untagged work; 0 on the
	// baseline row itself, which has nothing to compare with.
	Lift float64
}
