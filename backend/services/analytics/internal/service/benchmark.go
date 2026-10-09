package service

import (
	"campaigntrackerpro/services/campaigns"
	"context"
)

type BenchmarkResult struct {
	MedAll     float64
	Categories []campaigns.CategoryBenchmark
	// Festivals rides along with the category baselines because it is the
	// same question asked of a different grouping, and the page that shows
	// one should not need a second round trip for the other.
	Festivals []campaigns.FestivalStat
}

func (a *Analytics) Benchmark(ctx context.Context) (BenchmarkResult, error) {
	all, err := a.campaigns.All(ctx)
	if err != nil {
		return BenchmarkResult{}, err
	}
	running := campaigns.Running(all)
	medAll := campaigns.MedianReach(running)
	festivals, err := a.campaigns.Festivals(ctx)
	if err != nil {
		return BenchmarkResult{}, err
	}
	return BenchmarkResult{
		MedAll:     medAll,
		Categories: campaigns.CategoryBenchmarks(running),
		Festivals:  festivals,
	}, nil
}
