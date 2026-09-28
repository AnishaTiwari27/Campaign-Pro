// Thin fetch wrapper around the Campaign Tracker Pro API.
// Route shapes: ../../docs/API_CONTRACT.md
//
// Base URL is read from VITE_API_BASE_URL (see .env.example) and falls back
// to the Go backend's own default (`go run ./cmd/server`, port 8080).
import { ApiError } from "./errors";
import { refresh as refreshTokens } from "./auth";
import { getAccessToken, getRefreshToken, setSession, clearSession } from "../auth/session";

const API_BASE = import.meta.env?.VITE_API_BASE_URL || "http://localhost:8080/api/v1";

// Builds a query string from a filter object, dropping "All"/empty/undefined
// values so the backend sees the same "no filter" it does when a param is
// simply omitted.
function toQuery(params = {}) {
  const q = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "" || value === "All") continue;
    q.set(key, String(value));
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

function withAuthHeader(init = {}) {
  const token = getAccessToken();
  if (!token) return init;
  return { ...init, headers: { ...(init.headers || {}), Authorization: `Bearer ${token}` } };
}

// A burst of requests can all hit a just-expired access token at once; they
// share one in-flight refresh instead of each spending the (single-use,
// rotating — see docs/API_CONTRACT.md) refresh token themselves.
let refreshPromise = null;

function attemptRefresh() {
  const rt = getRefreshToken();
  if (!rt) return Promise.resolve(false);
  if (!refreshPromise) {
    refreshPromise = refreshTokens(rt)
      .then((pair) => {
        setSession(pair);
        return true;
      })
      .catch(() => {
        clearSession(); // refresh token is dead too — back to the login form
        return false;
      })
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
}

// authorizedFetch attaches the bearer header and, on a 401, tries exactly
// one silent refresh-and-retry before giving up.
async function authorizedFetch(url, init) {
  let res = await fetch(url, withAuthHeader(init));
  if (res.status === 401 && (await attemptRefresh())) {
    res = await fetch(url, withAuthHeader(init));
  }
  return res;
}

async function getJSON(path, params) {
  const res = await authorizedFetch(`${API_BASE}${path}${toQuery(params)}`);
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new ApiError(res.status, body?.error?.code ?? "unknown", body?.error?.message ?? res.statusText);
  }
  return res.json();
}

export function getMeta() {
  return getJSON("/meta");
}

export function listCampaigns(filters) {
  return getJSON("/campaigns", filters);
}

export function getCampaign(id) {
  return getJSON(`/campaigns/${id}`);
}

// updateCampaignBudget sets (or, given null, clears) a campaign's planned
// budget — admin-only server-side (403 for a viewer session); the only
// mutation route on an existing campaign, since there's no
// campaign-creation UI yet. Returns the updated campaign, with `pacing`
// freshly computed against the new budget.
export async function updateCampaignBudget(id, budgetRupees) {
  const res = await authorizedFetch(`${API_BASE}/campaigns/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ budget: budgetRupees }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new ApiError(res.status, body?.error?.code ?? "unknown", body?.error?.message ?? res.statusText);
  }
  return res.json();
}

// updateCampaignApprovalStatus moves a campaign between "pending",
// "approved", and "rejected" — admin-only server-side (403 for a viewer
// session), a separate route from updateCampaignBudget on purpose (see
// docs/API_CONTRACT.md). Returns the updated campaign.
export async function updateCampaignApprovalStatus(id, status) {
  const res = await authorizedFetch(`${API_BASE}/campaigns/${id}/approval`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ status }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new ApiError(res.status, body?.error?.code ?? "unknown", body?.error?.message ?? res.statusText);
  }
  return res.json();
}

// getAnomalies is a global "what needs attention" signal (peer-outlier
// campaigns + categories that have gone quiet) — deliberately unfiltered
// by the current filter bar, same reasoning as the server side
// (models.AnomalyReport's doc comment).
export function getAnomalies() {
  return getJSON("/campaigns/anomalies");
}

// emailReport sends the CSV export for filters to the caller's own account
// email instead of downloading it — the server always uses the
// authenticated session's email, never a client-supplied recipient.
// Always resolves (even when nothing was actually sent — see
// {sent, reason} — the backend never 5xxs just because SMTP isn't
// configured in this environment); only a real request failure rejects.
export async function emailReport(filters) {
  const res = await authorizedFetch(`${API_BASE}/campaigns/email-report${toQuery(filters)}`, { method: "POST" });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new ApiError(res.status, body?.error?.code ?? "unknown", body?.error?.message ?? res.statusText);
  }
  return res.json();
}

// listUsers/updateUserRole back the new Settings tab's "Users & roles"
// panel — admin-only server-side (403 otherwise). Note the base path:
// these live under /api/v1 (auth-service serves them, but /auth/* itself
// stays the pre-identity public surface — see docs/ARCHITECTURE.md).
export function listUsers() {
  return getJSON("/users");
}

export async function updateUserRole(id, role) {
  const res = await authorizedFetch(`${API_BASE}/users/${id}/role`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ role }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new ApiError(res.status, body?.error?.code ?? "unknown", body?.error?.message ?? res.statusText);
  }
  return res.json();
}

// getAuditLog backs the Settings tab's "Audit log" panel — admin-only
// server-side (403 otherwise). Paginated, newest first.
export function getAuditLog({ page, limit } = {}) {
  return getJSON("/audit-log", { page, limit });
}

// listCreatives/createCreative back the drawer's "Creatives" section —
// creatives are additive alongside a campaign, never a replacement for
// its own reach/spend (see docs/ARCHITECTURE.md). Create is editor/admin
// only server-side (403 otherwise); list is open to any authenticated role.
export function listCreatives(campaignId) {
  return getJSON(`/campaigns/${campaignId}/creatives`);
}

export async function createCreative(campaignId, { headline, creativeType, reach, spend }) {
  const res = await authorizedFetch(`${API_BASE}/campaigns/${campaignId}/creatives`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ headline, creativeType, reach, spend }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new ApiError(res.status, body?.error?.code ?? "unknown", body?.error?.message ?? res.statusText);
  }
  return res.json();
}

export function getKPIs(filters) {
  return getJSON("/kpis", filters);
}

export function getTrend(filters) {
  return getJSON("/trend", filters);
}

export function getRegions(filters) {
  return getJSON("/regions", filters);
}

export function getBenchmark(filters) {
  return getJSON("/benchmark", filters);
}

// Downloads the filtered set as CSV and returns the Blob — the caller wires
// it to an <a download> click, same as the old client-side-generated blob.
export async function exportCampaignsCSV(filters) {
  const res = await authorizedFetch(`${API_BASE}/campaigns/export${toQuery(filters)}`);
  if (!res.ok) throw new ApiError(res.status, "export_failed", res.statusText);
  return res.blob();
}

export { ApiError };
