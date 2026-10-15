package endorsements

// Published figures for the market this product reports on.
//
// Same rule as the rest of this package: every number is one somebody
// published, cited to where it was published, and none of it is
// derived, adjusted or combined. The point of showing them next to the
// agency's own medians is that those medians come from this
// deployment's own book — a few dozen campaigns — and a reader has no
// way to tell whether a category median of 88 lakh is normal without
// something outside the system to hold it against.
//
// Where sources disagree, both figures are shown. The market size is
// the clearest case: the organised-channel figure and the
// total-deployment figure differ by roughly threefold, because most
// Indian influencer spend moves directly between brand and creator
// and never touches a platform that counts it. Picking one and
// presenting it as the number would be choosing a side in a
// disagreement the reader should see.

// MarketFact is one published figure.
type MarketFact struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Period string `json:"period,omitempty"`
	// Note carries the qualification the figure does not survive
	// without — what was counted, or who disagrees.
	Note      string `json:"note,omitempty"`
	Source    string `json:"source"`
	SourceURL string `json:"sourceUrl"`
}

var marketFacts = []MarketFact{
	{
		Label: "Influencer marketing, organised channels", Value: "₹3,000–3,500 crore", Period: "2025",
		Note:      "Spend moving through visible, organised channels — about a quarter of the total.",
		Source:    "Kofluence, via Storyboard18",
		SourceURL: "https://www.storyboard18.com/how-it-works/creator-economy-in-india-hits-rs-3500-crore-fueled-by-ecomm-fmcg-sectors-kofluence-73457.htm",
	},
	{
		Label: "Influencer marketing, total deployment", Value: "₹10,000 crore+", Period: "2025",
		Note:      "The same market counted differently: roughly 75% is transacted directly between brand and creator and never reaches an organised channel.",
		Source:    "exchange4media",
		SourceURL: "https://www.exchange4media.com/influence-zone-news/indias-influencer-marketing-industry-crosses-10000-crore-mark-150072.html",
	},
	{
		Label: "Projected, organised channels", Value: "₹4,500–5,000 crore", Period: "by end 2026",
		Source:    "Kofluence, via Storyboard18",
		SourceURL: "https://www.storyboard18.com/how-it-works/creator-economy-in-india-hits-rs-3500-crore-fueled-by-ecomm-fmcg-sectors-kofluence-73457.htm",
	},
	{
		Label: "Engagement rate, tier-3 and tier-4 cities", Value: "4.5–5.5%", Period: "2025",
		Note:      "Against 3–4% in metros — the gap regional creators are priced on.",
		Source:    "Kofluence, via NetInfluencer",
		SourceURL: "https://www.netinfluencer.com/india-influencer-marketing-industry-estimated-at-400m-as-regional-creators-reshape-brand-strategy/",
	},
	{
		Label: "Engagement rate, metros", Value: "3–4%", Period: "2025",
		Source:    "Kofluence, via NetInfluencer",
		SourceURL: "https://www.netinfluencer.com/india-influencer-marketing-industry-estimated-at-400m-as-regional-creators-reshape-brand-strategy/",
	},
	{
		Label: "Campaign cost, smaller cities", Value: "₹35,000–90,000", Period: "2025",
		Note:      "Per campaign, against ₹3.8–4.5 lakh in metro markets.",
		Source:    "Apparel Resources",
		SourceURL: "https://apparelresources.com/business-news/retail/indias-influencer-marketing-industry-nears-us-500-million-says-report/",
	},
	{
		Label: "Campaign cost, metros", Value: "₹3.8–4.5 lakh", Period: "2025",
		Source:    "Apparel Resources",
		SourceURL: "https://apparelresources.com/business-news/retail/indias-influencer-marketing-industry-nears-us-500-million-says-report/",
	},
	{
		Label: "Creators in the Indian creator economy", Value: "3.5–4.5 million", Period: "2025",
		Source:    "Kofluence, via NetInfluencer",
		SourceURL: "https://www.netinfluencer.com/india-influencer-marketing-industry-estimated-at-400m-as-regional-creators-reshape-brand-strategy/",
	},
}

// Market returns the published figures.
func Market() []MarketFact {
	out := make([]MarketFact, len(marketFacts))
	copy(out, marketFacts)
	return out
}
