package creators_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"campaigntrackerpro/services/creators"
)

func TestInstagramLookup(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		w.Write([]byte(`{"business_discovery":{"username":"bhuvan.bam22","name":"Bhuvan Bam",
			"followers_count":17200000,"media_count":412},"id":"me"}`))
	}))
	defer srv.Close()

	ig := creators.NewInstagram(creators.InstagramConfig{
		AccessToken: "tok", BusinessAccountID: "17841400000000000",
		BaseURL: srv.URL, HTTP: srv.Client(),
	})

	p, err := ig.Lookup(t.Context(), "@bhuvan.bam22")
	if err != nil {
		t.Fatal(err)
	}
	if p.Followers != 17200000 || p.Username != "bhuvan.bam22" {
		t.Fatalf("got %+v", p)
	}
	// The query is made through our own account, not the creator's.
	if !strings.Contains(gotPath, "17841400000000000") {
		t.Errorf("path %q does not query through the configured account", gotPath)
	}
	// The @ must be stripped: Instagram's nested-field syntax takes a
	// bare username and returns nothing for "@name".
	if !strings.Contains(gotQuery, "username%28bhuvan.bam22%29") {
		t.Errorf("query %q did not pass a bare handle", gotQuery)
	}
}

// A 200 carrying nothing is the ordinary answer for a personal,
// private or age-gated account. It is not data and must not read as
// zero followers.
func TestInstagramLookupEmptyIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"id":"me"}`))
	}))
	defer srv.Close()

	ig := creators.NewInstagram(creators.InstagramConfig{
		AccessToken: "tok", BusinessAccountID: "acct", BaseURL: srv.URL, HTTP: srv.Client(),
	})
	if _, err := ig.Lookup(t.Context(), "someone"); err == nil {
		t.Error("an empty business_discovery read as success — it would write zero followers")
	}
}

func TestInstagramNotConfigured(t *testing.T) {
	ig := creators.NewInstagram(creators.InstagramConfig{})
	if ig.Configured() {
		t.Error("an empty config reads as configured")
	}
	if _, err := ig.Lookup(t.Context(), "someone"); err != creators.ErrInstagramNotConfigured {
		t.Errorf("got %v, want ErrInstagramNotConfigured", err)
	}
}

// A handle is typed by hand, and the wrong one attaches somebody
// else's audience to a real named person — which then flows into
// their tier, their index and every comparison drawn against them.
func TestSamePerson(t *testing.T) {
	match := []struct{ creator, name, username string }{
		{"Bhuvan Bam", "Bhuvan Bam", "bhuvan.bam22"},
		{"Prajakta Koli", "Prajakta Koli", "mostlysane"},
		// Display name missing: falls back to the handle.
		{"Kusha Kapila", "", "kushakapila"},
		{"Virat Kohli", "Virat Kohli", "virat.kohli"},
		// Extra words either side are fine.
		{"Tamannaah Bhatia", "Tamannaah Bhatia ✨", "tamannaahspeaks"},
	}
	for _, c := range match {
		if !creators.SamePerson(c.creator, creators.Profile{Name: c.name, Username: c.username}) {
			t.Errorf("%s / %s / @%s read as a different person", c.creator, c.name, c.username)
		}
	}

	wrong := []struct{ creator, name, username string }{
		{"Bhuvan Bam", "Carry Minati", "carryminati"},
		{"Prajakta Koli", "Kusha Kapila", "kushakapila"},
		{"Virat Kohli", "", "anushkasharma"},
	}
	for _, c := range wrong {
		if creators.SamePerson(c.creator, creators.Profile{Name: c.name, Username: c.username}) {
			t.Errorf("%s accepted @%s (%s) — that is another person's audience",
				c.creator, c.username, c.name)
		}
	}
}

func TestLookupMediaReturnsCaptionsAndPermalinks(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"business_discovery":{"username":"mostlysane","name":"Prajakta Koli",
			"followers_count":7200000,"media_count":1800,"media":{"data":[
			{"caption":"new routine ft. @nykaa #ad","permalink":"https://instagram.com/p/aaa","timestamp":"2026-03-11T09:30:00+0000"},
			{"caption":"sunday walk","permalink":"https://instagram.com/p/bbb","timestamp":"not a date"},
			{"caption":"no link here"}
			]}},"id":"me"}`))
	}))
	defer srv.Close()

	ig := creators.NewInstagram(creators.InstagramConfig{
		AccessToken: "tok", BusinessAccountID: "17841400000000000",
		BaseURL: srv.URL, HTTP: srv.Client(),
	})

	p, posts, err := ig.LookupMedia(t.Context(), "@mostlysane", 25)
	if err != nil {
		t.Fatal(err)
	}
	// The profile rides along so the identity guard can run on the same
	// response that produced the posts.
	if p.Username != "mostlysane" {
		t.Errorf("profile = %+v", p)
	}
	// Three in the payload, but one has no permalink: a detection
	// nobody can open is not evidence.
	if len(posts) != 2 {
		t.Fatalf("got %d posts, want the 2 with permalinks", len(posts))
	}
	if posts[0].Caption != "new routine ft. @nykaa #ad" {
		t.Errorf("caption = %q", posts[0].Caption)
	}
	if posts[0].PostedAt.IsZero() {
		t.Error("a valid timestamp did not parse")
	}
	// An unparseable date is not worth discarding the post over — the
	// caption is the evidence and the permalink proves its source.
	if !posts[1].PostedAt.IsZero() {
		t.Error("a bad timestamp should leave the zero time, not a guess")
	}
	if !strings.Contains(gotQuery, "media") {
		t.Errorf("query %q did not ask for media", gotQuery)
	}
}

func TestLookupMediaCapsTheRequestedLimit(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"business_discovery":{"username":"x","name":"X"},"id":"me"}`))
	}))
	defer srv.Close()

	ig := creators.NewInstagram(creators.InstagramConfig{
		AccessToken: "tok", BusinessAccountID: "acct",
		BaseURL: srv.URL, HTTP: srv.Client(),
	})
	for _, limit := range []int{0, -3, 9999} {
		if _, _, err := ig.LookupMedia(t.Context(), "x", limit); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(gotQuery, "limit%2825%29") && !strings.Contains(gotQuery, "limit(25)") {
			t.Errorf("limit %d did not clamp to %d: %q", limit, creators.MediaScanLimit, gotQuery)
		}
	}
}

func TestLookupMediaNotConfigured(t *testing.T) {
	ig := creators.NewInstagram(creators.InstagramConfig{})
	if _, _, err := ig.LookupMedia(t.Context(), "x", 25); err == nil {
		t.Fatal("an unconfigured client must refuse rather than call out")
	}
}

func TestLookupMediaCarriesEngagementFigures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"business_discovery":{"username":"mostlysane","name":"Prajakta Koli",
			"followers_count":7200000,"media":{"data":[
			{"caption":"reel","permalink":"https://instagram.com/p/a","like_count":48000,"comments_count":1200,"media_product_type":"REELS"},
			{"caption":"promo","permalink":"https://instagram.com/p/b","like_count":9,"comments_count":1,"media_product_type":"AD"},
			{"caption":"old one","permalink":"https://instagram.com/p/c","like_count":500,"comments_count":10}
			]}},"id":"me"}`))
	}))
	defer srv.Close()

	ig := creators.NewInstagram(creators.InstagramConfig{
		AccessToken: "tok", BusinessAccountID: "acct",
		BaseURL: srv.URL, HTTP: srv.Client(),
	})
	_, posts, err := ig.LookupMedia(t.Context(), "mostlysane", 25)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 3 {
		t.Fatalf("got %d posts", len(posts))
	}
	if posts[0].Likes != 48000 || posts[0].Comments != 1200 {
		t.Errorf("engagement figures lost: %+v", posts[0])
	}
	if posts[0].Type != creators.MediaReels || posts[1].Type != creators.MediaAd {
		t.Errorf("product types lost: %q %q", posts[0].Type, posts[1].Type)
	}
	// Older media can arrive without the field. Dropping it out of the
	// organic figures would be worse than assuming a feed post.
	if posts[2].Type != creators.MediaFeed {
		t.Errorf("a missing product type became %q, want FEED", posts[2].Type)
	}
}
