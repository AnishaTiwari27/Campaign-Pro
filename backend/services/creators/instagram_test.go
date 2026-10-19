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
