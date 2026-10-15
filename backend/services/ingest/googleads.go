package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"campaigntrackerpro/platform/oauth"
)

// ErrNotConfigured is returned by a provider whose credentials are
// absent. Distinct from a transport failure on purpose: "you have not
// set this up" and "the platform is down" need different responses,
// and a sync that reports the second when it means the first sends
// people to look at the wrong thing.
var ErrNotConfigured = errors.New("provider is not configured")

// GoogleAdsConfig is what the Google Ads API needs before it will
// answer. The OAuth client and refresh token belong to a specific
// Google Cloud project, and it is that project — not a token — which
// now carries the API access level.
type GoogleAdsConfig struct {
	// DeveloperToken is optional and kept only for continuity.
	//
	// Google sunset developer tokens on 9 September 2026: access
	// levels (Test, Explorer, Basic, Standard) attach to the Google
	// Cloud project that issued the OAuth credentials, and a token
	// sent in a request is ignored. It is not required here, and must
	// not be, because a setup starting today cannot obtain one —
	// requiring it would leave this provider permanently
	// unconfigurable.
	DeveloperToken string
	ClientID       string
	ClientSecret   string
	RefreshToken   string
	CustomerID     string
	// LoginCustomerID is the manager account, needed when the customer
	// is managed by an MCC rather than being the authorised account.
	LoginCustomerID string
	// APIVersion is pinned rather than defaulted to "latest": Google
	// sunsets versions on a schedule, and a sync that silently moves
	// version is one that silently changes field semantics.
	//
	// The flip side is that a pin goes stale on its own. This one did:
	// it was v18, which Google no longer lists as supported, so the
	// first live call would have failed on the version in the path.
	// Check it against Google's deprecation and sunset page before a
	// first run.
	APIVersion string
	BaseURL    string
	HTTP       *http.Client
}

// complete reports whether there is enough to attempt a call. The
// developer token is deliberately absent from this list — see the
// field's comment.
func (c GoogleAdsConfig) complete() bool {
	return c.ClientID != "" && c.ClientSecret != "" &&
		c.RefreshToken != "" && c.CustomerID != ""
}

// GoogleAdsProvider pulls campaigns from a Google Ads account.
//
// NOT YET RUN AGAINST THE LIVE API. The request shape, the GAQL query
// and the response mapping are written from Google's published
// reference, and the parsing is covered by tests against recorded
// response shapes — but nobody has pointed this at a real account,
// because doing so needs an authorised Google Ads account.
// `make verify-ingest SOURCE=google-ads` does exactly that check once
// credentials exist, and until it has been run this should be treated
// as unproven rather than working.
type GoogleAdsProvider struct {
	Config GoogleAdsConfig
	tokens *oauth.RefreshTokenSource
}

func (p GoogleAdsProvider) Source() string   { return "google-ads" }
func (p GoogleAdsProvider) Configured() bool { return p.Config.complete() }

func (p GoogleAdsProvider) apiVersion() string {
	if p.Config.APIVersion != "" {
		return p.Config.APIVersion
	}
	// Supported versions as of October 2026 are v24, v25 and v26.
	return "v26"
}

func (p GoogleAdsProvider) baseURL() string {
	if p.Config.BaseURL != "" {
		return p.Config.BaseURL
	}
	return "https://googleads.googleapis.com"
}

func (p GoogleAdsProvider) httpClient() *http.Client {
	if p.Config.HTTP != nil {
		return p.Config.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}

// gaql asks for the fields this product stores and nothing else.
// Metrics come from the same row as the campaign, so one query covers
// both and there is no second pass to correlate.
const gaqlTemplate = `SELECT
  campaign.id,
  campaign.name,
  campaign.status,
  campaign.advertising_channel_type,
  campaign_budget.amount_micros,
  metrics.impressions,
  metrics.cost_micros,
  metrics.average_cpm,
  segments.date
FROM campaign
WHERE segments.date >= '%s'
ORDER BY campaign.id`

type gaqlRequest struct {
	Query string `json:"query"`
}

// gaqlResponse follows searchStream, which returns a JSON array of
// result chunks rather than one object.
type gaqlChunk struct {
	Results []struct {
		Campaign struct {
			ID                     json.Number `json:"id"`
			Name                   string      `json:"name"`
			Status                 string      `json:"status"`
			AdvertisingChannelType string      `json:"advertisingChannelType"`
		} `json:"campaign"`
		CampaignBudget struct {
			AmountMicros json.Number `json:"amountMicros"`
		} `json:"campaignBudget"`
		Metrics struct {
			Impressions json.Number `json:"impressions"`
			CostMicros  json.Number `json:"costMicros"`
			AverageCpm  json.Number `json:"averageCpm"`
		} `json:"metrics"`
	} `json:"results"`
	Error *struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

func (p *GoogleAdsProvider) tokenSource() *oauth.RefreshTokenSource {
	if p.tokens == nil {
		p.tokens = &oauth.RefreshTokenSource{
			TokenURL:     "https://oauth2.googleapis.com/token",
			ClientID:     p.Config.ClientID,
			ClientSecret: p.Config.ClientSecret,
			RefreshToken: p.Config.RefreshToken,
			HTTP:         p.Config.HTTP,
		}
	}
	return p.tokens
}

func (p GoogleAdsProvider) Fetch(ctx context.Context, since time.Time) ([]ExternalCampaign, error) {
	if !p.Configured() {
		return nil, ErrNotConfigured
	}
	if since.IsZero() {
		since = time.Now().AddDate(0, 0, -30)
	}

	token, err := (&p).tokenSource().Token(ctx)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(gaqlRequest{Query: fmt.Sprintf(gaqlTemplate, since.Format("2006-01-02"))})
	if err != nil {
		return nil, err
	}
	customer := strings.ReplaceAll(p.Config.CustomerID, "-", "")
	url := fmt.Sprintf("%s/%s/customers/%s/googleAds:searchStream", p.baseURL(), p.apiVersion(), customer)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	// Sent only when one was configured. Google ignores it either way,
	// but an empty header is a claim to carry a credential that is not
	// there, and it muddies anything reading the wire.
	if p.Config.DeveloperToken != "" {
		req.Header.Set("developer-token", p.Config.DeveloperToken)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.Config.LoginCustomerID != "" {
		req.Header.Set("login-customer-id", strings.ReplaceAll(p.Config.LoginCustomerID, "-", ""))
	}

	resp, err := p.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("google ads: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google ads: returned %s: %s", resp.Status, truncate(string(raw), 400))
	}
	return parseGoogleAds(raw)
}

// parseGoogleAds is separated from the transport so the mapping can be
// tested against recorded response bodies without a network or a
// credential — which is the only part of this client that can be
// proven before someone has a developer token.
func parseGoogleAds(raw []byte) ([]ExternalCampaign, error) {
	var chunks []gaqlChunk
	if err := json.Unmarshal(raw, &chunks); err != nil {
		// searchStream returns an array; a bare object is how the API
		// reports an error on some paths.
		var single gaqlChunk
		if err2 := json.Unmarshal(raw, &single); err2 != nil {
			return nil, fmt.Errorf("google ads: malformed response: %w", err)
		}
		chunks = []gaqlChunk{single}
	}

	// A campaign appears once per date segment, so rows are folded
	// together: cost and impressions sum, and the budget is the same on
	// every row. Without this a thirty-day window would report thirty
	// copies of each campaign, each with one day's spend.
	byID := map[string]*ExternalCampaign{}
	order := []string{}

	for _, chunk := range chunks {
		if chunk.Error != nil {
			return nil, fmt.Errorf("google ads: %s: %s", chunk.Error.Status, chunk.Error.Message)
		}
		for _, r := range chunk.Results {
			id := r.Campaign.ID.String()
			if id == "" {
				continue
			}
			c, seen := byID[id]
			if !seen {
				c = &ExternalCampaign{
					ExternalID: id,
					Name:       r.Campaign.Name,
					Platform:   "Google",
					Status:     normaliseStatus(r.Campaign.Status),
					AdType:     normaliseAdType(r.Campaign.AdvertisingChannelType),
					Budget:     micros(r.CampaignBudget.AmountMicros),
				}
				byID[id] = c
				order = append(order, id)
			}
			c.Spend += micros(r.Metrics.CostMicros)
			// Impressions are the closest thing Google reports to the
			// reach this product stores. Not the same measure — reach
			// is unique people — but it is what the API gives, and
			// inventing a de-duplication factor to convert it would be
			// making up the number.
			c.ReachLakh += numberOf(r.Metrics.Impressions) / 100000
		}
	}

	out := make([]ExternalCampaign, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out, nil
}

// micros converts Google's micro-units to whole rupees.
func micros(n json.Number) int64 {
	v, err := strconv.ParseInt(n.String(), 10, 64)
	if err != nil {
		return 0
	}
	return v / 1_000_000
}

func numberOf(n json.Number) float64 {
	v, err := strconv.ParseFloat(n.String(), 64)
	if err != nil {
		return 0
	}
	return v
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
