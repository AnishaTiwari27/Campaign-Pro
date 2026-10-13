package main

// Realistic ad creatives, written per brand.
//
// Every creative used to be the same four strings — "Brand film — 30s
// cut", "Search ad — brand keywords" — repeated across unrelated brands,
// with the language, hook and festival each picked by a hash. The result
// read as a format list rather than advertising: Dream11 running in
// Telugu, Swiggy in Bengali, Eid attached to campaigns at random.
//
// The copy below is written in each brand's register but is original: a
// brand's registered tagline is their trademark, and a demo is not a
// reason to reproduce one.

type adCreative struct {
	headline string
	kind     string // video | banner | text
	duration string
	hook     string
	ctrMul   float64 // against the campaign's base CTR for that ad type
	reachMul float64
}

// brandCreatives is keyed by campaign id. Two or three per brand, in the
// formats that brand actually buys.
var brandCreatives = map[string][]adCreative{
	"amul": {
		{"The taste that grew up with you", "banner", "static", "story", 0.9, 0.50},
		{"Fresh from the dairy, every morning", "banner", "static", "announcement", 0.7, 0.35},
	},
	"blinkit": {
		{"Groceries before the kettle boils", "text", "article", "offer", 1.2, 0.60},
		{"Forgot the dhania? Ten minutes.", "text", "article", "demo", 1.0, 0.30},
	},
	"cred": {
		{"Your credit card bill, finally worth paying", "text", "static", "offer", 1.1, 0.55},
		{"Rewards that are not a points table", "video", "0:15", "story", 1.3, 0.40},
	},
	"dream11": {
		{"Pick eleven. Watch differently.", "video", "0:30", "story", 1.0, 0.65},
		{"The match is better when it is yours", "video", "0:15", "announcement", 1.2, 0.35},
	},
	"ethos-watches": {
		{"A movement you will hand down", "banner", "static", "story", 0.8, 0.52},
		{"Swiss craft, Delhi showroom", "banner", "static", "announcement", 0.6, 0.33},
	},
	"lenskart": {
		{"Try a hundred frames from your sofa", "video", "0:30", "demo", 1.0, 0.62},
		{"Eye test at home, free", "video", "0:15", "offer", 1.1, 0.38},
	},
	"licious": {
		{"Cut this morning, cooked tonight", "text", "static", "story", 1.0, 0.55},
		{"Never frozen. Ever.", "video", "0:15", "demo", 1.2, 0.40},
	},
	"mamaearth": {
		{"Read the back of the bottle", "text", "static", "demo", 1.0, 0.55},
		{"Gentle enough for the whole family", "video", "0:15", "testimonial", 1.2, 0.40},
	},
	"myntra": {
		{"The wardrobe refresh, sorted", "text", "static", "offer", 1.0, 0.55},
		{"Everything you saved, now in your size", "video", "0:15", "announcement", 1.2, 0.40},
	},
	"nykaa": {
		{"The festive edit is live", "text", "static", "offer", 1.0, 0.55},
		{"Shades that suit Indian skin", "video", "0:15", "demo", 1.2, 0.42},
	},
	"phonepe": {
		{"Scan anything. Pay anywhere.", "text", "article", "demo", 1.0, 0.60},
		{"Works even where the signal does not", "text", "article", "announcement", 0.9, 0.30},
	},
	"rapido": {
		{"Beat the 6pm crawl", "banner", "static", "offer", 0.9, 0.50},
		{"Across town for the price of a chai", "banner", "static", "announcement", 0.8, 0.35},
	},
	"swiggy": {
		{"Dinner decided in ninety seconds", "text", "article", "offer", 1.0, 0.60},
		{"Your usual, one tap away", "text", "article", "demo", 0.9, 0.32},
	},
	"tanishq": {
		{"For the women who built the family", "banner", "static", "story", 0.9, 0.52},
		{"Certified gold, honest weight", "banner", "static", "testimonial", 0.7, 0.34},
	},
	"urban-company": {
		{"Professionals, not strangers", "text", "article", "testimonial", 1.0, 0.58},
		{"Deep clean, booked for Sunday", "text", "article", "offer", 0.9, 0.32},
	},
	"zepto": {
		{"Ten minutes, or it is on us", "text", "article", "offer", 1.1, 0.60},
		{"Midnight cravings, solved", "text", "article", "story", 0.9, 0.33},
	},
	"boat": {
		{"Bass you can feel on the metro", "video", "0:30", "demo", 1.0, 0.65},
		{"Forty hours between charges", "video", "0:15", "announcement", 1.1, 0.35},
	},
}

// creatorEndorsements is the copy for creator-fronted work.
//
// An endorsement is not the brand's own headline read out by someone
// else: it is written as something the person would plausibly say in
// their own feed, which is why these are first-person and the brand
// creatives above are not. Three per brand, in the formats a creator
// deal actually buys — a reel, a short cut, a grid post — so that a
// person with four brands on their profile has four different pieces of
// work rather than the same line four times.
//
// Original copy, for the reason given at the top of this file.
var creatorEndorsements = map[string][]adCreative{
	"Zepto": {
		{"What I actually keep in my fridge", "video", "0:20", "testimonial", 1.1, 0.72},
		{"Ordered mid-recipe, arrived before the oil was hot", "video", "0:15", "demo", 1.2, 0.64},
		{"My 2am list, delivered", "banner", "static", "story", 0.9, 0.55},
	},
	"Nykaa": {
		{"The five things in my travel pouch", "video", "0:30", "demo", 1.1, 0.75},
		{"What I wear when there is no makeup artist", "video", "0:20", "testimonial", 1.2, 0.68},
		{"Shade matching, finally done right", "banner", "static", "demo", 0.9, 0.52},
	},
	"Lenskart": {
		{"Three frames, one face, you decide", "video", "0:20", "demo", 1.2, 0.70},
		{"I stopped squinting at the teleprompter", "video", "0:15", "testimonial", 1.0, 0.62},
		{"Try them on without leaving the sofa", "banner", "static", "offer", 0.9, 0.54},
	},
	"Mamaearth": {
		{"What I use on my daughter's hair", "video", "0:20", "testimonial", 1.2, 0.74},
		{"Gentle enough that I stopped reading labels", "video", "0:15", "story", 1.0, 0.60},
		{"The one bottle that travels with me", "banner", "static", "story", 0.8, 0.50},
	},
	"CRED": {
		{"I pay my bill first now, oddly", "video", "0:20", "story", 1.1, 0.68},
		{"The only app that rewards me for being boring", "video", "0:15", "story", 1.3, 0.72},
		{"My credit score, checked on a Sunday", "banner", "static", "demo", 0.9, 0.48},
	},
	"Dream11": {
		{"My eleven for tonight, and why", "video", "0:30", "story", 1.2, 0.80},
		{"I have never picked the obvious captain", "video", "0:20", "story", 1.1, 0.70},
		{"Team locked. Toss in ten.", "banner", "static", "announcement", 1.0, 0.58},
	},
	"boAt": {
		{"What I train in, every morning", "video", "0:20", "demo", 1.1, 0.70},
		{"I charge them on Sunday and forget", "video", "0:15", "announcement", 1.0, 0.62},
		{"The pair that survived my gym bag", "banner", "static", "testimonial", 0.9, 0.52},
	},
	"Lakmé": {
		{"Backstage, this is all I use", "video", "0:30", "demo", 1.2, 0.76},
		{"The look I do in four minutes", "video", "0:20", "demo", 1.1, 0.66},
		{"Festive, but still my face", "banner", "static", "story", 0.9, 0.54},
	},
	"Myntra": {
		{"Everything in this reel is on sale", "video", "0:20", "offer", 1.3, 0.74},
		{"Five outfits, one suitcase", "video", "0:30", "demo", 1.1, 0.68},
		{"What I actually re-wear", "banner", "static", "story", 0.9, 0.50},
	},
	"PhonePe": {
		{"I paid the sabziwala without a note", "video", "0:15", "demo", 1.1, 0.66},
		{"Split the bill before the plates clear", "video", "0:20", "demo", 1.0, 0.60},
		{"Recharge, bill, done", "banner", "static", "offer", 0.9, 0.46},
	},
	"Tanishq": {
		{"The piece I will not be taking off", "video", "0:30", "story", 1.2, 0.72},
		{"My mother's taste, my size", "video", "0:20", "story", 1.1, 0.64},
		{"Light enough to wear all evening", "banner", "static", "demo", 0.9, 0.50},
	},
	"Urban Company": {
		{"They arrived, and I did nothing", "video", "0:20", "testimonial", 1.1, 0.64},
		{"The deep clean I had been putting off", "video", "0:30", "demo", 1.0, 0.58},
		{"Booked at eleven, done by two", "banner", "static", "offer", 0.9, 0.44},
	},
}

// creatorCreative is the endorsement format: one asset, the person's own
// feed, the brand's product. Picked by campaign id so the same pairing is
// always the same piece of work, and so one brand running with several
// creators does not put an identical line in each of their feeds.
func creatorCreative(brand, campaignID string) adCreative {
	set := creatorEndorsements[brand]
	if len(set) == 0 {
		// A brand with no written set still gets a campaign; it just gets
		// the generic line rather than nothing.
		return adCreative{
			headline: brand + " — sponsored reel",
			kind:     "video",
			duration: "0:20",
			hook:     "testimonial",
			ctrMul:   1.0,
			reachMul: 0.70,
		}
	}
	return set[fnv32(campaignID)%uint32(len(set))]
}

// fnv32 is FNV-1a. Deterministic across machines and runs, which is the
// whole requirement — a reseed must not reshuffle who ran what.
func fnv32(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h = (h ^ uint32(s[i])) * 16777619
	}
	return h
}

// regionLanguages is what a campaign in that market would actually run in.
// A Pan-India buy runs in the two link languages; a zone buy adds the
// language spoken there. Random assignment put Dream11's Pan-India film in
// Telugu and Swiggy's Bengaluru buy in Bengali.
var regionLanguages = map[string][]string{
	"Pan-India":  {"Hindi", "English"},
	"Delhi NCR":  {"Hindi", "English"},
	"Mumbai":     {"Marathi", "Hindi", "English"},
	"Bengaluru":  {"Kannada", "English"},
	"South Zone": {"Tamil", "Telugu", "English"},
	"West Zone":  {"Marathi", "Gujarati", "English"},
}

func languageFor(region string, n int) string {
	langs, ok := regionLanguages[region]
	if !ok || len(langs) == 0 {
		return "English"
	}
	return langs[n%len(langs)]
}

// categoryFestivals is which tentpole a category actually plans around.
// Jewellery and beauty buy Diwali; sport buys the IPL; an FMCG brand runs
// through the calendar. A category with no entry runs evergreen.
var categoryFestivals = map[string][]string{
	"Luxury":     {"Diwali", "Raksha Bandhan"},
	"Beauty":     {"Diwali", "Navratri"},
	"Fashion":    {"Diwali", "Eid"},
	"Sports":     {"IPL"},
	"FMCG":       {"Diwali", "Holi", "Pongal"},
	"E-commerce": {"Diwali", "Independence Day"},
	"Fintech":    {"Diwali"},
}

// festivalFor tags a creative when its category plans around one, and
// leaves the rest evergreen — which is most of them, as it should be.
func festivalFor(campaignID, category string, n int) string {
	fests, ok := categoryFestivals[category]
	if !ok {
		return ""
	}
	h := (fnv32(campaignID) ^ uint32(n)) * 16777619
	// Rather more than half of a festive-category brand's work is still
	// evergreen, so only some creatives carry a tentpole.
	if h%10 < 4 {
		return ""
	}
	return fests[int(h/10)%len(fests)]
}

// creativesForBrand returns a brand's own creatives, falling back to a
// format-shaped default for any campaign without written copy. Reach and
// CTR stay derived from the campaign so the numbers keep their shape.
func creativesForBrand(campaignID, adType string, reach float64) []creativeSpec {
	base := baseCTR(adType)
	ads, ok := brandCreatives[campaignID]
	if !ok {
		return creativesForFormat(adType, reach)
	}
	out := make([]creativeSpec, 0, len(ads))
	for _, a := range ads {
		out = append(out, creativeSpec{
			headline: a.headline, kind: a.kind, duration: a.duration,
			reach: reach * a.reachMul, ctr: base * a.ctrMul, hook: a.hook,
		})
	}
	return out
}

func baseCTR(adType string) float64 {
	switch adType {
	case "Video":
		return 2.1
	case "Display":
		return 0.8
	case "Social":
		return 1.6
	case "Influencer":
		return 2.6
	case "Google Ads", "Performance":
		return 3.1
	default:
		return 1.0
	}
}

// creativesForFormat is the fallback for a campaign with no written copy:
// shaped by its ad type, and honest about being generic.
func creativesForFormat(adType string, reach float64) []creativeSpec {
	base := baseCTR(adType)
	switch adType {
	case "Video":
		return []creativeSpec{{"Brand film", "video", "0:30", reach * 0.65, base, "story"}}
	case "Display":
		return []creativeSpec{{"Hero banner", "banner", "static", reach * 0.5, base, "announcement"}}
	case "Social":
		return []creativeSpec{{"Feed post", "text", "static", reach * 0.55, base, "announcement"}}
	case "Influencer":
		return []creativeSpec{{"Sponsored reel", "video", "0:20", reach * 0.7, base, "testimonial"}}
	default:
		return []creativeSpec{{"Search ad", "text", "article", reach * 0.6, base, "offer"}}
	}
}
