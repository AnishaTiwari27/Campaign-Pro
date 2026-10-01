import type { ListParams } from "../api/types";

// Converts a Campaigns ListParams snapshot into the scopeFilters payload a
// report stores, and a human label for the "read-only scope" display.
export function paramsToScope(p: ListParams): Record<string, unknown> {
  const scope: Record<string, unknown> = {};
  if (p.q) scope.q = p.q;
  if (p.type) scope.type = p.type;
  if (p.category) scope.category = p.category;
  if (p.region) scope.region = p.region;
  if (p.status) scope.status = p.status;
  if (p.approval) scope.approval = p.approval;
  if (p.range) scope.range = p.range;
  if (p.sort) scope.sort = p.sort;
  if (p.dir) scope.dir = p.dir;
  return scope;
}

export function describeScope(p: ListParams): string {
  const parts: string[] = [];
  if (p.q) parts.push(`"${p.q}"`);
  if (p.type) parts.push(p.type === "brand" ? "Brands" : "People");
  if (p.category) parts.push(p.category);
  if (p.region) parts.push(p.region);
  if (p.status) parts.push(p.status[0].toUpperCase() + p.status.slice(1));
  if (p.approval) parts.push(p.approval[0].toUpperCase() + p.approval.slice(1));
  if (p.range) parts.push(p.range === "7" ? "Last 7 days" : "Last 30 days");
  return parts.length > 0 ? parts.join(" · ") : "All campaigns";
}
