// Package models holds campaigns-service's JSON-facing Campaign type.
//
// Field naming note: the original monolith called this field "brand" —
// renamed to "subject" here since a campaign can now be about a brand *or*
// a person. This is a deliberate breaking change to the API contract,
// paired with a frontend update in the same pass (not an oversight).
package models

import "time"

type Campaign struct {
	ID          int64     `json:"id"`
	Subject     string    `json:"subject"`
	SubjectType string    `json:"subjectType"` // "brand" | "person"
	Category    string    `json:"category"`
	Region      string    `json:"region"`
	AdType      string    `json:"adType"`
	Platform    string    `json:"platform"`
	Start       time.Time `json:"-"`
	End         time.Time `json:"-"`
	Reach       int64     `json:"reach"`
	Spend       int64     `json:"spend"` // rupees
	Budget      *int64    `json:"-"`     // rupees, nil = no budget set
	Status      string    `json:"status"`
	Pacing      string    `json:"-"` // "" | "under" | "on" | "over" — derived, see PacingOf
	// ApprovalStatus is "pending" | "approved" | "rejected" — a review
	// checkpoint, not a visibility gate (a pending campaign still shows up
	// everywhere). Stored, unlike Status/Pacing, because it's a decision a
	// person makes, not something derivable from the row's own dates —
	// see ApprovalStatusValid and handleUpdateApprovalStatus.
	ApprovalStatus string `json:"-"`
}

const dateLayout = "2006-01-02"

// campaignJSON mirrors Campaign but with dates formatted as YYYY-MM-DD —
// same reasoning as the original monolith's models.go.
type campaignJSON struct {
	ID          int64  `json:"id"`
	Subject     string `json:"subject"`
	SubjectType string `json:"subjectType"`
	Category    string `json:"category"`
	Region      string `json:"region"`
	AdType      string `json:"adType"`
	Platform    string `json:"platform"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Reach       int64  `json:"reach"`
	Spend       int64  `json:"spend"`
	Budget      *int64 `json:"budget,omitempty"`
	Status      string `json:"status"`
	Pacing      string `json:"pacing,omitempty"`
	// No omitempty — unlike Budget/Pacing, every campaign always has a
	// real approval status; there's no "not applicable" case to hide.
	ApprovalStatus string `json:"approvalStatus"`
}

func (c Campaign) AsJSON() any {
	return campaignJSON{
		ID: c.ID, Subject: c.Subject, SubjectType: c.SubjectType, Category: c.Category,
		Region: c.Region, AdType: c.AdType, Platform: c.Platform,
		Start: c.Start.Format(dateLayout), End: c.End.Format(dateLayout),
		Reach: c.Reach, Spend: c.Spend, Budget: c.Budget, Status: c.Status, Pacing: c.Pacing,
		ApprovalStatus: c.ApprovalStatus,
	}
}

// CreateRequest is the POST /api/v1/campaigns body.
type CreateRequest struct {
	Subject     string `json:"subject"`
	SubjectType string `json:"subjectType"`
	Region      string `json:"region"`
	AdType      string `json:"adType"`
	Platform    string `json:"platform"`
	Start       string `json:"start"` // YYYY-MM-DD
	End         string `json:"end"`
	Reach       int64  `json:"reach"`
	Spend       int64  `json:"spend"`
	Budget      *int64 `json:"budget,omitempty"` // rupees; omitted/null = no budget set
	Source      string `json:"source,omitempty"`
}

// UpdateBudgetRequest is the PATCH /api/v1/campaigns/{id} body — the only
// mutable field on an existing campaign right now. There's no campaign
// creation UI yet (see docs/ROADMAP.md), so this is the only way an admin
// assigns or changes a budget through the product, not just via
// seed/curl.
type UpdateBudgetRequest struct {
	Budget *int64 `json:"budget"` // rupees; null clears it
}

// UpdateApprovalStatusRequest is the PATCH /api/v1/campaigns/{id}/approval
// body — a separate route from the budget PATCH above (not folded into
// the same one), so a request that only touches one field can never
// silently reset the other: there's no ambiguity to resolve about
// "absent" vs. "explicitly cleared" when each field has its own route.
type UpdateApprovalStatusRequest struct {
	Status string `json:"status"`
}

// ApprovalStatusValid reports whether status is one of the three states
// approval_status's CHECK constraint allows — checked here too so a bad
// value is a clean 400 from the handler, not a raw DB constraint-violation
// error surfacing as a 500.
func ApprovalStatusValid(status string) bool {
	switch status {
	case "pending", "approved", "rejected":
		return true
	default:
		return false
	}
}

// Pacing thresholds: more than 15% ahead of the expected spend-to-date is
// "over," more than 15% behind is "under," otherwise "on." Arbitrary but
// documented — tightening/loosening this is a one-line change, not a
// redesign.
const (
	pacingOverThreshold  = 1.15
	pacingUnderThreshold = 0.85
)

// PacingOf reports how a campaign's actual spend compares to where it
// should be, given how much of its flight (start->end) has elapsed — ""
// if no budget is set (nothing to pace against). Derived at read time,
// never stored, same as Status ("Live"/"Completed" — see statusOf in
// store/postgres.go) so a background job never has to keep it in sync.
func PacingOf(budget *int64, spend int64, start, end, today time.Time) string {
	if budget == nil || *budget <= 0 {
		return ""
	}

	totalDays := end.Sub(start).Hours() / 24
	if totalDays <= 0 {
		// A same-day (or malformed) flight has no meaningful "elapsed
		// fraction" — the whole budget is already "expected" from day one.
		totalDays = 1
	}
	elapsedFraction := today.Sub(start).Hours() / 24 / totalDays
	if elapsedFraction < 0 {
		elapsedFraction = 0 // hasn't started yet
	}
	if elapsedFraction > 1 {
		elapsedFraction = 1 // flight's over — compare final spend to the full budget
	}

	expectedSpend := float64(*budget) * elapsedFraction
	if expectedSpend <= 0 {
		// Nothing was expected yet (campaign starts today or later) — any
		// spend at all reads as ahead of schedule, none at all is on track.
		if spend > 0 {
			return "over"
		}
		return "on"
	}

	switch ratio := float64(spend) / expectedSpend; {
	case ratio > pacingOverThreshold:
		return "over"
	case ratio < pacingUnderThreshold:
		return "under"
	default:
		return "on"
	}
}
