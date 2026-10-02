package service

import (
	"campaigntrackerpro/services/campaigns"
	"context"
)

type BenchmarkResult struct {
	MedAll     float64
	Categories []campaigns.CategoryBenchmark
}

func (a *Analytics) Benchmark(ctx context.Context) (BenchmarkResult, error) {
	all, err := a.campaigns.All(ctx)
	if err != nil {
		return BenchmarkResult{}, err
	}
	running := campaigns.Running(all)
	medAll := campaigns.MedianReach(running)
	return BenchmarkResult{MedAll: medAll, Categories: campaigns.CategoryBenchmarks(running)}, nil
}
