package creators_test

import (
	"testing"
	"time"

	"campaigntrackerpro/services/creators"
)

var scanAt = time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

func post(daysAgo int, kind string, likes, comments int64) creators.PostStats {
	return creators.PostStats{
		Permalink: "https://instagram.com/p/x",
		PostedAt:  scanAt.AddDate(0, 0, -daysAgo),
		Type:      kind, Likes: likes, Comments: comments,
	}
}

func TestEngagementRateIsComputedFromRealFigures(t *testing.T) {
	// 100k followers, four posts averaging 5,000 interactions → 5%.
	posts := []creators.PostStats{
		post(7, creators.MediaFeed, 4800, 200),
		post(14, creators.MediaReels, 4700, 300),
		post(21, creators.MediaFeed, 4900, 100),
		post(28, creators.MediaReels, 4600, 400),
	}
	a := creators.AuditPosts(100_000, posts, scanAt)
	if !a.HasRate {
		t.Fatal("four posts and a follower count should give a rate")
	}
	if a.EngagementRate < 4.9 || a.EngagementRate > 5.1 {
		t.Errorf("rate = %.2f%%, want about 5%%", a.EngagementRate)
	}
	if a.Reels != 2 || a.Images != 2 {
		t.Errorf("reels=%d images=%d, want 2 and 2", a.Reels, a.Images)
	}
}

// A post with paid reach draws likes from people who were shown it,
// not from followers who chose to look. Counting it inflates the very
// figure being used to judge whether the audience is real.
func TestAdsAreExcludedFromEngagement(t *testing.T) {
	organic := []creators.PostStats{
		post(7, creators.MediaFeed, 1000, 0),
		post(14, creators.MediaFeed, 1000, 0),
		post(21, creators.MediaFeed, 1000, 0),
	}
	withAd := append([]creators.PostStats{post(3, creators.MediaAd, 900_000, 0)}, organic...)

	base := creators.AuditPosts(100_000, organic, scanAt)
	plus := creators.AuditPosts(100_000, withAd, scanAt)

	if plus.EngagementRate != base.EngagementRate {
		t.Errorf("an ad changed the organic rate: %.3f vs %.3f", plus.EngagementRate, base.EngagementRate)
	}
	if plus.AdPosts != 1 {
		t.Errorf("the ad should still be counted, got %d", plus.AdPosts)
	}
	if plus.Posts != 3 {
		t.Errorf("organic posts = %d, want 3", plus.Posts)
	}
}

// Below the minimum, one viral post swings the figure entirely, and an
// unstable number is worse than none — it gets compared to a benchmark
// as though it were evidence.
func TestTooFewPostsGivesNoRateAtAll(t *testing.T) {
	for n := 0; n < creators.MinPostsForRate; n++ {
		var posts []creators.PostStats
		for i := 0; i < n; i++ {
			posts = append(posts, post(7*(i+1), creators.MediaFeed, 5000, 100))
		}
		a := creators.AuditPosts(100_000, posts, scanAt)
		if a.HasRate {
			t.Errorf("%d posts produced a rate", n)
		}
		if a.EngagementRate != 0 {
			t.Errorf("%d posts left a rate of %.2f behind", n, a.EngagementRate)
		}
	}
}

func TestNoFollowerCountGivesNoRate(t *testing.T) {
	posts := []creators.PostStats{
		post(7, creators.MediaFeed, 100, 10),
		post(14, creators.MediaFeed, 100, 10),
		post(21, creators.MediaFeed, 100, 10),
	}
	if a := creators.AuditPosts(0, posts, scanAt); a.HasRate {
		t.Error("dividing by zero followers produced a rate")
	}
}

// Scanned is what separates "looked and found nothing" from "never
// looked". Every other zero is meaningless without it.
func TestAnEmptyScanIsStillAScan(t *testing.T) {
	a := creators.AuditPosts(100_000, nil, scanAt)
	if !a.Scanned {
		t.Error("a scan that found no posts is still a scan")
	}
	if a.Posts != 0 || a.HasRate {
		t.Errorf("got %+v", a)
	}
}

// One viral post should not drag the typical figure up with it.
func TestMedianLikesResistAViralOutlier(t *testing.T) {
	posts := []creators.PostStats{
		post(7, creators.MediaFeed, 1000, 0),
		post(14, creators.MediaFeed, 1100, 0),
		post(21, creators.MediaFeed, 900, 0),
		post(28, creators.MediaFeed, 5_000_000, 0),
	}
	a := creators.AuditPosts(100_000, posts, scanAt)
	if a.MedianLikes < 900 || a.MedianLikes > 1100 {
		t.Errorf("median likes = %d, want the typical post not the outlier", a.MedianLikes)
	}
}

// Business discovery returns a fixed number of recent posts, so a
// prolific creator's window is days and a quiet one's is months.
// Dividing by a fixed period would report the prolific one as less
// active, which is backwards.
func TestCadenceMeasuresTheWindowActuallySeen(t *testing.T) {
	busy := []creators.PostStats{
		post(7, creators.MediaFeed, 10, 1), post(10, creators.MediaFeed, 10, 1),
		post(12, creators.MediaFeed, 10, 1), post(14, creators.MediaFeed, 10, 1),
	}
	quiet := []creators.PostStats{
		post(30, creators.MediaFeed, 10, 1), post(90, creators.MediaFeed, 10, 1),
		post(150, creators.MediaFeed, 10, 1), post(210, creators.MediaFeed, 10, 1),
	}
	b := creators.AuditPosts(1000, busy, scanAt)
	q := creators.AuditPosts(1000, quiet, scanAt)
	if b.PostsPerWeek <= q.PostsPerWeek {
		t.Errorf("busy %.2f/wk is not above quiet %.2f/wk", b.PostsPerWeek, q.PostsPerWeek)
	}
}

func TestCadenceIsNotReportedForASingleDaysWorth(t *testing.T) {
	// Everything inside a day gives a span near zero, which would
	// divide into a meaningless spike.
	posts := []creators.PostStats{
		post(0, creators.MediaFeed, 10, 1),
		post(0, creators.MediaFeed, 10, 1),
		post(0, creators.MediaFeed, 10, 1),
	}
	if a := creators.AuditPosts(1000, posts, scanAt); a.PostsPerWeek != 0 {
		t.Errorf("posts per week = %.2f, want it withheld", a.PostsPerWeek)
	}
}

func TestPostsWithNoTimestampDoNotBreakCadence(t *testing.T) {
	posts := []creators.PostStats{
		{Type: creators.MediaFeed, Likes: 10},
		{Type: creators.MediaFeed, Likes: 10},
		{Type: creators.MediaFeed, Likes: 10},
	}
	a := creators.AuditPosts(1000, posts, scanAt)
	if a.PostsPerWeek != 0 {
		t.Errorf("undated posts produced a cadence of %.2f", a.PostsPerWeek)
	}
	if !a.HasRate {
		t.Error("a missing timestamp should not cost the engagement rate")
	}
}

// The fake-follower proxy, phrased as a comparison rather than an
// accusation.
func TestEngagementComparesAgainstTheTierMedian(t *testing.T) {
	posts := []creators.PostStats{
		post(7, creators.MediaFeed, 400, 0),
		post(14, creators.MediaFeed, 400, 0),
		post(21, creators.MediaFeed, 400, 0),
	}
	a := creators.AuditPosts(100_000, posts, scanAt) // 0.4%
	v := creators.CompareEngagement(a, 1.8)
	if !v.Known {
		t.Fatal("a computable comparison was withheld")
	}
	if v.Ratio > 0.3 {
		t.Errorf("ratio = %.2f, want well under 1", v.Ratio)
	}
	if v.Comment != "Well below the median for their tier" {
		t.Errorf("comment = %q", v.Comment)
	}
}

func TestComparisonIsWithheldWithoutARate(t *testing.T) {
	a := creators.AuditPosts(100_000, []creators.PostStats{post(7, creators.MediaFeed, 400, 0)}, scanAt)
	v := creators.CompareEngagement(a, 1.8)
	if v.Known {
		t.Error("compared against a median on one post")
	}
	if v.Comment == "" {
		t.Error("a withheld comparison still has to say why")
	}
	// And with no median to compare against.
	full := creators.AuditPosts(100_000, []creators.PostStats{
		post(7, creators.MediaFeed, 400, 0), post(14, creators.MediaFeed, 400, 0),
		post(21, creators.MediaFeed, 400, 0),
	}, scanAt)
	if creators.CompareEngagement(full, 0).Known {
		t.Error("compared against a median of zero")
	}
}

// The wording is the point: a measurement we can stand behind, never a
// claim about the person we would have to defend.
func TestVerdictsNeverAccuseTheCreator(t *testing.T) {
	posts := []creators.PostStats{
		post(7, creators.MediaFeed, 10, 0), post(14, creators.MediaFeed, 10, 0),
		post(21, creators.MediaFeed, 10, 0),
	}
	a := creators.AuditPosts(10_000_000, posts, scanAt)
	for _, median := range []float64{0.5, 1.8, 4.0} {
		c := creators.CompareEngagement(a, median).Comment
		for _, banned := range []string{"fake", "bot", "bought", "fraud", "inauthentic"} {
			if contains(c, banned) {
				t.Errorf("verdict %q accuses the creator", c)
			}
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
