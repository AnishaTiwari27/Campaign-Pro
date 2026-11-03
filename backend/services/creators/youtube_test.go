package creators_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"campaigntrackerpro/services/creators"
)

func ytServer(t *testing.T, handler http.HandlerFunc) *creators.YouTube {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return creators.NewYouTube(creators.YouTubeConfig{
		APIKey: "k", BaseURL: srv.URL, HTTP: srv.Client(),
		Region: "IN", Language: "hi",
	}, creators.NewBudget(creators.DailyQuota))
}

func TestSearchReturnsChannelIDs(t *testing.T) {
	var gotQuery string
	yt := ytServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"items":[
			{"id":{"channelId":"UC1"},"snippet":{"title":"Dance Co"}},
			{"id":{"channelId":"UC2"},"snippet":{"title":"Steps"}},
			{"id":{"channelId":"UC1"},"snippet":{"title":"Dance Co again"}},
			{"id":{},"snippet":{"title":"a video, not a channel"}}
		]}`))
	})

	ids, err := yt.SearchChannels(t.Context(), "dance creators bangalore", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != "UC1" || ids[1] != "UC2" {
		t.Fatalf("got %v, want the two distinct channels", ids)
	}
	// An Indian brief answered with global defaults is the wrong
	// answer, not a neutral one.
	if !strings.Contains(gotQuery, "regionCode=IN") || !strings.Contains(gotQuery, "relevanceLanguage=hi") {
		t.Errorf("query %q does not steer towards the market", gotQuery)
	}
	if !strings.Contains(gotQuery, "type=channel") {
		t.Errorf("query %q did not ask for channels", gotQuery)
	}
}

// The 100:1 cost ratio is the whole shape of the client, so the
// accounting has to be right.
func TestSearchCostsAHundredTimesAChannelLookup(t *testing.T) {
	yt := ytServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"items":[{"id":{"channelId":"UC1"},"snippet":{}}]}`))
	})
	start := yt.Budget.Remaining

	if _, err := yt.SearchChannels(t.Context(), "x", 50); err != nil {
		t.Fatal(err)
	}
	if spent := start - yt.Budget.Remaining; spent != creators.QuotaSearch {
		t.Errorf("a search spent %d units, want %d", spent, creators.QuotaSearch)
	}
}

// Batching is not an optimisation: it is the difference between ten
// thousand lookups a day and two hundred.
func TestFiftyChannelsCostOneUnit(t *testing.T) {
	var calls int
	yt := ytServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Write([]byte(`{"items":[]}`))
	})
	ids := make([]string, creators.ChannelBatch)
	for i := range ids {
		ids[i] = "UC" + string(rune('a'+i%26))
	}
	start := yt.Budget.Remaining
	if _, err := yt.Channels(t.Context(), ids); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("%d channels took %d calls, want 1", len(ids), calls)
	}
	if spent := start - yt.Budget.Remaining; spent != creators.QuotaChannels {
		t.Errorf("spent %d units for one batch, want %d", spent, creators.QuotaChannels)
	}
}

func TestMoreThanABatchSplitsIntoSeveralCalls(t *testing.T) {
	var calls int
	yt := ytServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Write([]byte(`{"items":[]}`))
	})
	ids := make([]string, creators.ChannelBatch+1)
	for i := range ids {
		ids[i] = "UC" + strings.Repeat("x", i%5) + string(rune('a'+i%26))
	}
	if _, err := yt.Channels(t.Context(), ids); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("%d ids took %d calls, want 2", len(ids), calls)
	}
}

// A hidden count is "not allowed to know". Recording it as zero would
// file a large channel as a nano creator and drag the tier median
// down with it.
func TestAHiddenSubscriberCountIsNotZero(t *testing.T) {
	yt := ytServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"items":[
			{"id":"UC1","snippet":{"title":"Open"},"statistics":{"subscriberCount":"125000","videoCount":"310","viewCount":"9000000"}},
			{"id":"UC2","snippet":{"title":"Hidden"},"statistics":{"hiddenSubscriberCount":true,"subscriberCount":"0","videoCount":"88","viewCount":"4000000"}}
		]}`))
	})
	chans, err := yt.Channels(t.Context(), []string{"UC1", "UC2"})
	if err != nil {
		t.Fatal(err)
	}
	if !chans[0].HasSubscribers || chans[0].Subscribers != 125000 {
		t.Errorf("open channel = %+v", chans[0])
	}
	if chans[1].HasSubscribers {
		t.Error("a hidden count was reported as known")
	}
	if chans[1].Subscribers != 0 {
		t.Errorf("hidden count leaked a figure: %d", chans[1].Subscribers)
	}
	// The rest of the statistics are still real and still useful.
	if chans[1].Videos != 88 || chans[1].Views != 4000000 {
		t.Errorf("hidden subscribers cost the other figures: %+v", chans[1])
	}
}

// Refusing before the call rather than after: a refused call still
// costs the round trip and says nothing about what was left.
func TestTheBudgetRefusesBeforeSpending(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Write([]byte(`{"items":[]}`))
	}))
	defer srv.Close()
	// Enough for one search and nothing more.
	yt := creators.NewYouTube(creators.YouTubeConfig{
		APIKey: "k", BaseURL: srv.URL, HTTP: srv.Client(),
	}, creators.NewBudget(creators.QuotaSearch))

	if _, err := yt.SearchChannels(t.Context(), "first", 50); err != nil {
		t.Fatal(err)
	}
	_, err := yt.SearchChannels(t.Context(), "second", 50)
	if !errors.Is(err, creators.ErrQuotaExhausted) {
		t.Fatalf("second search error = %v, want ErrQuotaExhausted", err)
	}
	if calls != 1 {
		t.Errorf("%d calls made; the refused one still went out", calls)
	}
}

func TestSearchesLeftReportsTheScarceCall(t *testing.T) {
	b := creators.NewBudget(creators.DailyQuota)
	if got := b.SearchesLeft(); got != 100 {
		t.Errorf("searches left = %d, want 100", got)
	}
	b.Remaining = 99
	if got := b.SearchesLeft(); got != 0 {
		t.Errorf("searches left = %d with 99 units, want 0", got)
	}
}

// Channels already paid for are returned alongside the error: a quota
// exhausted half way is a stopping point, not a reason to throw away
// what the budget bought.
func TestAnExhaustedBudgetKeepsWhatWasAlreadyFetched(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"items":[{"id":"UC1","snippet":{"title":"One"},"statistics":{"subscriberCount":"10"}}]}`))
	}))
	defer srv.Close()
	yt := creators.NewYouTube(creators.YouTubeConfig{
		APIKey: "k", BaseURL: srv.URL, HTTP: srv.Client(),
	}, creators.NewBudget(1)) // one batch only

	ids := make([]string, creators.ChannelBatch*2)
	for i := range ids {
		ids[i] = "UC" + strings.Repeat("y", i)
	}
	got, err := yt.Channels(t.Context(), ids)
	if !errors.Is(err, creators.ErrQuotaExhausted) {
		t.Fatalf("err = %v, want ErrQuotaExhausted", err)
	}
	if len(got) != 1 {
		t.Errorf("lost the channels the first batch paid for: %v", got)
	}
}

// 403 is nearly always the quota, and saying so saves an hour looking
// for a credentials problem that is not there.
func TestA403NamesTheQuota(t *testing.T) {
	yt := ytServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"message":"quotaExceeded"}}`))
	})
	_, err := yt.SearchChannels(t.Context(), "x", 50)
	if err == nil || !strings.Contains(err.Error(), "quota") {
		t.Fatalf("err = %v, want it to mention the quota", err)
	}
}

func TestNotConfiguredRefusesWithoutCallingOut(t *testing.T) {
	yt := creators.NewYouTube(creators.YouTubeConfig{}, creators.NewBudget(creators.DailyQuota))
	if yt.Configured() {
		t.Fatal("an empty config reported itself configured")
	}
	if _, err := yt.SearchChannels(t.Context(), "x", 50); !errors.Is(err, creators.ErrYouTubeNotConfigured) {
		t.Errorf("err = %v, want ErrYouTubeNotConfigured", err)
	}
}

func TestAnEmptySearchIsRefused(t *testing.T) {
	yt := ytServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("an empty query reached the API")
	})
	start := yt.Budget.Remaining
	if _, err := yt.SearchChannels(t.Context(), "   ", 50); err == nil {
		t.Fatal("an empty search was allowed")
	}
	if yt.Budget.Remaining != start {
		t.Error("a refused search still cost quota")
	}
}
