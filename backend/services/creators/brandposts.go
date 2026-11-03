package creators

import (
	"regexp"
	"strings"
	"time"
)

// Brand-post detection: deciding, from a creator's own captions, whether
// they have done paid brand work.
//
// There is no API for this anywhere. Instagram's paid-partnership label
// is not queryable in aggregate, and Meta's Branded Content API only
// covers content where your own brand is the tagged partner — it cannot
// tell you who someone else has worked with. So this reads the captions
// business discovery already returns and looks for the disclosure the
// creator published themselves.
//
// India makes this work better than it would elsewhere: ASCI requires a
// disclosure label on paid promotions, so Indian creators tag more
// reliably than the global average.
//
// What comes out is an inference, and it is stored as evidence rather
// than as a verdict — which post, which marker, which brand — so a
// human can see why and overturn it. A bare "has done brand work: yes"
// against a real named person is a commercial claim we could not
// substantiate.

// Detection is one reason to believe a post is paid brand work.
type Detection struct {
	// Brand is empty where a disclosure fired but named nobody. That is
	// a real and common case — "#ad" with no tag — and an empty brand
	// is more honest than guessing one from the caption.
	Brand string
	// MatchedOn is the literal token that fired, kept so the evidence
	// can be shown rather than asserted.
	MatchedOn string
	// Method is hashtag | mention | phrase.
	Method string
}

// disclosureTags are the hashtags that announce paid work.
//
// Matched as whole tags, never as substrings — see hashtag below. The
// list is where to add regional-language disclosures as they turn up in
// real captions; guessing at them here would only add noise.
var disclosureTags = map[string]bool{
	"ad": true, "ads": true, "advertisement": true, "sponsored": true,
	"sponsoredpost": true, "paidpartnership": true, "paidpromotion": true,
	"paidad": true, "collab": true, "collaboration": true,
	"brandpartner": true, "brandambassador": true, "ambassador": true,
	"gifted": true, "partnership": true,
}

// disclosurePhrases are multi-word forms. Substring matching is safe
// here in a way it is not for "#ad": these are long enough that an
// accidental match is not a realistic worry.
var disclosurePhrases = []string{
	"paid partnership",
	"paid promotion",
	"in collaboration with",
	"in partnership with",
	"brand ambassador for",
	"sponsored by",
}

var (
	// Unicode letters, so a Devanagari or Tamil tag is read as one tag
	// rather than split into pieces.
	hashtag = regexp.MustCompile(`#([\p{L}\p{N}_]+)`)
	// Instagram handles: letters, digits, dots, underscores. A trailing
	// dot is punctuation rather than part of the handle.
	mention = regexp.MustCompile(`@([A-Za-z0-9_][A-Za-z0-9_.]*)`)
)

// DetectBrandPost reads one caption and reports every reason to think it
// is paid brand work. An empty slice means no disclosure was found,
// which is not the same as "not an ad" — an undisclosed promotion is
// invisible here, and that is a limit of the method, not a verdict.
//
// brandHandles are the accounts worth recognising, lowercased by the
// caller or not — matching is case-insensitive either way.
func DetectBrandPost(caption string, brandHandles []string) []Detection {
	if strings.TrimSpace(caption) == "" {
		return nil
	}
	lower := strings.ToLower(caption)

	known := make(map[string]bool, len(brandHandles))
	for _, b := range brandHandles {
		if b = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(b), "@")); b != "" {
			known[b] = true
		}
	}

	// The brand, if one is named, is attached to every detection in the
	// caption — a post that says "#ad" and tags @nykaa is one piece of
	// brand work, not an anonymous disclosure plus a separate mention.
	var brand string
	var mentions []Detection
	for _, m := range mention.FindAllStringSubmatch(caption, -1) {
		h := strings.ToLower(strings.TrimRight(m[1], "."))
		if known[h] {
			if brand == "" {
				brand = h
			}
			mentions = append(mentions, Detection{Brand: h, MatchedOn: "@" + h, Method: "mention"})
		}
	}

	var out []Detection
	// Whole-tag comparison. A substring test would match "#ad" inside
	// "#advice", "#adventure" and "#adorable", which is the single most
	// likely way this misreads an ordinary post as an advertisement.
	seen := map[string]bool{}
	for _, m := range hashtag.FindAllStringSubmatch(caption, -1) {
		tag := strings.ToLower(m[1])
		if disclosureTags[tag] && !seen[tag] {
			seen[tag] = true
			out = append(out, Detection{Brand: brand, MatchedOn: "#" + tag, Method: "hashtag"})
		}
	}
	for _, p := range disclosurePhrases {
		if strings.Contains(lower, p) {
			out = append(out, Detection{Brand: brand, MatchedOn: p, Method: "phrase"})
		}
	}
	// A tagged brand alone is not a disclosure: creators mention brands
	// they merely like. It counts only alongside one.
	if len(out) > 0 {
		out = append(out, mentions...)
	}
	return out
}

// Stage is how far along a creator is in doing paid brand work. It is a
// separate axis from Tier: tier is how big the audience is, stage is
// whether anyone has paid them yet, and a nano creator can be proven
// while a macro one is new to it.
type Stage string

const (
	// StageUnknown is a creator whose media we have never read. The
	// distinction from StageNone is the one that matters: conflating
	// them would label every creator "new" before the sync has run,
	// which is a claim about them drawn from our own backlog.
	StageUnknown Stage = "unknown"
	// StageNone is a creator whose recent media we have read and found
	// no disclosure in. Business discovery returns recent posts rather
	// than a full history, so this means "none we can see", never
	// "none ever".
	StageNone Stage = "none"
	// StageProven has at least one disclosed brand post.
	StageProven Stage = "proven"
)

// StageFor classifies a creator. scanned is whether their media has
// ever been read; posts is how many disclosed brand posts were found.
func StageFor(scanned bool, posts int) Stage {
	switch {
	case !scanned:
		return StageUnknown
	case posts > 0:
		return StageProven
	default:
		return StageNone
	}
}

// StageLabel is the wording shown to a person. None is phrased as what
// we observed rather than as what the creator is: "new creator" would
// be a statement about their career drawn from a partial feed.
func StageLabel(s Stage) string {
	switch s {
	case StageProven:
		return "Brand work seen"
	case StageNone:
		return "None in recent posts"
	default:
		return "Not yet checked"
	}
}

// BrandPostEvidence is one detection, with the post it came from. The
// detection alone is an assertion; with the permalink it is something
// a person can open and overturn.
type BrandPostEvidence struct {
	Detection
	Permalink string
	PostedAt  time.Time
}

// EvidenceFrom pairs each detection in a post with that post.
func EvidenceFrom(p Post, brandHandles []string) []BrandPostEvidence {
	var out []BrandPostEvidence
	for _, d := range DetectBrandPost(p.Caption, brandHandles) {
		out = append(out, BrandPostEvidence{Detection: d, Permalink: p.Permalink, PostedAt: p.PostedAt})
	}
	return out
}

// FollowerPoint is one day in a creator's growth series.
type FollowerPoint struct {
	On             time.Time
	Followers      int64
	EngagementRate float64
	HasRate        bool
	Posts          int
}
