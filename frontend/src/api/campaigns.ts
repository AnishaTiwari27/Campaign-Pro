import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { ApiError, get, post } from "./client";
import { queryKeys } from "./queryKeys";
import type { Campaign, CampaignDetail, AuditEvent, ListParams, ListResponse, Overview } from "./types";
import { toastNegative, toastPositive } from "../app/toastStore";

function paramsToQuery(p: ListParams): Record<string, string | number | undefined> {
  return {
    q: p.q, type: p.type, category: p.category, region: p.region, adType: p.adType,
    status: p.status, approval: p.approval, range: p.range, sort: p.sort, dir: p.dir,
    page: p.page, per: p.per,
  };
}

export function useCampaignsList(params: ListParams) {
  return useQuery({
    queryKey: queryKeys.campaignsList(params),
    queryFn: () => get<ListResponse>("/campaigns", paramsToQuery(params)),
    placeholderData: (prev) => prev,
  });
}

export function useCampaignDetail(id: string | undefined, listParams: ListParams) {
  return useQuery({
    queryKey: queryKeys.campaignDetail(id ?? "", listParams),
    queryFn: () => get<CampaignDetail>(`/campaigns/${id}`, paramsToQuery(listParams)),
    enabled: !!id,
  });
}

export function useAnomalies() {
  return useQuery({
    queryKey: queryKeys.anomalies(),
    queryFn: () => get<{ items: Campaign[] }>("/campaigns/anomalies"),
  });
}

// --- optimistic-update plumbing shared by every campaign-mutating hook ---

function snapshotCampaignCaches(qc: QueryClient) {
  return {
    lists: qc.getQueriesData<ListResponse>({ queryKey: ["campaigns", "list"] }),
    details: qc.getQueriesData<CampaignDetail>({ queryKey: ["campaigns", "detail"] }),
    anomalies: qc.getQueriesData<{ items: Campaign[] }>({ queryKey: ["campaigns", "anomalies"] }),
    overview: qc.getQueriesData<Overview>({ queryKey: queryKeys.overview() }),
  };
}
type CampaignCachesSnapshot = ReturnType<typeof snapshotCampaignCaches>;

function restoreCampaignCaches(qc: QueryClient, snapshot: CampaignCachesSnapshot) {
  snapshot.lists.forEach(([key, data]) => qc.setQueryData(key, data));
  snapshot.details.forEach(([key, data]) => qc.setQueryData(key, data));
  snapshot.anomalies.forEach(([key, data]) => qc.setQueryData(key, data));
  snapshot.overview.forEach(([key, data]) => qc.setQueryData(key, data));
}

function findCampaignInCaches(qc: QueryClient, id: string): Campaign | undefined {
  for (const [, data] of qc.getQueriesData<ListResponse>({ queryKey: ["campaigns", "list"] })) {
    const found = data?.items.find((c) => c.id === id);
    if (found) return found;
  }
  for (const [, data] of qc.getQueriesData<CampaignDetail>({ queryKey: ["campaigns", "detail"] })) {
    if (data && data.id === id) return data;
  }
  return undefined;
}

function patchCampaignInCaches(qc: QueryClient, id: string, patch: Partial<Campaign>) {
  qc.setQueriesData<ListResponse>({ queryKey: ["campaigns", "list"] }, (old) =>
    old ? { ...old, items: old.items.map((c) => (c.id === id ? { ...c, ...patch } : c)) } : old,
  );
  qc.setQueriesData<CampaignDetail>({ queryKey: ["campaigns", "detail"] }, (old) =>
    old && old.id === id ? { ...old, ...patch } : old,
  );
  qc.setQueriesData<{ items: Campaign[] }>({ queryKey: ["campaigns", "anomalies"] }, (old) =>
    old ? { items: old.items.map((c) => (c.id === id ? { ...c, ...patch } : c)) } : old,
  );
  qc.setQueriesData<Overview>({ queryKey: queryKeys.overview() }, (old) => {
    if (!old) return old;
    const mapList = (list: Campaign[]) => list.map((c) => (c.id === id ? { ...c, ...patch } : c));
    return {
      ...old,
      needsDecision: mapList(old.needsDecision),
      flagged: mapList(old.flagged),
      movers: mapList(old.movers),
      people: mapList(old.people),
      spotlight: old.spotlight && old.spotlight.id === id ? { ...old.spotlight, ...patch } : old.spotlight,
    };
  });
}

function invalidateAfterWrite(qc: QueryClient) {
  qc.invalidateQueries({ queryKey: ["campaigns"] });
  qc.invalidateQueries({ queryKey: queryKeys.overview() });
  qc.invalidateQueries({ queryKey: queryKeys.benchmark() });
  qc.invalidateQueries({ queryKey: queryKeys.regions() });
}

// --- mutations ---

// A 409 is the server refusing a decision because of the campaign's own state
// — over budget, most often. That reason is written for the approver and is
// the only useful thing to show, so it replaces the generic failure message.
// Anything else is a genuine fault and keeps the retry wording.
function decisionErrorMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError && err.status === 409) return err.message;
  return fallback;
}

export type DecisionAction = "approve" | "reject" | "reopen";

export function useDecision() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action }: { id: string; action: DecisionAction }) =>
      post<Campaign>(`/campaigns/${id}/decision`, { action }),
    onMutate: async ({ id, action }) => {
      await qc.cancelQueries({ queryKey: ["campaigns"] });
      const snapshot = snapshotCampaignCaches(qc);
      const approval = action === "approve" ? "approved" : action === "reject" ? "rejected" : "pending";
      patchCampaignInCaches(qc, id, { approval });
      return { snapshot };
    },
    onError: (err, _vars, context) => {
      if (context) restoreCampaignCaches(qc, context.snapshot);
      toastNegative(decisionErrorMessage(err, "Couldn't save that decision. Please try again."));
    },
    onSuccess: (_data, { action }) => {
      if (action === "approve") toastPositive("Campaign approved.");
      else if (action === "reject") toastNegative("Campaign rejected.");
      else toastPositive("Reopened for review.");
    },
    onSettled: () => invalidateAfterWrite(qc),
  });
}

export function useBulkDecision() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ ids, action }: { ids: string[]; action: "approve" | "reject" }) =>
      post<{ items: Campaign[] }>("/campaigns/bulk-decision", { ids, action }),
    onMutate: async ({ ids, action }) => {
      await qc.cancelQueries({ queryKey: ["campaigns"] });
      const snapshot = snapshotCampaignCaches(qc);
      const approval = action === "approve" ? "approved" : "rejected";
      ids.forEach((id) => patchCampaignInCaches(qc, id, { approval }));
      return { snapshot };
    },
    onError: (err, _vars, context) => {
      if (context) restoreCampaignCaches(qc, context.snapshot);
      toastNegative(decisionErrorMessage(err, "Couldn't save those decisions."));
    },
    onSuccess: (_data, { ids, action }) => {
      if (action === "approve") toastPositive(`${ids.length} campaign${ids.length === 1 ? "" : "s"} approved.`);
      else toastNegative(`${ids.length} campaign${ids.length === 1 ? "" : "s"} rejected.`);
    },
    onSettled: () => invalidateAfterWrite(qc),
  });
}

export function usePause() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => post<Campaign>(`/campaigns/${id}/pause`),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: ["campaigns"] });
      const snapshot = snapshotCampaignCaches(qc);
      const current = findCampaignInCaches(qc, id);
      if (current && (current.status === "live" || current.status === "paused")) {
        patchCampaignInCaches(qc, id, { status: current.status === "live" ? "paused" : "live" });
      }
      return { snapshot };
    },
    onError: (_err, _id, context) => {
      if (context) restoreCampaignCaches(qc, context.snapshot);
      toastNegative("Couldn't update that campaign's status.");
    },
    onSuccess: (data) => {
      if (data.status === "paused") toastNegative(`${data.name} paused.`);
      else toastPositive(`${data.name} resumed.`);
    },
    onSettled: () => invalidateAfterWrite(qc),
  });
}

export function useAddNote(campaignId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (text: string) => post<AuditEvent>(`/campaigns/${campaignId}/notes`, { text }),
    onMutate: async (text) => {
      await qc.cancelQueries({ queryKey: ["campaigns", "detail", campaignId] });
      const snapshot = qc.getQueriesData<CampaignDetail>({ queryKey: ["campaigns", "detail", campaignId] });
      const optimisticEvent: AuditEvent = {
        id: -Date.now(), actor: "You", action: text, kind: "user", createdAt: new Date().toISOString(),
      };
      qc.setQueriesData<CampaignDetail>({ queryKey: ["campaigns", "detail", campaignId] }, (old) =>
        old ? { ...old, audit: [optimisticEvent, ...old.audit] } : old,
      );
      return { snapshot };
    },
    onError: (_err, _text, context) => {
      context?.snapshot.forEach(([key, data]) => qc.setQueryData(key, data));
      toastNegative("Couldn't save that note.");
    },
    onSuccess: () => toastPositive("Note added."),
    onSettled: () => qc.invalidateQueries({ queryKey: ["campaigns", "detail", campaignId] }),
  });
}
