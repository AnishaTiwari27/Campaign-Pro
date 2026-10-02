const BASE = "/api";

export class ApiError extends Error {
  status: number;
  field?: string;
  constructor(status: number, message: string, field?: string) {
    super(message);
    this.status = status;
    this.field = field;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
  });
  if (!res.ok) {
    let message = res.statusText;
    let field: string | undefined;
    try {
      const body = await res.json();
      if (typeof body.error === "string") message = body.error;
      if (typeof body.field === "string") field = body.field;
    } catch {
      // no JSON body to read
    }
    throw new ApiError(res.status, message, field);
  }
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export function buildQuery(params: Record<string, string | number | undefined>): string {
  const usp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") usp.set(k, String(v));
  }
  const s = usp.toString();
  return s ? `?${s}` : "";
}

export function get<T>(path: string, params?: Record<string, string | number | undefined>): Promise<T> {
  return request<T>(`${path}${params ? buildQuery(params) : ""}`);
}

export function post<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, { method: "POST", body: body !== undefined ? JSON.stringify(body) : undefined });
}

export function patch<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, { method: "PATCH", body: body !== undefined ? JSON.stringify(body) : undefined });
}

export function exportUrl(params: Record<string, string | number | undefined>): string {
  return `${BASE}/campaigns/export.csv${buildQuery(params)}`;
}
