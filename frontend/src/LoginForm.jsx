import React, { useState } from "react";
import { Radio } from "lucide-react";
import { login } from "./api/auth";
import { setSession } from "./auth/session";
import AuthCard from "./AuthCard";

// Standalone login gate — see AuthCard for the shared visual shell with
// SignupForm. Defined independently since this renders before any session
// exists, never alongside the dashboard.
export default function LoginForm({ onShowSignup }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const pair = await login(email, password);
      setSession(pair); // notifies AuthGate, which swaps this out for the dashboard
    } catch (err) {
      setError(err.message || "Login failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthCard onSubmit={handleSubmit} error={error}>
      <label className="flex flex-col gap-1">
        <span className="text-xs" style={{ color: "var(--muted)" }}>Email</span>
        <input
          type="email" required autoFocus value={email} onChange={(e) => setEmail(e.target.value)}
          className="ctp-login-input px-3 py-2 text-sm" placeholder="you@campaigntracker.dev"
        />
      </label>

      <label className="flex flex-col gap-1">
        <span className="text-xs" style={{ color: "var(--muted)" }}>Password</span>
        <input
          type="password" required value={password} onChange={(e) => setPassword(e.target.value)}
          className="ctp-login-input px-3 py-2 text-sm"
        />
      </label>

      <button type="submit" disabled={submitting} className="ctp-login-submit px-3 py-2 text-sm font-medium mt-2">
        {submitting ? "Signing in…" : "Sign in"}
      </button>

      <div className="flex items-center gap-1.5 text-xs mt-1" style={{ color: "var(--muted)" }}>
        <Radio size={12} style={{ color: "var(--accent)" }} />
        Sessions are short-lived — you'll stay signed in as you use the dashboard.
      </div>

      <div className="text-xs text-center mt-1" style={{ color: "var(--muted)" }}>
        Don't have an account?{" "}
        <button type="button" onClick={onShowSignup} className="ctp-login-link" style={{ color: "var(--ink)" }}>
          Sign up
        </button>
      </div>
    </AuthCard>
  );
}
