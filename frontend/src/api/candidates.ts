import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { get, post } from "./client";
import type { Candidate, Creator } from "./types";

export function useCandidates() {
  return useQuery({
    queryKey: ["candidates"],
    queryFn: () => get<{ items: Candidate[] }>("/creators/candidates"),
  });
}

export interface PromoteBody {
  role?: string;
  category: string;
  region: string;
  languages?: string[];
  /** Required where the channel hides its subscriber count. */
  followers?: number;
}

export function usePromoteCandidate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: PromoteBody }) =>
      post<Creator>(`/creators/candidates/${id}/promote`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["candidates"] });
      // The roster gains a row, and every tier median is recomputed
      // from it — so the creators list is stale the moment this lands.
      qc.invalidateQueries({ queryKey: ["creators"] });
    },
  });
}

export function useRejectCandidate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => post<void>(`/creators/candidates/${id}/reject`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["candidates"] }),
  });
}
