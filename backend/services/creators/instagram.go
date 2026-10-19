package creators

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrInstagramNotConfigured is returned when no token is set.
// Distinct from a transport failure: "you have not set this up" and
// "Instagram is down" need different responses.
var ErrInstagramNotConfigured = errors.New("instagram is not configured")

// InstagramConfig is what the Graph API needs before it will answer.
//
// Business discovery is queried *through* an account you control, so
// both of these are yours: the token authorises you, and the account
// id is the account doing the asking. The creators being looked up
// are not involved and grant nothing — only their public professional
// profile is returned.
type InstagramConfig struct {
	AccessToken string
	// BusinessAccountID is your own Instagram professional account,
	// the one the query is made through.
	BusinessAccountID string
	APIVersion        string
	BaseURL           string
	HTTP              *http.Client
}

func (c InstagramConfig) complete() bool {
	return c.AccessToken != "" && c.BusinessAccountID != ""
}

// Instagram reads public profile figures for professional accounts.
//
// NOT YET RUN AGAINST THE LIVE API. The request shape is written from
// Meta's published reference for business_discovery and covered by
// tests against recorded responses, but nobody has pointed it at a
// real token. Treat it as unproven until it has returned a real
// follower count.
type Instagram struct{ cfg InstagramConfig }

func NewInstagram(cfg InstagramConfig) *Instagram { return &Instagram{cfg: cfg} }

func (i *Instagram) Configured() bool { return i.cfg.complete() }

func (i *Instagram) apiVersion() string {
	if i.cfg.APIVersion != "" {
		return i.cfg.APIVersion
	}
	// Pinned, like the ad platform clients, and stale-prone for the
	// same reason. Current as of October 2026.
	return "v26.0"
}

func (i *Instagram) baseURL() string {
	if i.cfg.BaseURL != "" {
		return i.cfg.BaseURL
	}
	return "https://graph.facebook.com"
}

func (i *Instagram) client() *http.Client {
	if i.cfg.HTTP != nil {
		return i.cfg.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

// Profile is what business discovery returns about one account.
type Profile struct {
	Username   string
	Name       string
	Followers  int64
	MediaCount int64
}

type discoveryResponse struct {
	BusinessDiscovery struct {
		Username       string `json:"username"`
		Name           string `json:"name"`
		FollowersCount int64  `json:"followers_count"`
		MediaCount     int64  `json:"media_count"`
	} `json:"business_discovery"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// Lookup reads one professional account's public figures.
//
// Only professional accounts are visible this way, and only public
// ones: a personal or private account returns an error rather than
// data, which is the platform drawing the line in the right place.
func (i *Instagram) Lookup(ctx context.Context, handle string) (Profile, error) {
	if !i.cfg.complete() {
		return Profile{}, ErrInstagramNotConfigured
	}
	handle = strings.TrimPrefix(strings.TrimSpace(handle), "@")
	if handle == "" {
		return Profile{}, fmt.Errorf("no handle to look up")
	}

	q := url.Values{}
	// The nested-field syntax business discovery takes. Only the
	// profile counters: this is not a reason to pull someone's media.
	q.Set("fields", fmt.Sprintf(
		"business_discovery.username(%s){username,name,followers_count,media_count}", handle))
	q.Set("access_token", i.cfg.AccessToken)

	endpoint := fmt.Sprintf("%s/%s/%s?%s",
		i.baseURL(), i.apiVersion(), i.cfg.BusinessAccountID, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Profile{}, err
	}
	resp, err := i.client().Do(req)
	if err != nil {
		return Profile{}, fmt.Errorf("instagram: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Profile{}, err
	}
	var out discoveryResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return Profile{}, fmt.Errorf("instagram: malformed response: %w", err)
	}
	if out.Error != nil {
		return Profile{}, fmt.Errorf("instagram: %s (code %d)", out.Error.Message, out.Error.Code)
	}
	d := out.BusinessDiscovery
	if d.Username == "" {
		// A 200 with nothing in it. Usually the account is personal,
		// private or age-gated — none of which business discovery
		// returns, and none of which is an error worth retrying.
		return Profile{}, fmt.Errorf("instagram: no public professional account for %q", handle)
	}
	return Profile{
		Username:   d.Username,
		Name:       d.Name,
		Followers:  d.FollowersCount,
		MediaCount: d.MediaCount,
	}, nil
}

// SamePerson reports whether a looked-up profile plausibly belongs to
// the creator it was recorded against.
//
// A handle is typed in by hand, and a wrong one is not a harmless
// mistake: it attaches somebody else's audience to a real named
// person, and the figure then flows into their tier, their index and
// every comparison drawn against them. The check is deliberately
// loose — display names are inconsistent and people change them — so
// it catches a transposition, not a stage name.
func SamePerson(creatorName string, p Profile) bool {
	norm := func(s string) string {
		return strings.Join(strings.Fields(strings.ToLower(s)), " ")
	}
	name, profile, handle := norm(creatorName), norm(p.Name), norm(p.Username)
	if name == "" {
		return false
	}
	if profile != "" && (strings.Contains(profile, name) || strings.Contains(name, profile)) {
		return true
	}
	// Fall back to the handle, with separators removed: "bhuvan.bam22"
	// against "bhuvan bam".
	flat := strings.NewReplacer(".", "", "_", "", " ", "", "-", "").Replace(handle)
	for _, part := range strings.Fields(name) {
		if len(part) > 2 && !strings.Contains(flat, part) {
			return false
		}
	}
	return true
}
