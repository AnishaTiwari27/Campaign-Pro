// Package insights explains the signals the anomaly detector raises.
//
// The detector already says a campaign is a "Reach outlier". It does
// not say against what, by how much, or what the options are, and that
// is the question anyone asks next. This writes those two sentences.
//
// The whole design is shaped by one constraint: everything else in this
// product refuses to invent figures. The public record cites its
// sources, the seeded data is labelled fixture data, and the ingestion
// pipeline will not make up a campaign it has not seen. An explanation
// that quietly rounded a number, or produced a plausible one that was
// not in the database, would undo that — and it would be believed,
// because it would be sitting next to figures that are real.
//
// So the model never sees the database and never supplies a number.
// It is given an Evidence struct built from the campaign's own row,
// and every numeric token in what it writes is checked back against
// that evidence before anyone sees it. If a figure appears that was
// not supplied, the explanation is discarded and a deterministic one
// is shown instead. Degrading to something plainer and true beats
// showing something fluent and wrong.
package insights

// Fact is one figure the model is allowed to mention, with the exact
// string it must use. Pairing the label with the rendered value is what
// makes verification possible: anything numeric in the reply that is
// not one of these Display strings did not come from the data.
type Fact struct {
	Label   string `json:"label"`
	Display string `json:"display"`
}

// Evidence is everything the model is told. It is deliberately a flat
// list of rendered figures rather than the campaign struct: a model
// handed raw numbers will format them itself, and a number it formatted
// is a number it could have altered.
type Evidence struct {
	CampaignID string `json:"campaignId"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	Region     string `json:"region"`
	Status     string `json:"status"`
	FlagReason string `json:"flagReason"`
	Facts      []Fact `json:"facts"`
}

// Explanation is what comes back.
type Explanation struct {
	CampaignID string `json:"campaignId"`
	Text       string `json:"text"`
	// Source is "model" or "fallback". Shown in the UI, because a
	// reader is entitled to know whether a sentence was written or
	// assembled — and because a silent fallback is a bug nobody finds.
	Source string `json:"source"`
	// Facts are the figures the explanation was allowed to draw on,
	// returned so the UI can show the working next to the prose.
	Facts []Fact `json:"facts"`
}

const (
	SourceModel    = "model"
	SourceFallback = "fallback"
)
