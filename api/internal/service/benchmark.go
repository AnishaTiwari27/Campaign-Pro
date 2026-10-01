package service

import (
	"context"

	"campaigntrackerpro/internal/domain"
)

type BenchmarkResult struct {
	MedAll     float64
	Categories []domain.CategoryBenchmark
}

func (c *Campaigns) Benchmark(ctx context.Context) (BenchmarkResult, error) {
	all, err := c.Store.ListCampaigns(ctx)
	if err != nil {
		return BenchmarkResult{}, err
	}
	running := domain.Running(all)
	medAll := domain.MedianReach(running)
	return BenchmarkResult{MedAll: medAll, Categories: domain.CategoryBenchmarks(running)}, nil
}
