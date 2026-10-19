package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// MetaConfig is what the Marketing API needs. The access token is a
// long-lived page or system-user token; unlike Google there is no
// refresh exchange, so the token itself is the credential and expires
// on Meta's own schedule.
type MetaConfig struct {
	AccessToken string
	// AdAccountID with or without the act_ prefix; it is added if absent.
	AdAccountID string
	APIVersion  string
	BaseURL     string
	HTTP        *http.Client
}

func (c MetaConfig) complete() bool { return c.AccessToken != "" && c.AdAccountID != "" }

// MetaProvider pulls campaigns from a Meta ad account.
//
// NOT YET RUN AGAINST THE LIVE API, for the same reason as the Google
// client: reaching it needs an app with ads_read granted through app
// review. The request shape, the field list, the pagination and the
// response mapping follow Meta's published reference and the parsing
// is covered by tests against recorded shapes. Treat it as unproven
// until `make verify-ingest SOURCE=meta` has been run.
type MetaProvider struct{ Config MetaConfig }

func (p MetaProvider) Source() string   { return "meta" }
func (p MetaProvider) Configured() bool { return p.Config.complete() }

// apiVersion is pinned rather than tracking "latest", for the same
// reason the Google client pins: a version change can move field
// semantics underneath a sync that nobody is watching.
//
// The flip side is that a pin goes stale on its own, and this one had.
// It read v21.0, which shipped in October 2024 — Meta guarantees a
// version for two years from release, so it was at its expiry. The
// first live call would have failed on the version in the path, which
// is exactly how the Google client's v18 pin was found.
func (p MetaProvider) apiVersion() string {
	if p.Config.APIVersion != "" {
		return p.Config.APIVersion
	}
	// Current as of October 2026. Check Meta's changelog before a
	// first run; the test below only knows what it was told.
	return "v26.0"
}

func (p MetaProvider) baseURL() string {
	if p.Config.BaseURL != "" {
		return p.Config.BaseURL
	}
	return "https://graph.facebook.com"
}

func (p MetaProvider) httpClient() *http.Client {
	if p.Config.HTTP != nil {
		return p.Config.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}

// metaPage is one page of campaigns. Insights arrive nested under each
// campaign as their own single-element list, which is why the struct
// is shaped this oddly — it mirrors what the API returns rather than
// what would be convenient.
type metaPage struct {
	Data []struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Status         string `json:"status"`
		Objective      string `json:"objective"`
		DailyBudget    string `json:"daily_budget"`
		LifetimeBudget string `json:"lifetime_budget"`
		Insights       struct {
			Data []struct {
				Spend       string `json:"spend"`
				Impressions string `json:"impressions"`
				Reach       string `json:"reach"`
			} `json:"data"`
		} `json:"insights"`
	} `json:"data"`
	Paging struct {
		Next string `json:"next"`
	} `json:"paging"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// maxPages bounds pagination. An account with a runaway number of
// campaigns, or a cursor that loops, should stop rather than fetch
// forever and hold the sync open.
const maxPages = 50

func (p MetaProvider) Fetch(ctx context.Context, since time.Time) ([]ExternalCampaign, error) {
	if !p.Configured() {
		return nil, ErrNotConfigured
	}
	if since.IsZero() {
		since = time.Now().AddDate(0, 0, -30)
	}

	account := p.Config.AdAccountID
	if !strings.HasPrefix(account, "act_") {
		account = "act_" + account
	}
	timeRange, err := json.Marshal(map[string]string{
		"since": since.Format("2006-01-02"),
		"until": time.Now().Format("2006-01-02"),
	})
	if err != nil {
		return nil, err
	}

	q := url.Values{
		"access_token": {p.Config.AccessToken},
		"limit":        {"100"},
		// Insights are requested inline so campaigns and their metrics
		// arrive together; fetching them separately would be one call
		// per campaign.
		"fields": {"id,name,status,objective,daily_budget,lifetime_budget," +
			"insights.time_range(" + string(timeRange) + "){spend,impressions,reach}"},
	}
	next := fmt.Sprintf("%s/%s/%s/campaigns?%s", p.baseURL(), p.apiVersion(), account, q.Encode())

	var out []ExternalCampaign
	for page := 0; next != "" && page < maxPages; page++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, next, nil)
		if err != nil {
			return nil, err
		}
		resp, err := p.httpClient().Do(req)
		if err != nil {
			return nil, fmt.Errorf("meta: request failed: %w", err)
		}
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("meta: returned %s: %s", resp.Status, truncate(string(raw), 400))
		}
		batch, nextURL, err := parseMeta(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
		next = nextURL
	}
	return out, nil
}

// parseMeta is separated from the transport so the mapping can be
// proven without a credential.
func parseMeta(raw []byte) ([]ExternalCampaign, string, error) {
	var page metaPage
	if err := json.Unmarshal(raw, &page); err != nil {
		return nil, "", fmt.Errorf("meta: malformed response: %w", err)
	}
	if page.Error != nil {
		return nil, "", fmt.Errorf("meta: %s (%d): %s", page.Error.Type, page.Error.Code, page.Error.Message)
	}

	out := make([]ExternalCampaign, 0, len(page.Data))
	for _, d := range page.Data {
		c := ExternalCampaign{
			ExternalID: d.ID,
			Name:       d.Name,
			Platform:   "Meta",
			Status:     normaliseStatus(d.Status),
			AdType:     normaliseAdType(d.Objective),
			// Meta reports money in the account's minor unit as a
			// string. Lifetime budget wins when both are set, since a
			// campaign with a lifetime cap is bounded by it.
			Budget: minorUnits(firstNonEmpty(d.LifetimeBudget, d.DailyBudget)),
		}
		if len(d.Insights.Data) > 0 {
			in := d.Insights.Data[0]
			c.Spend = int64(parseFloat(in.Spend))
			// Meta reports reach directly — unique people — which is
			// the measure this product stores, so it is used in
			// preference to impressions.
			reach := parseFloat(in.Reach)
			if reach == 0 {
				reach = parseFloat(in.Impressions)
			}
			c.ReachLakh = reach / 100000
		}
		out = append(out, c)
	}
	return out, page.Paging.Next, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" && v != "0" {
			return v
		}
	}
	return ""
}

// minorUnits converts Meta's paise-denominated budget string to rupees.
func minorUnits(s string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return v / 100
}

func parseFloat(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return v
}
