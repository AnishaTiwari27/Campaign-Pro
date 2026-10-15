import { useQuery } from "@tanstack/react-query";
import { get } from "./client";
import { queryKeys } from "./queryKeys";
import type { Insight } from "./types";

/**
 * The explanation for one campaign's signal.
 *
 * Kept out of the campaign detail query on purpose: that query is what
 * paints the page, and this one may wait on a model. The page is
 * useful the moment the figures land; the sentence can arrive after.
 */
export function useInsight(campaignId: string, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.insight(campaignId),
    queryFn: () => get<Insight>(`/insights/campaigns/${campaignId}`),
    enabled: enabled && campaignId !== "",
    staleTime: 10 * 60 * 1000,
    retry: false,
  });
}
