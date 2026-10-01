import { useQuery } from "@tanstack/react-query";
import { get } from "./client";
import type { CreatorDetail, CreatorPerformance } from "./types";

export function useCreators() {
  return useQuery({
    queryKey: ["creators"],
    queryFn: () => get<{ items: CreatorPerformance[] }>("/creators"),
  });
}

export function useCreatorDetail(id: string | undefined) {
  return useQuery({
    queryKey: ["creators", id],
    queryFn: () => get<CreatorDetail>(`/creators/${id}`),
    enabled: !!id,
  });
}
