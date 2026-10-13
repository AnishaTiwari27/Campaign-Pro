import { useQuery } from "@tanstack/react-query";
import { get } from "./client";
import { queryKeys } from "./queryKeys";
import type { CreatorPerformance } from "./types";

/**
 * creatorId -> photograph, for the campaign tiles.
 *
 * A campaign row carries its creator's id but not their picture, and it
 * should not: campaigns owns its own table and does not depend on
 * creators. Resolving the two here keeps that boundary and costs one
 * request per session — the roster is small and never changes mid-visit.
 *
 * Failure is silent on purpose. A missing map means every tile draws its
 * generated avatar, which is exactly what it does today.
 */
export function useCreatorPhotos(): Record<string, string> {
  const { data } = useQuery({
    queryKey: queryKeys.creatorPhotos(),
    queryFn: () => get<{ items: CreatorPerformance[] }>("/creators"),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
  });

  const photos: Record<string, string> = {};
  for (const c of data?.items ?? []) {
    if (c.imageUrl) photos[c.id] = c.imageUrl;
  }
  return photos;
}
