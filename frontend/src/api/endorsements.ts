import { useQuery } from "@tanstack/react-query";
import { get } from "./client";
import { queryKeys } from "./queryKeys";
import type { EndorsementPerson } from "./types";

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
