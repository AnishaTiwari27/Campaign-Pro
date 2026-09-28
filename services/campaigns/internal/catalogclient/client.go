// Package catalogclient is campaigns-service's only way of talking to
// catalog-service — over HTTP, never SQL (campaigns_service has no grant on
// the catalog schema; see db/init/00_roles.sql). Used once per campaign
// create, to validate the subject exists and snapshot its category.
package catalogclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"campaigntrackerpro/platform"
)

type Subject struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Category string `json:"category"`
}

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: platform.NewInternalClient()}
}

// Lookup returns (subject, true, nil) if it exists, (_, false, nil) if it
// doesn't, and a non-nil error only for a genuine transport/upstream
// failure — callers should treat "not found" and "catalog unreachable" as
// distinct outcomes.
func (c *Client) Lookup(ctx context.Context, subjectType, name string) (Subject, bool, error) {
	u := fmt.Sprintf("%s/internal/subjects/lookup?type=%s&name=%s",
		c.baseURL, url.QueryEscape(subjectType), url.QueryEscape(name))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Subject{}, false, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Subject{}, false, fmt.Errorf("catalog-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return Subject{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return Subject{}, false, fmt.Errorf("catalog-service returned %d", resp.StatusCode)
	}

	var subj Subject
	if err := json.NewDecoder(resp.Body).Decode(&subj); err != nil {
		return Subject{}, false, err
	}
	return subj, true, nil
}

// ListAll returns every brand and person catalog-service knows about — used
// by the demo ticker to pick a random subject for a synthesized campaign.
func (c *Client) ListAll(ctx context.Context) ([]Subject, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/subjects", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog-service returned %d", resp.StatusCode)
	}
	var subjects []Subject
	if err := json.NewDecoder(resp.Body).Decode(&subjects); err != nil {
		return nil, err
	}
	return subjects, nil
}
