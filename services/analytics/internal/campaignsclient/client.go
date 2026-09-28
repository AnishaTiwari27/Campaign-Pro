// Package campaignsclient is analytics-service's only upstream dependency —
// deliberately not catalog-service (see docs/ARCHITECTURE.md's review notes):
// campaigns-service's rows already carry subject/category/region/ad-type
// names, so there's nothing catalog-service could add for aggregation.
//
// Every method here calls one of campaigns-service's /internal/aggregates/*
// routes — pre-aggregated in SQL, not the full filtered row set. See
// docs/ROADMAP.md's Phase A: pulling every row over HTTP and summing in Go
// memory (what this client used to do, via a since-removed ListAll) was
// the real DB bottleneck at scale.
package campaignsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"campaigntrackerpro/platform"
	"campaigntrackerpro/services/analytics/internal/models"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: platform.NewInternalClient()}
}

func (c *Client) KPIs(ctx context.Context, rawQuery string) (models.KPIRawData, error) {
	var out models.KPIRawData
	err := c.getJSON(ctx, "/internal/aggregates/kpis", rawQuery, &out)
	return out, err
}

func (c *Client) Trend(ctx context.Context, rawQuery string) ([]models.TrendBucket, error) {
	var out []models.TrendBucket
	err := c.getJSON(ctx, "/internal/aggregates/trend", rawQuery, &out)
	return out, err
}

func (c *Client) Regions(ctx context.Context, rawQuery string) ([]models.RegionCount, error) {
	var out []models.RegionCount
	err := c.getJSON(ctx, "/internal/aggregates/regions", rawQuery, &out)
	return out, err
}

// Benchmark forwards rawQuery as-is, same as the other three — sort/dir
// are already part of it (the frontend sends them as ordinary query
// params), so campaigns-service's handler reads them the same way it
// reads category/region/adType.
func (c *Client) Benchmark(ctx context.Context, rawQuery string) ([]models.BenchmarkAgg, error) {
	var out []models.BenchmarkAgg
	err := c.getJSON(ctx, "/internal/aggregates/benchmark", rawQuery, &out)
	return out, err
}

func (c *Client) getJSON(ctx context.Context, path, rawQuery string, dest any) error {
	u := c.baseURL + path
	if rawQuery != "" {
		u += "?" + rawQuery
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("campaigns-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("campaigns-service returned %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}
