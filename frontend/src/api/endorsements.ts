import { useQuery } from "@tanstack/react-query";
import { get } from "./client";
import { queryKeys } from "./queryKeys";
import type { EndorsementPerson, MarketFact } from "./types";

// Published reference data: it does not change between visits and is the
// same for every account, so it is cached for the session.
export function useEndorsements() {
  return useQuery({
    queryKey: queryKeys.endorsements(),
    queryFn: () => get<{ items: EndorsementPerson[] }>("/endorsements"),
    staleTime: Infinity,
  });
}

export function useEndorsementPerson(id: string) {
  return useQuery({
    queryKey: queryKeys.endorsementPerson(id),
    queryFn: () => get<EndorsementPerson>(`/endorsements/${id}`),
    staleTime: Infinity,
    enabled: id !== "",
  });
}

/**
 * Published figures about the market itself.
 *
 * Separate from the agency's own benchmarks on purpose: these come
 * from outside the system and are what its internal medians have to be
 * judged against. Cached for the session — they change when somebody
 * publishes a new report, not when a campaign does.
 */
export function useMarketFacts() {
  return useQuery({
    queryKey: queryKeys.marketFacts(),
    queryFn: () => get<{ items: MarketFact[] }>("/endorsements/market"),
    staleTime: Infinity,
    retry: false,
  });
}
