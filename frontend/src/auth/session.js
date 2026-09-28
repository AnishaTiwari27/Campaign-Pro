// Holds the current session (access token in memory, refresh token in
// localStorage) and notifies subscribers when it changes — the one place
// both api/client.js (attaches the bearer header, retries after a silent
// refresh) and the UI (show the login form vs. the dashboard) read from.
//
// The access token is deliberately never persisted — only kept in this
// module's memory — so it doesn't outlive the tab. The refresh token has
// to survive a reload to avoid forcing a fresh login on every page refresh,
// so it goes in localStorage; see docs/ARCHITECTURE.md's Auth section for
// the tradeoff (an httpOnly cookie would be stronger, but auth-service
// issues tokens as JSON, not a Set-Cookie).
const REFRESH_TOKEN_KEY = "ctp_refresh_token";

let accessToken = null;
const listeners = new Set();

function notify() {
  for (const fn of listeners) fn(isAuthenticated());
}

export function getAccessToken() {
  return accessToken;
}

// getRole decodes the access token's own claims to answer "is this user
// an admin" — a UI hint only (e.g. "show the set-budget control"), never
// a security boundary: nothing client-side verifies the token's
// signature, and every write the server accepts re-checks X-User-Role
// itself after independently verifying the token (see
// services/gateway/internal/api/authmw.go). A forged role here gets a
// control to click, not a way past the server's own check.
export function getRole() {
  const claims = decodeAccessTokenClaims();
  return claims?.role ?? null;
}

// getUserId decodes the access token's own subject claim — used to spot
// "this row is me" in the Settings tab's user list (e.g. disabling your
// own role control, matching the server's own self-target rejection in
// handleUpdateUserRole). Same UI-hint-only caveat as getRole.
export function getUserId() {
  const claims = decodeAccessTokenClaims();
  return claims?.sub ?? null;
}

// hasAnyRole is getRole() checked against a set instead of one exact
// string — same UI-hint-only caveat, mirrors the backend's
// platform.RequireRole(w, r, msg, allowed...) shape so a permission gate
// reads the same way on both sides.
export function hasAnyRole(...roles) {
  return roles.includes(getRole());
}

function decodeAccessTokenClaims() {
  if (!accessToken) return null;
  try {
    const payload = accessToken.split(".")[1];
    const base64 = payload.replace(/-/g, "+").replace(/_/g, "/").padEnd(payload.length + ((4 - (payload.length % 4)) % 4), "=");
    const bytes = Uint8Array.from(atob(base64), (c) => c.charCodeAt(0));
    return JSON.parse(new TextDecoder().decode(bytes));
  } catch {
    return null; // malformed token — treat as "no claims," not a crash
  }
}

export function getRefreshToken() {
  try {
    return localStorage.getItem(REFRESH_TOKEN_KEY);
  } catch {
    return null; // private browsing / storage disabled — treat as "no session"
  }
}

export function isAuthenticated() {
  return accessToken !== null;
}

// setSession stores a fresh { accessToken, refreshToken } pair — called
// after login and after every refresh (refreshToken rotates every time;
// see docs/API_CONTRACT.md's Auth section).
export function setSession({ accessToken: access, refreshToken }) {
  accessToken = access;
  try {
    localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  } catch {
    // Storage unavailable — session still works for this tab's lifetime,
    // just won't survive a reload.
  }
  notify();
}

export function clearSession() {
  accessToken = null;
  try {
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  } catch {
    // no-op
  }
  notify();
}

// subscribe registers fn(isAuthenticated) to run on every session change and
// returns an unsubscribe function — the shape a useEffect cleanup expects.
export function subscribe(fn) {
  listeners.add(fn);
  return () => listeners.delete(fn);
}
