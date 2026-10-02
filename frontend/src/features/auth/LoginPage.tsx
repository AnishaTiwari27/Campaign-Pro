import { useState, type FormEvent } from "react";
import { useLogin } from "../../api/auth";
import { ApiError } from "../../api/client";
import "./LoginPage.css";

export function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const login = useLogin();

  function submit(e: FormEvent) {
    e.preventDefault();
    if (!email.trim() || !password) return;
    login.mutate({ email: email.trim(), password });
  }

  // The server returns one message for a bad email and a bad password
  // alike, so this can't be used to discover which accounts exist.
  const message =
    login.error instanceof ApiError
      ? login.error.message
      : login.error
        ? "Something went wrong. Please try again."
        : null;

  return (
    <div className="login-page">
      <form className="login-card card" onSubmit={submit}>
        <div className="login-brand">
          <span className="login-mark">CTP</span>
          <span className="login-brand-text">
            Campaign Tracker Pro
            <span className="login-brand-sub">India · all regions</span>
          </span>
        </div>

        <h1>Sign in</h1>

        <label className="login-field">
          <span>Email</span>
          <input
            className="input"
            type="email"
            autoComplete="username"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoFocus
            required
          />
        </label>

        <label className="login-field">
          <span>Password</span>
          <input
            className="input"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </label>

        {message && (
          <p className="login-error" role="alert">
            {message}
          </p>
        )}

        <button type="submit" className="btn btn-primary login-submit" disabled={login.isPending}>
          {login.isPending ? "Signing in…" : "Sign in"}
        </button>
      </form>
    </div>
  );
}
