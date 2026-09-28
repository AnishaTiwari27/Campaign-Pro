// Calls auth-service's routes through the gateway. These live at the
// gateway's root (/auth/...), not under /api/v1 like every other route in
// client.js — see docs/API_CONTRACT.md's Auth section.
import { ApiError } from "./errors";

const API_BASE = import.meta.env?.VITE_API_BASE_URL || "http://localhost:8080/api/v1";
const AUTH_BASE = API_BASE.replace(/\/api\/v1\/?$/, "");

async function postJSON(path, body) {
  const res = await fetch(`${AUTH_BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const parsed = await res.json().catch(() => null);
    throw new ApiError(res.status, parsed?.error?.code ?? "unknown", parsed?.error?.message ?? res.statusText);
  }
  return res.status === 204 ? null : res.json();
}

// register creates a new "viewer" account and logs it straight in — the
// response is the same { accessToken, refreshToken } shape as login.
export function register(email, password) {
  return postJSON("/auth/register", { email, password });
}

export function login(email, password) {
  return postJSON("/auth/login", { email, password });
}

export function refresh(refreshToken) {
  return postJSON("/auth/refresh", { refreshToken });
}

export function logout(refreshToken) {
  return postJSON("/auth/logout", { refreshToken });
}
