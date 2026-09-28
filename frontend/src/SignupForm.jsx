import React, { useState } from "react";
import { UserPlus } from "lucide-react";
import { register } from "./api/auth";
import { setSession } from "./auth/session";
import AuthCard from "./AuthCard";

const MIN_PASSWORD_LENGTH = 8; // mirrors auth-service's own check — see docs/API_CONTRACT.md

// Standalone signup gate — every account created here is a "viewer"
// (auth-service hardcodes that; there's no way to request "admin" through
// this form). See AuthCard for the shared shell with LoginForm.
export default function SignupForm({ onShowLogin }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e) {
    e.preventDefault();
    setError(null);

    if (password.length < MIN_PASSWORD_LENGTH) {
      setError(`Password must be at least ${MIN_PASSWORD_LENGTH} characters`);
      return;
    }
    if (password !== confirmPassword) {
      setError("Passwords don't match");
      return;
    }

    setSubmitting(true);
    try {
      const pair = await register(email, password);
      setSession(pair); // registering logs you straight in — notifies AuthGate
    } catch (err) {
      setError(err.message || "Sign up failed");
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
          className="ctp-login-input px-3 py-2 text-sm" placeholder={`At least ${MIN_PASSWORD_LENGTH} characters`}
        />
      </label>

      <label className="flex flex-col gap-1">
        <span className="text-xs" style={{ color: "var(--muted)" }}>Confirm password</span>
        <input
          type="password" required value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)}
          className="ctp-login-input px-3 py-2 text-sm"
        />
      </label>

      <button type="submit" disabled={submitting} className="ctp-login-submit px-3 py-2 text-sm font-medium mt-2">
        {submitting ? "Creating account…" : "Create account"}
      </button>

      <div className="flex items-center gap-1.5 text-xs mt-1" style={{ color: "var(--muted)" }}>
        <UserPlus size={12} style={{ color: "var(--accent)" }} />
        New accounts are viewer-only — read access, no campaign creation.
      </div>

      <div className="text-xs text-center mt-1" style={{ color: "var(--muted)" }}>
        Already have an account?{" "}
        <button type="button" onClick={onShowLogin} className="ctp-login-link" style={{ color: "var(--ink)" }}>
          Sign in
        </button>
      </div>
    </AuthCard>
  );
}
