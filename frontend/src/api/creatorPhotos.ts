import { useQuery } from "@tanstack/react-query";
import { get } from "./client";
import { queryKeys } from "./queryKeys";
import type { CreatorPerformance } from "./types";

/**
 * creatorId -> the creator, for campaign rows and tiles.
 *
 * A campaign row carries its creator's id but not their name or picture,
 * and it should not: campaigns owns its own table and does not depend on
 * creators. Resolving the two here keeps that boundary and costs one
 * request per session — the roster is small and never changes mid-visit.
 *
 * Failure is silent on purpose. An empty map means every tile draws its
 * generated avatar and shows no byline, which is exactly what it did
 * before any of this existed.
 *
 * Scoping comes from the server: a client account's /creators is already
 * filtered to people with at least one campaign they can see, so a byline
 * can never name someone the viewer isn't allowed to know about.
 */
export function useCreatorIndex(): Record<string, CreatorPerformance> {
  const { data } = useQuery({
    queryKey: queryKeys.creatorPhotos(),
    queryFn: () => get<{ items: CreatorPerformance[] }>("/creators"),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
  });

  const byId: Record<string, CreatorPerformance> = {};
  for (const c of data?.items ?? []) byId[c.id] = c;
  return byId;
}

/** creatorId -> photograph. The narrow view, for callers that only paint. */
export function useCreatorPhotos(): Record<string, string> {
  const index = useCreatorIndex();
  const photos: Record<string, string> = {};
  for (const [id, c] of Object.entries(index)) {
    if (c.imageUrl) photos[id] = c.imageUrl;
  }
  return photos;
}
