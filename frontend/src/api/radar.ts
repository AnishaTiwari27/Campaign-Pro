import { useQuery } from "@tanstack/react-query";
import { get } from "./client";
import { queryKeys } from "./queryKeys";
import type { Launch } from "./types";

export function useLaunches() {
  return useQuery({
    queryKey: queryKeys.launches(),
    queryFn: () => get<{ items: Launch[] }>("/radar/launches"),
    staleTime: 10 * 60 * 1000,
  });
}
