package creators

import (
	"sort"
	"time"
)

// Verification computed from a creator's own public posts.
//
// This is the part a vendor is not needed for. Everything here comes
// from figures the platform publishes about a professional account —
// follower count, likes, comments, post type, timestamps — so it is
// free, it is first-hand, and the follower count is more accurate than
// any vendor's estimate of it.
//
// What is deliberately absent: audience demographics and a
// fake-follower percentage. Both need a sample of the creator's
// follower list, which Instagram exposes to nobody at any price.
// Vendors produce them from their own crawled copy of the graph. The
// engagement ratio below is the honest proxy — an account with bought
// followers cannot buy proportionate comments.

// PostStats is one post, with the figures the platform reports.
type PostStats struct {
	Permalink string
	Caption   string
	PostedAt  time.Time
	Likes     int64
	Comments  int64
	// Type is FEED, REELS, STORY or AD as media_product_type reports it.
	Type string
}

// Media product types, as Instagram names them.
const (
	MediaFeed  = "FEED"
	MediaReels = "REELS"
	MediaStory = "STORY"
	MediaAd    = "AD"
)

// MinPostsForRate is how many organic posts an engagement rate needs
// before it means anything. Below this a single viral post or a single
// quiet one swings the figure entirely, and a number that unstable is
// worse than no number — it gets compared against a benchmark as
// though it were evidence.
const MinPostsForRate = 3

// Audit is what one scan of a creator's recent posts establishes.
type Audit struct {
	// Scanned separates "we looked and found nothing" from "we never
	// looked". Every other zero in this struct is meaningless without
	// it.
	Scanned bool
	ScanAt  time.Time

	Posts  int // organic posts seen, excluding ads
	Reels  int
	Images int

	// EngagementRate is (likes + comments) / followers, averaged across
	// organic posts, as a percentage. Zero where it could not be
	// computed — check HasRate rather than testing for zero.
	EngagementRate float64
	HasRate        bool

	// MedianLikes resists the one viral post that drags a mean upward.
	MedianLikes int64

	// PostsPerWeek over the window actually observed, not over a
	// calendar assumption.
	PostsPerWeek float64

	// AdPosts are posts Instagram marks as an ad placement. Counted,
	// but kept out of the engagement figures.
	AdPosts int
}

// AuditPosts computes what one scan establishes about a creator.
//
// Ads are excluded from every engagement figure. A post with paid reach
// behind it draws likes from people who were shown it rather than from
// followers who chose to look, so including them inflates the organic
// rate — which is precisely the figure being used to judge whether an
// audience is real.
func AuditPosts(followers int64, posts []PostStats, now time.Time) Audit {
	a := Audit{Scanned: true, ScanAt: now}

	var organic []PostStats
	for _, p := range posts {
		if p.Type == MediaAd {
			a.AdPosts++
			continue
		}
		organic = append(organic, p)
		switch p.Type {
		case MediaReels:
			a.Reels++
		default:
			a.Images++
		}
	}
	a.Posts = len(organic)
	if a.Posts == 0 {
		return a
	}

	likes := make([]int64, 0, len(organic))
	var interactions int64
	for _, p := range organic {
		likes = append(likes, p.Likes)
		interactions += p.Likes + p.Comments
	}
	sort.Slice(likes, func(i, j int) bool { return likes[i] < likes[j] })
	a.MedianLikes = median(likes)

	// A rate needs a follower base to divide by and enough posts to be
	// stable. Either missing and the field stays unset rather than
	// holding a figure that looks computed.
	if followers > 0 && a.Posts >= MinPostsForRate {
		a.EngagementRate = float64(interactions) / float64(a.Posts) / float64(followers) * 100
		a.HasRate = true
	}

	a.PostsPerWeek = postsPerWeek(organic, now)
	return a
}

func median(sorted []int64) int64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// postsPerWeek measures over the span actually seen, from the oldest
// post to now.
//
// Not over the calendar: business discovery returns a fixed number of
// recent posts, so a prolific creator's window is days and a quiet
// one's is months. Dividing by a fixed period would report the
// prolific creator as less active, which is backwards.
func postsPerWeek(posts []PostStats, now time.Time) float64 {
	oldest := time.Time{}
	for _, p := range posts {
		if p.PostedAt.IsZero() {
			continue
		}
		if oldest.IsZero() || p.PostedAt.Before(oldest) {
			oldest = p.PostedAt
		}
	}
	if oldest.IsZero() {
		return 0
	}
	weeks := now.Sub(oldest).Hours() / 24 / 7
	// Everything posted inside a day gives a span near zero, which
	// would divide into a meaningless spike.
	if weeks < 1 {
		return 0
	}
	return float64(len(posts)) / weeks
}

// EngagementVerdict compares a creator's rate against the median for
// their tier.
//
// This is the whole fake-follower proxy, and it is phrased as a
// comparison rather than an accusation on purpose. "Engagement 0.4%
// against a tier median of 1.8%" is a measurement we can stand behind.
// "42% of their followers are fake" is a claim about a named person
// that no free data supports and that we would have to defend.
type EngagementVerdict struct {
	Rate       float64
	TierMedian float64
	// Ratio is the creator's rate over the tier median. Below 1 is
	// under-performing for their size.
	Ratio   float64
	Known   bool
	Comment string
}

func CompareEngagement(a Audit, tierMedian float64) EngagementVerdict {
	v := EngagementVerdict{Rate: a.EngagementRate, TierMedian: tierMedian}
	if !a.HasRate || tierMedian <= 0 {
		v.Comment = "Not enough posts to compare"
		return v
	}
	v.Known = true
	v.Ratio = a.EngagementRate / tierMedian
	switch {
	case v.Ratio < 0.4:
		v.Comment = "Well below the median for their tier"
	case v.Ratio < 0.8:
		v.Comment = "Below the median for their tier"
	case v.Ratio <= 1.25:
		v.Comment = "In line with their tier"
	default:
		v.Comment = "Above the median for their tier"
	}
	return v
}

// Signals is what the platform itself says about a creator, as opposed
// to what our own campaign history says. Kept apart from the campaign
// figures on the roster row because the two have different standing:
// these are first-hand and current, those are ours and historical.
type Signals struct {
	// Scanned is whether this creator's media has ever been read. Every
	// other field here is meaningless without it — a zero brand-post
	// count on an unscanned creator is our backlog, not a fact about
	// them.
	Scanned bool
	ScanAt  time.Time

	BrandPosts     int
	EngagementRate float64
	HasRate        bool
	PostsScanned   int
}

// Stage is the commercial axis: has anyone paid them yet.
func (s Signals) Stage() Stage { return StageFor(s.Scanned, s.BrandPosts) }
