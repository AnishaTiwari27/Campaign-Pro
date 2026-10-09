import type { ListParams } from "./types";

export const queryKeys = {
  campaignsList: (params: ListParams) => ["campaigns", "list", params] as const,
  campaignDetail: (id: string, listParams: ListParams) => ["campaigns", "detail", id, listParams] as const,
  anomalies: () => ["campaigns", "anomalies"] as const,
  overview: () => ["overview"] as const,
  benchmark: () => ["benchmark"] as const,
  regions: () => ["regions"] as const,
  region: (id: string) => ["regions", id] as const,
  reports: () => ["reports"] as const,
  report: (id: string) => ["reports", id] as const,
  reportRuns: (id: string) => ["reports", id, "runs"] as const,
  search: (q: string) => ["search", q] as const,
  me: () => ["me"] as const,
  users: () => ["users"] as const,
  authOptions: () => ["auth", "options"] as const,
  health: () => ["health"] as const,
};
