import { useState, type FormEvent } from "react";
import { useLogin } from "../../api/auth";
import { ApiError } from "../../api/client";
import { SsoButton } from "./SsoButton";

export function SignInForm({
  initialEmail = "",
  notice,
  error,
  onSwitchToSignUp,
  ssoEnabled = false,
  ssoLabel = "single sign-on",
}: {
  initialEmail?: string;
  /** Shown above the fields, e.g. after an account has just been created. */
  notice?: string | null;
  /** A failure that happened before this form was mounted — a refused
   *  trip to the identity provider, which redirects back here. Shown
   *  until this form produces an error of its own. */
  error?: string | null;
  /** Absent when signup is disabled, which hides the link entirely. */
  onSwitchToSignUp?: () => void;
  /** True only when the server actually discovered a provider at boot. */
  ssoEnabled?: boolean;
  /** The provider's name, derived server-side from the issuer. */
  ssoLabel?: string;
}) {
  const [email, setEmail] = useState(initialEmail);
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
        : (error ?? null);

  return (
    <form className="auth-form" onSubmit={submit} noValidate>
      <div className="auth-form-head">
        <h1>Welcome back</h1>
        <p>Sign in to pick up where your campaigns left off.</p>
      </div>

      {/* The notice is replaced by any error, so the two never stack up
          and contradict each other. */}
      {notice && !message && (
        <p className="auth-notice" role="status">
          {notice}
        </p>
      )}

      <label className="auth-field">
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

      <label className="auth-field">
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
        <p className="auth-error" role="alert">
          {message}
        </p>
      )}

      <button type="submit" className="btn btn-primary auth-submit" disabled={login.isPending}>
        {login.isPending ? "Signing in…" : "Sign in"}
      </button>

      {/* Shown only when the server discovered a provider at boot, so
          this is never a button that leads nowhere. */}
      {ssoEnabled && <SsoButton label={ssoLabel} />}

      {onSwitchToSignUp && (
        <p className="auth-switch">
          New here?{" "}
          <button type="button" className="auth-link" onClick={onSwitchToSignUp}>
            Create an account
          </button>
        </p>
      )}
    </form>
  );
}
