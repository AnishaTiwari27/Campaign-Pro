import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { get, patch, post } from "./client";
import { queryKeys } from "./queryKeys";
import type { Cadence, Report, ReportRun } from "./types";
import { toastNegative, toastPositive } from "../app/toastStore";

// Toggles a report's enabled state from the list screen — a standalone
// mutation (not the id-bound useUpdateReport) since a hook can't be called
// once per row inside a .map().
export function useToggleReportEnabled() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => patch<Report>(`/reports/${id}`, { enabled }),
    onMutate: async ({ id, enabled }) => {
      await qc.cancelQueries({ queryKey: queryKeys.reports() });
      const previous = qc.getQueryData<{ items: Report[] }>(queryKeys.reports());
      qc.setQueryData<{ items: Report[] }>(queryKeys.reports(), (old) =>
        old ? { items: old.items.map((r) => (r.id === id ? { ...r, enabled } : r)) } : old,
      );
      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) qc.setQueryData(queryKeys.reports(), context.previous);
      toastNegative("Couldn't update that report.");
    },
    onSettled: () => qc.invalidateQueries({ queryKey: queryKeys.reports() }),
  });
}

export function useReports() {
  return useQuery({
    queryKey: queryKeys.reports(),
    queryFn: () => get<{ items: Report[] }>("/reports"),
  });
}

export function useReport(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.report(id ?? ""),
    queryFn: () => get<Report>(`/reports/${id}`),
    enabled: !!id,
  });
}

export function useReportRuns(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.reportRuns(id ?? ""),
    queryFn: () => get<{ items: ReportRun[] }>(`/reports/${id}/runs`),
    enabled: !!id,
  });
}

export interface CreateReportInput {
  name: string;
  cadence: Cadence;
  recipients: string[];
  scopeFilters: Record<string, unknown>;
  scopeLabel: string;
  columns?: string[];
}

export function useCreateReport() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateReportInput) => post<Report>("/reports", input),
    onSuccess: () => {
      toastPositive("Report created.");
      qc.invalidateQueries({ queryKey: queryKeys.reports() });
    },
    onError: () => toastNegative("Couldn't create that report."),
  });
}

export interface UpdateReportInput {
  name?: string;
  enabled?: boolean;
  cadence?: Cadence;
  recipients?: string[];
  columns?: string[];
}

// Report detail "autosaves": every field edit calls this, optimistically
// patching the cached report and rolling back with a toast on failure.
export function useUpdateReport(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateReportInput) => patch<Report>(`/reports/${id}`, input),
    onMutate: async (input) => {
      await qc.cancelQueries({ queryKey: queryKeys.report(id) });
      const previous = qc.getQueryData<Report>(queryKeys.report(id));
      qc.setQueryData<Report>(queryKeys.report(id), (old) => (old ? { ...old, ...input } : old));
      return { previous };
    },
    onError: (_err, _input, context) => {
      if (context?.previous) qc.setQueryData(queryKeys.report(id), context.previous);
      toastNegative("Couldn't save that change.");
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: queryKeys.report(id) });
      qc.invalidateQueries({ queryKey: queryKeys.reports() });
    },
  });
}

export function useRunReport(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => post<ReportRun>(`/reports/${id}/run`),
    onSuccess: (run) => {
      if (run.result === "Delivered") toastPositive("Report sent.");
      else if (run.result === "Not sending") toastNegative("Report has no recipients — nothing was sent.");
      else toastNegative("Report failed to send.");
      qc.invalidateQueries({ queryKey: queryKeys.reportRuns(id) });
      qc.invalidateQueries({ queryKey: queryKeys.report(id) });
    },
    onError: () => toastNegative("Couldn't run that report."),
  });
}

export function useTestReport(id: string) {
  return useMutation({
    mutationFn: () => post<{ sent: boolean; to: string }>(`/reports/${id}/test`),
    onSuccess: (res) => toastPositive(`Test sent to ${res.to}.`),
    onError: () => toastNegative("Couldn't send the test."),
  });
}
