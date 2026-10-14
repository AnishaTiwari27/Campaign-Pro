import { useQuery } from "@tanstack/react-query";
import { get } from "./client";
import { queryKeys } from "./queryKeys";
import type { Benchmark, Overview, BenchmarkTrendPoint } from "./types";

export function useOverview() {
  return useQuery({
    queryKey: queryKeys.overview(),
    queryFn: () => get<Overview>("/overview"),
  });
}

export function useBenchmark() {
  return useQuery({
    queryKey: queryKeys.benchmark(),
    queryFn: () => get<Benchmark>("/benchmark"),
  });
}

// Shares the /overview cache entry — used by the rail's Approvals badge so
// it never issues a second request just to show a count.
export function usePendingCount() {
  return useQuery({
    queryKey: queryKeys.overview(),
    queryFn: () => get<Overview>("/overview"),
    select: (d) => d.pendingCount,
  });
}

/**
 * The benchmark history. Separate from useBenchmark because the two
 * answer different questions and the trend is the expensive one: the
 * page is useful the moment the snapshot lands, and the series can
 * arrive after it.
 */
export function useBenchmarkTrend() {
  return useQuery({
    queryKey: queryKeys.benchmarkTrend(),
    queryFn: () => get<{ items: BenchmarkTrendPoint[] }>("/benchmark/trend"),
    staleTime: 5 * 60 * 1000,
    retry: false,
  });
}
