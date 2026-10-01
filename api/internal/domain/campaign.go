// Package domain holds DB-independent types and the metric formulas that
// define what this app means by pace, index, benchmark, spotlight, etc.
// Every formula lives here exactly once; web/src/lib/metrics.ts mirrors it
// for the frontend, and both are unit-tested against the same fixtures.
package domain

import "time"

type SubjectType string

const (
	SubjectBrand  SubjectType = "brand"
	SubjectPerson SubjectType = "person"
)

type AdType string

const (
	AdSocial      AdType = "Social"
	AdInfluencer  AdType = "Influencer"
	AdGoogleAds   AdType = "Google Ads"
	AdDisplay     AdType = "Display"
	AdVideo       AdType = "Video"
	AdPerformance AdType = "Performance"
)

type Status string

const (
	StatusLive      Status = "live"
	StatusEnded     Status = "ended"
	StatusScheduled Status = "scheduled"
	StatusPaused    Status = "paused"
)

type Approval string

const (
	ApprovalPending  Approval = "pending"
	ApprovalApproved Approval = "approved"
	ApprovalRejected Approval = "rejected"
)

type CurveShape string

const (
	CurveFast   CurveShape = "fast"
	CurveSteady CurveShape = "steady"
	CurveSlow   CurveShape = "slow"
)

// Campaign is the app's canonical in-memory shape, independent of the
// generated DB structs — service code maps gen.Campaign to this once at
// the store boundary so domain logic never imports pgtype.
type Campaign struct {
	ID          string
	Name        string
	SubjectType SubjectType
	Role        string
	Initials    string
	Category    string
	Region      string
	AdType      AdType
	Platform    string
	Status      Status
	DaysRunning int
	Reach       float64 // lakh (1L = 100,000 people)
	Spend       int64   // whole rupees
	Budget      int64   // whole rupees
	Frequency   float64
	Approval    Approval
	CurveShape  CurveShape
	FlagReason  string
	// CreatorID is set when the subject is a person, linking the campaign
	// to the creator who ran it. Empty for brand campaigns.
	CreatorID string
	// BrandDomain resolves the campaign's brand logo; empty when unknown.
	BrandDomain string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (c Campaign) IsRunning() bool { return c.Status != StatusScheduled }
func (c Campaign) IsFlagged() bool { return c.FlagReason != "" }

type Creative struct {
	ID            string
	CampaignID    string
	Headline      string
	Kind          string
	DurationLabel string
	Reach         float64
	CTR           float64
	// Set by the CreativeAnalyzer; empty until an asset has been analyzed.
	Language   string
	HookType   string
	Claim      string
	Festival   string
	AnalyzedAt time.Time
	CreatedAt  time.Time
}

type AuditEvent struct {
	ID         int64
	CampaignID string
	Actor      string
	Action     string
	Kind       string
	CreatedAt  time.Time
}

type Report struct {
	ID           string
	Name         string
	Enabled      bool
	Cadence      string
	Recipients   []string
	ScopeFilters map[string]any
	ScopeLabel   string
	Columns      []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ReportRun struct {
	ID       int64
	ReportID string
	RanAt    time.Time
	Result   string
	RowCount int
}
