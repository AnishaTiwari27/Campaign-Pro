import React, { useEffect, useRef, useState } from "react";
import AppRoutes from "./AppRoutes";
import LoginForm from "./LoginForm";
import SignupForm from "./SignupForm";
import { refresh as refreshTokens, logout as logoutRequest } from "./api/auth";
import { isAuthenticated, getRefreshToken, setSession, clearSession, subscribe } from "./auth/session";

// Gates the dashboard behind a session: on mount, tries one silent refresh
// off whatever refresh token survived a reload in localStorage (see
// auth/session.js) so a page refresh doesn't force a fresh login every
// time; otherwise renders LoginForm/SignupForm (toggled locally, see
// authView) until one succeeds.
export default function AuthGate() {
  const [authed, setAuthed] = useState(isAuthenticated());
  const [checking, setChecking] = useState(!isAuthenticated());
  const [authView, setAuthView] = useState("login"); // "login" | "signup"

  useEffect(() => subscribe(setAuthed), []);

  // Guards against React.StrictMode's dev-only double-invoke of this
  // effect (main.jsx wraps the app in <React.StrictMode>): without it,
  // this fires the same refresh token twice on every mount. Refresh
  // tokens are single-use/rotating (see docs/API_CONTRACT.md), so the
  // second call always 401s — and if its .catch(clearSession) settles
  // *after* the first call's .then(setSession) already restored a valid
  // session, it silently signs the user back out right after logging
  // them in. A ref survives StrictMode's simulated remount (it doesn't
  // recreate the component instance), so this only lets the real,
  // sticking invocation fire the request.
  const refreshAttempted = useRef(false);
  useEffect(() => {
    if (isAuthenticated()) {
      setChecking(false);
      return;
    }
    const rt = getRefreshToken();
    if (!rt) {
      setChecking(false);
      return;
    }
    if (refreshAttempted.current) return;
    refreshAttempted.current = true;
    refreshTokens(rt)
      .then(setSession)
      .catch(clearSession)
      .finally(() => setChecking(false));
  }, []);

  async function handleSignOut() {
    const rt = getRefreshToken();
    clearSession(); // sign out locally immediately, regardless of network
    if (rt) logoutRequest(rt).catch(() => {}); // best-effort server-side revoke
  }

  if (checking) return null; // avoids a login-form flash while the silent refresh is in flight
  if (authed) return <AppRoutes onSignOut={handleSignOut} />;
  return authView === "signup"
    ? <SignupForm onShowLogin={() => setAuthView("login")} />
    : <LoginForm onShowSignup={() => setAuthView("signup")} />;
}
