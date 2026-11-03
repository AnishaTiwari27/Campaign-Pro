package creators

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// YouTube is the one platform that will answer "who makes this kind of
// thing", which is the question Instagram refuses at any price.
//
// NOT YET RUN AGAINST THE LIVE API. The request shapes are written
// from Google's published reference for the Data API v3 and covered by
// tests against recorded responses.

var ErrYouTubeNotConfigured = errors.New("youtube is not configured")

// ErrQuotaExhausted is returned before a call that would exceed the
// day's allowance, rather than after Google refuses it. A refused call
// still costs the round trip and tells you nothing about how much was
// left.
var ErrQuotaExhausted = errors.New("youtube: daily quota would be exceeded")

// Quota costs, in the units Google charges.
//
// The 100:1 ratio between them is the whole shape of this client. A
// search costs a hundred times a channel lookup, so the economical
// pattern is few searches feeding batched lookups — and a lookup
// batched fifty at a time costs one unit for fifty channels, where
// fifty separate calls cost fifty.
const (
	QuotaSearch   = 100
	QuotaChannels = 1
	// DailyQuota is Google's default allowance for a new project.
	DailyQuota = 10_000
	// ChannelBatch is the most ids channels.list accepts in one call.
	ChannelBatch = 50
)

// Budget tracks what the day's allowance has left.
//
// Deliberately not thread-safe and deliberately not persistent: it
// bounds one run of the discovery command, which is a single sequential
// pass. A budget shared across processes would need Google's own
// accounting to be trustworthy, and theirs is the one that counts.
type Budget struct {
	Remaining int
}

func NewBudget(units int) *Budget {
	if units <= 0 {
		units = DailyQuota
	}
	return &Budget{Remaining: units}
}

// spend reserves units, refusing rather than going negative.
func (b *Budget) spend(units int) error {
	if b == nil {
		return nil
	}
	if b.Remaining < units {
		return fmt.Errorf("%w: %d units left, %d needed", ErrQuotaExhausted, b.Remaining, units)
	}
	b.Remaining -= units
	return nil
}

// SearchesLeft is how many more searches the budget allows, which is
// the number worth showing an operator: it is the scarce call.
func (b *Budget) SearchesLeft() int {
	if b == nil {
		return 0
	}
	return b.Remaining / QuotaSearch
}

type YouTubeConfig struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
	// Region and Language steer results towards the market we work in.
	// Empty means Google's global default, which for an Indian brief is
	// the wrong default rather than a neutral one.
	Region   string
	Language string
}

type YouTube struct {
	cfg    YouTubeConfig
	Budget *Budget
}

func NewYouTube(cfg YouTubeConfig, budget *Budget) *YouTube {
	return &YouTube{cfg: cfg, Budget: budget}
}

func (y *YouTube) Configured() bool { return y.cfg.APIKey != "" }

func (y *YouTube) baseURL() string {
	if y.cfg.BaseURL != "" {
		return y.cfg.BaseURL
	}
	return "https://www.googleapis.com/youtube/v3"
}

func (y *YouTube) client() *http.Client {
	if y.cfg.HTTP != nil {
		return y.cfg.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

// Channel is one creator as YouTube describes them.
type Channel struct {
	ID          string
	Title       string
	Description string
	// Subscribers is zero and HasSubscribers false where the channel
	// hides its count. Instagram has no equivalent, so this is the
	// first place the roster has to carry "we are not allowed to know"
	// as distinct from "nobody follows them".
	Subscribers    int64
	HasSubscribers bool
	Videos         int64
	Views          int64
}

func (y *YouTube) get(ctx context.Context, path string, q url.Values, out any) error {
	if !y.Configured() {
		return ErrYouTubeNotConfigured
	}
	q.Set("key", y.cfg.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/%s?%s", y.baseURL(), path, q.Encode()), nil)
	if err != nil {
		return err
	}
	resp, err := y.client().Do(req)
	if err != nil {
		return fmt.Errorf("youtube: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		// 403 here is nearly always the quota, and saying so saves an
		// hour of looking for a credentials problem that is not there.
		if resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("youtube: refused (status 403) — usually the daily quota: %s", snippet(body))
		}
		return fmt.Errorf("youtube: status %d: %s", resp.StatusCode, snippet(body))
	}
	return json.Unmarshal(body, out)
}

func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}

type searchResponse struct {
	Items []struct {
		ID struct {
			ChannelID string `json:"channelId"`
		} `json:"id"`
		Snippet struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"snippet"`
	} `json:"items"`
}

// SearchChannels finds channel ids matching a brief.
//
// Returns ids rather than full channels: search results carry a title
// and a blurb but no statistics, and filling them in costs one unit
// for fifty channels through Channels below. Treating the search
// result as the answer would mean either a roster with no figures or
// a per-channel lookup that burns the day's quota.
func (y *YouTube) SearchChannels(ctx context.Context, query string, limit int) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("youtube: empty search")
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	if err := y.Budget.spend(QuotaSearch); err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("part", "snippet")
	q.Set("type", "channel")
	q.Set("q", query)
	q.Set("maxResults", strconv.Itoa(limit))
	if y.cfg.Region != "" {
		q.Set("regionCode", y.cfg.Region)
	}
	if y.cfg.Language != "" {
		q.Set("relevanceLanguage", y.cfg.Language)
	}

	var out searchResponse
	if err := y.get(ctx, "search", q, &out); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	ids := make([]string, 0, len(out.Items))
	for _, it := range out.Items {
		id := it.ID.ChannelID
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}

type channelsResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"snippet"`
		Statistics struct {
			SubscriberCount       string `json:"subscriberCount"`
			HiddenSubscriberCount bool   `json:"hiddenSubscriberCount"`
			VideoCount            string `json:"videoCount"`
			ViewCount             string `json:"viewCount"`
		} `json:"statistics"`
	} `json:"items"`
}

// Channels fills in the figures for up to fifty ids per request.
//
// Batching is not an optimisation here, it is the difference between
// ten thousand channel lookups a day and two hundred: the call costs
// one unit whether it carries one id or fifty.
func (y *YouTube) Channels(ctx context.Context, ids []string) ([]Channel, error) {
	var out []Channel
	for start := 0; start < len(ids); start += ChannelBatch {
		end := start + ChannelBatch
		if end > len(ids) {
			end = len(ids)
		}
		batch, err := y.channelBatch(ctx, ids[start:end])
		if err != nil {
			// What was already fetched is returned with the error: a
			// quota exhausted half way through a run is a stopping
			// point, not a reason to discard the channels we paid for.
			return out, err
		}
		out = append(out, batch...)
	}
	return out, nil
}

func (y *YouTube) channelBatch(ctx context.Context, ids []string) ([]Channel, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	if err := y.Budget.spend(QuotaChannels); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("part", "snippet,statistics")
	q.Set("id", strings.Join(ids, ","))
	q.Set("maxResults", strconv.Itoa(ChannelBatch))

	var resp channelsResponse
	if err := y.get(ctx, "channels", q, &resp); err != nil {
		return nil, err
	}
	out := make([]Channel, 0, len(resp.Items))
	for _, it := range resp.Items {
		c := Channel{
			ID: it.ID, Title: it.Snippet.Title, Description: it.Snippet.Description,
			Videos: atoi(it.Statistics.VideoCount), Views: atoi(it.Statistics.ViewCount),
		}
		// A hidden count is "not allowed to know", which is not zero.
		// Recording it as zero would file a large channel as a nano
		// creator and quietly drag every tier median down with it.
		if !it.Statistics.HiddenSubscriberCount {
			c.Subscribers, c.HasSubscribers = atoi(it.Statistics.SubscriberCount), true
		}
		out = append(out, c)
	}
	return out, nil
}

// atoi reads the strings YouTube sends its counters as. A value that
// does not parse becomes zero, which is why HasSubscribers exists
// rather than callers testing the count.
func atoi(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// Candidate is a channel discovery turned up, before anyone has
// decided whether it belongs on the roster.
//
// Deliberately not a Creator. Every baseline the product computes —
// tier medians, benchmarks, cost indices — is drawn from the roster,
// so an unreviewed search result joining it would move those figures
// using channels nobody has looked at.
type Candidate struct {
	ID          string
	Platform    string
	ExternalID  string
	Handle      string
	Name        string
	Description string
	// HasFollowers is false where the channel hides its subscriber
	// count. The count is then nothing, not zero.
	Followers    int64
	HasFollowers bool
	Videos       int64
	Views        int64
	// FoundFor is the search that turned them up, so a reviewer can
	// see what they were meant to match.
	FoundFor string
	Status   string
}

// CandidateFrom turns a channel into a candidate for one search.
func CandidateFrom(c Channel, foundFor string) Candidate {
	return Candidate{
		Platform: "youtube", ExternalID: c.ID, Name: c.Title,
		Description: c.Description, Followers: c.Subscribers,
		HasFollowers: c.HasSubscribers, Videos: c.Videos, Views: c.Views,
		FoundFor: foundFor,
	}
}
