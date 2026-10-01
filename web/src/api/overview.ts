import { useQuery } from "@tanstack/react-query";
import { get } from "./client";
import { queryKeys } from "./queryKeys";
import type { Benchmark, Overview } from "./types";

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
