package radar

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HackerNews reads Show HN, which is where a lot of products are
// announced on the day they ship.
//
// It needs no credential, which is why it is the source that works out
// of the box. It is also the wrong shape for this product's market:
// Show HN skews to developer tools and global SaaS, not the Indian
// consumer brands an influencer agency pitches. It is here because a
// pipeline with one real source beats a pipeline with none, and
// because it proves the path end to end — the sources that fit the
// market (Product Hunt, Crunchbase, the Indian startup press) all want
// a key or an agreement.
type HackerNews struct {
	HTTP    *http.Client
	BaseURL string
}

func (h HackerNews) Name() string { return "hacker-news" }

func (h HackerNews) client() *http.Client {
	if h.HTTP != nil {
		return h.HTTP
	}
	return &http.Client{Timeout: 25 * time.Second}
}

func (h HackerNews) baseURL() string {
	if h.BaseURL != "" {
		return h.BaseURL
	}
	return "https://hn.algolia.com"
}

type hnResponse struct {
	Hits []struct {
		ObjectID    string `json:"objectID"`
		Title       string `json:"title"`
		URL         string `json:"url"`
		Points      int32  `json:"points"`
		NumComments int32  `json:"num_comments"`
		CreatedAt   string `json:"created_at"`
	} `json:"hits"`
}

func (h HackerNews) Fetch(since time.Time) ([]Launch, error) {
	if since.IsZero() {
		since = time.Now().AddDate(0, 0, -30)
	}
	q := url.Values{
		"tags":                 {"show_hn"},
		"hitsPerPage":          {"100"},
		"numericFilters":       {fmt.Sprintf("created_at_i>%d", since.Unix())},
		"attributesToRetrieve": {"objectID,title,url,points,num_comments,created_at"},
	}
	endpoint := h.baseURL() + "/api/v1/search_by_date?" + q.Encode()

	resp, err := h.client().Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("hacker news: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hacker news: returned %s", resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	return parseHN(raw)
}

func parseHN(raw []byte) ([]Launch, error) {
	var out hnResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("hacker news: malformed response: %w", err)
	}
	launches := make([]Launch, 0, len(out.Hits))
	for _, hit := range out.Hits {
		launchedAt, err := time.Parse(time.RFC3339, hit.CreatedAt)
		if err != nil {
			// Without a date there is no age, and age is the signal the
			// banding leans on hardest. Skipped rather than banded on
			// nothing.
			continue
		}
		name, tagline := splitShowHN(hit.Title)
		if name == "" {
			continue
		}
		l := Launch{
			Source: "hacker-news", ExternalID: hit.ObjectID,
			Name: name, Tagline: tagline, URL: hit.URL,
			LaunchedAt: launchedAt,
			// Points and comments together: a post with attention from
			// either is one somebody noticed.
			Traction: hit.Points + hit.NumComments,
			HasSite:  strings.HasPrefix(hit.URL, "http"),
		}
		l.Readiness, l.Basis = Assess(l, time.Now())
		launches = append(launches, l)
	}
	return launches, nil
}

// splitShowHN separates the product from its description.
// "Show HN: Theorem – An offline PDF reader" is a name and a tagline,
// and storing the whole string as the brand name would make every
// entry start with "Show HN:".
func splitShowHN(title string) (name, tagline string) {
	t := strings.TrimSpace(title)
	for _, prefix := range []string{"Show HN:", "Show HN –", "Show HN -"} {
		if strings.HasPrefix(t, prefix) {
			t = strings.TrimSpace(strings.TrimPrefix(t, prefix))
			break
		}
	}
	// The separator is an en dash in most posts and a hyphen in the
	// rest; a colon shows up occasionally.
	for _, sep := range []string{" – ", " — ", " - ", ": "} {
		if i := strings.Index(t, sep); i > 0 {
			return strings.TrimSpace(t[:i]), strings.TrimSpace(t[i+len(sep):])
		}
	}
	return t, ""
}
