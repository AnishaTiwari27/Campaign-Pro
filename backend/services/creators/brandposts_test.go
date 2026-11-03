package creators_test

import (
	"testing"

	"campaigntrackerpro/services/creators"
)

var brands = []string{"nykaa", "boat.lifestyle", "mamaearth.india", "Puma"}

func markers(ds []creators.Detection) []string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.MatchedOn
	}
	return out
}

func TestADisclosedPostIsDetected(t *testing.T) {
	for _, caption := range []string{
		"Loving the new serum #ad",
		"New drop today #sponsored",
		"Trying these out #PaidPartnership",
		"Out now! #collab",
		"This is a paid partnership with a brand I love",
		"Sponsored by someone lovely",
	} {
		if ds := creators.DetectBrandPost(caption, brands); len(ds) == 0 {
			t.Errorf("no detection for %q", caption)
		}
	}
}

// The single most likely way this misreads an ordinary post as an
// advertisement. "#ad" lives inside a lot of innocent words.
func TestAdDoesNotMatchInsideAnotherTag(t *testing.T) {
	for _, caption := range []string{
		"Some #advice for new runners",
		"Weekend #adventure with the dog",
		"She is so #adorable",
		"#adidas haul",    // a brand tag, not a disclosure
		"#roadtrip vibes", // 'ad' inside the middle of a word
		"#gradient nails",
	} {
		if ds := creators.DetectBrandPost(caption, brands); len(ds) > 0 {
			t.Errorf("%q was read as an advertisement, matched %v", caption, markers(ds))
		}
	}
}

func TestAnUndisclosedBrandMentionIsNotADetection(t *testing.T) {
	// Creators mention brands they merely like. Treating that as paid
	// work would invent a commercial relationship.
	if ds := creators.DetectBrandPost("Obsessed with my @nykaa order", brands); len(ds) > 0 {
		t.Errorf("a bare brand mention counted as brand work: %v", markers(ds))
	}
}

func TestADisclosureCarriesTheBrandItNames(t *testing.T) {
	ds := creators.DetectBrandPost("New routine ft. @nykaa #ad", brands)
	if len(ds) == 0 {
		t.Fatal("no detection")
	}
	for _, d := range ds {
		if d.Brand != "nykaa" {
			t.Errorf("detection %q carries brand %q, want nykaa", d.MatchedOn, d.Brand)
		}
	}
}

// A disclosure with nobody tagged is real and common. An empty brand is
// more honest than guessing one out of the caption text.
func TestADisclosureWithNoBrandNamedStillCounts(t *testing.T) {
	ds := creators.DetectBrandPost("Big news coming #ad", brands)
	if len(ds) != 1 {
		t.Fatalf("want one detection, got %v", markers(ds))
	}
	if ds[0].Brand != "" {
		t.Errorf("brand = %q, want empty rather than a guess", ds[0].Brand)
	}
}

func TestUnknownBrandsAreNotResolved(t *testing.T) {
	ds := creators.DetectBrandPost("Thanks @some.random.person #ad", brands)
	if len(ds) != 1 {
		t.Fatalf("want the hashtag alone, got %v", markers(ds))
	}
	if ds[0].Brand != "" {
		t.Errorf("brand = %q — an unknown handle must not become a brand", ds[0].Brand)
	}
}

func TestDetectionIsCaseInsensitive(t *testing.T) {
	for _, caption := range []string{"#AD", "#Ad", "#SPONSORED", "#PaidPartnership"} {
		if ds := creators.DetectBrandPost(caption, brands); len(ds) == 0 {
			t.Errorf("no detection for %q", caption)
		}
	}
	ds := creators.DetectBrandPost("love it @NYKAA #ad", brands)
	if len(ds) == 0 || ds[0].Brand != "nykaa" {
		t.Errorf("a differently-cased handle did not resolve: %v", ds)
	}
}

// A handle at the end of a sentence picks up the full stop.
func TestATrailingFullStopIsNotPartOfTheHandle(t *testing.T) {
	ds := creators.DetectBrandPost("Styled with @nykaa. #ad", brands)
	if len(ds) == 0 || ds[0].Brand != "nykaa" {
		t.Errorf("punctuation broke the handle match: %v", ds)
	}
}

// Handles legitimately contain dots, so the trailing-stop trim must not
// eat one in the middle.
func TestDotsInsideAHandleSurvive(t *testing.T) {
	ds := creators.DetectBrandPost("new pair from @boat.lifestyle #collab", brands)
	if len(ds) == 0 || ds[0].Brand != "boat.lifestyle" {
		t.Errorf("brand = %+v, want boat.lifestyle", ds)
	}
}

func TestTheSameTagTwiceIsOneDetection(t *testing.T) {
	ds := creators.DetectBrandPost("#ad big news #ad again #ad", brands)
	if len(ds) != 1 {
		t.Errorf("want one detection for a repeated tag, got %v", markers(ds))
	}
}

func TestAnEmptyCaptionDetectsNothing(t *testing.T) {
	for _, caption := range []string{"", "   ", "\n\t"} {
		if ds := creators.DetectBrandPost(caption, brands); ds != nil {
			t.Errorf("%q produced %v", caption, markers(ds))
		}
	}
}

// The distinction the whole Stage type exists for: a creator we have
// never scanned must not be reported as having no brand work, because
// that would be our backlog presented as a fact about them.
func TestUnscannedIsNotTheSameAsNone(t *testing.T) {
	if got := creators.StageFor(false, 0); got != creators.StageUnknown {
		t.Errorf("unscanned creator = %q, want unknown", got)
	}
	if got := creators.StageFor(true, 0); got != creators.StageNone {
		t.Errorf("scanned with nothing found = %q, want none", got)
	}
	if got := creators.StageFor(true, 3); got != creators.StageProven {
		t.Errorf("scanned with posts = %q, want proven", got)
	}
	// Even a creator with posts recorded but never marked scanned is
	// unknown: the flag is what says we looked.
	if got := creators.StageFor(false, 3); got != creators.StageUnknown {
		t.Errorf("unscanned with posts = %q, want unknown", got)
	}
}

// Stage is a separate axis from Tier. The labels must not read as a
// claim about the person's career drawn from a partial feed.
func TestStageLabelsDescribeWhatWeSawNotWhoTheyAre(t *testing.T) {
	if got := creators.StageLabel(creators.StageNone); got != "None in recent posts" {
		t.Errorf("none label = %q — it has to say it is about recent posts", got)
	}
	if got := creators.StageLabel(creators.StageUnknown); got != "Not yet checked" {
		t.Errorf("unknown label = %q", got)
	}
	if creators.StageLabel(creators.StageProven) == "" {
		t.Error("proven has no label")
	}
}
