package main

// instagramHandles are the accounts the follower sync looks up.
//
// Only the ones that could be established from an independent source
// are here. The rest are deliberately absent, because a guessed
// handle is worse than no handle: it does not merely show a wrong
// number, it attaches another person's audience to a real named
// individual, and that figure then flows into their tier, their index
// and every comparison the product draws against them.
//
// Instagram refuses unauthenticated profile reads — which is the
// whole reason the business discovery endpoint exists — so these
// could not be confirmed by fetching the profiles directly. The sync
// confirms them instead: it compares the name the API returns against
// the creator's and refuses to write a figure that does not match.
//
// Adding one is a two-minute job once the sync is connected: set it,
// run the sync, and the mismatch guard tells you immediately whether
// you had the right account.
var instagramHandles = map[string]string{
	"virat-kohli":      "virat.kohli",
	"bhuvan-bam":       "bhuvan.bam22",
	"prajakta-koli":    "mostlysane",
	"kusha-kapila":     "kushakapila",
	"tamannaah-bhatia": "tamannaahspeaks",

	// Not set, pending confirmation: alia-bhatt, salman-khan,
	// deepika-padukone, ranveer-singh, ms-dhoni. Their seeded
	// follower counts stand until someone establishes the account.
}
