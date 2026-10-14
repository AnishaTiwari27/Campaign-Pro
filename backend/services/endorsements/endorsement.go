// Package endorsements is the public record: real, named people and the
// brand relationships reported about them in public sources.
//
// It is deliberately a service of its own, with no database and no
// dependency on any other service, because of what it must never become.
// Campaigns and creators hold the agency's own book — spend, reach,
// pacing, flags, approvals — and every one of those figures is fixture
// data invented for this demo. Attaching an invented figure to a real,
// identifiable person would fabricate a commercial relationship, and
// where the figure is an unflattering one ("under-delivering", "over
// budget") it would fabricate a reputational claim about them too.
//
// So nothing in this package carries a metric, and there is nowhere to
// put one. A Deal records who, which brand, roughly when, and where it
// was reported. That is the whole of what it can hold, by construction
// rather than by convention.
package endorsements

// Photo is a freely-licensed photograph plus everything needed to credit
// it. Every licence behind these (CC BY, CC BY-SA, CC0, GODL-India)
// requires attribution, so the credit travels with the image rather than
// living in a file somewhere that a redesign would quietly drop.
type Photo struct {
	URL        string `json:"url"`
	Artist     string `json:"artist"`
	License    string `json:"license"`
	LicenseURL string `json:"licenseUrl"`
	Source     string `json:"source"`
}

// Deal is one reported brand relationship. Period is free text because
// the sources are: some give a year, some a range, some nothing at all,
// and inventing precision the source does not have is the exact failure
// this package exists to avoid.
type Deal struct {
	Brand  string `json:"brand"`
	Period string `json:"period,omitempty"`
	Note   string `json:"note,omitempty"`
	// Kind separates an endorsement from an ownership stake. Several of
	// these people are investors or founders in brands they also front,
	// and reporting the two as the same thing would be wrong.
	Kind string `json:"kind"`
}

// Person is one public figure and their reported brand work.
type Person struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Initials string `json:"initials"`
	// Field is the arena they are known from, not a category in the
	// agency's taxonomy — these people are not in the agency's book.
	Field  string `json:"field"`
	Photo  Photo  `json:"photo"`
	Deals  []Deal `json:"deals"`
	Source string `json:"source"`
}

const (
	KindEndorsement = "endorsement"
	KindAmbassador  = "ambassador"
	KindStake       = "stake"
)
