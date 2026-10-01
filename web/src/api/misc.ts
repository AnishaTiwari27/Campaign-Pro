import { useQuery } from "@tanstack/react-query";
import { get } from "./client";
import { queryKeys } from "./queryKeys";
import type { Health, Me, SearchResult } from "./types";

export function useSearch(q: string) {
  return useQuery({
    queryKey: queryKeys.search(q),
    queryFn: () => get<{ items: SearchResult[] }>("/search", { q }),
    placeholderData: (prev) => prev,
  });
}

export function useMe() {
  return useQuery({
    queryKey: queryKeys.me(),
    queryFn: () => get<Me>("/me"),
    staleTime: Infinity,
  });
}

export function useHealth() {
  return useQuery({
    queryKey: queryKeys.health(),
    queryFn: () => get<Health>("/health"),
    refetchInterval: 30_000,
  });
}
