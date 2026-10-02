import { useQuery } from "@tanstack/react-query";
import { get } from "./client";
import { queryKeys } from "./queryKeys";
import type { Region, RegionDetail } from "./types";

export function useRegions() {
  return useQuery({
    queryKey: queryKeys.regions(),
    queryFn: () => get<{ items: Region[] }>("/regions"),
  });
}

export function useRegionDetail(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.region(id ?? ""),
    queryFn: () => get<RegionDetail>(`/regions/${encodeURIComponent(id ?? "")}`),
    enabled: !!id,
  });
}
