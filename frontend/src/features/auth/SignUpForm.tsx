import { useState, type FormEvent } from "react";
import { useSignup } from "../../api/auth";
import { ApiError } from "../../api/client";

export function SignUpForm({
  minPasswordLength,
  onCreated,
  onSwitchToSignIn,
}: {
  minPasswordLength: number;
  /** Hands the new address to the sign-in form, which is where signup
   *  ends: the account exists, now prove the password works. */
  onCreated: (email: string) => void;
  onSwitchToSignIn: () => void;
}) {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  // Mismatch is checked here and nowhere else: the server has no second
  // field to compare against, and should not be told about one.
  const [localError, setLocalError] = useState<string | null>(null);
  const signup = useSignup();

  function submit(e: FormEvent) {
    e.preventDefault();
    setLocalError(null);

    if (password.length < minPasswordLength) {
      setLocalError(`Password must be at least ${minPasswordLength} characters.`);
      return;
    }
    // There is no password-reset flow in this product, so a typo here
    // would lock someone out of an account for good. Hence the second
    // field, and hence checking it before anything is created.
    if (password !== confirm) {
      setLocalError("Those passwords do not match.");
      return;
    }

    signup.mutate(
      { name: name.trim(), email: email.trim(), password },
      { onSuccess: (user) => onCreated(user.email) },
    );
  }

  const apiError = signup.error instanceof ApiError ? signup.error : null;
  const message =
    localError ??
    apiError?.message ??
    (signup.error ? "Something went wrong. Please try again." : null);

  // The server names the field it rejected, so the message can sit under
  // the input it belongs to instead of floating above the whole form.
  const fieldError = (field: string) =>
    !localError && apiError?.field === field ? apiError.message : null;

  return (
    <form className="auth-form" onSubmit={submit} noValidate>
      <div className="auth-form-head">
        <h1>Create your account</h1>
        <p>
          You'll get analyst access: full visibility of campaigns, reports and
          analytics. Approvals and user management stay with an admin.
        </p>
      </div>

      <label className="auth-field">
        <span>Full name</span>
        <input
          className="input"
          type="text"
          autoComplete="name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          autoFocus
          required
        />
        {fieldError("name") && <small className="auth-field-error">{fieldError("name")}</small>}
      </label>

      <label className="auth-field">
        <span>Work email</span>
        <input
          className="input"
          type="email"
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        {fieldError("email") && <small className="auth-field-error">{fieldError("email")}</small>}
      </label>

      <label className="auth-field">
        <span>Password</span>
        <input
          className="input"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        <small className="auth-hint">At least {minPasswordLength} characters.</small>
        {fieldError("password") && (
          <small className="auth-field-error">{fieldError("password")}</small>
        )}
      </label>

      <label className="auth-field">
        <span>Confirm password</span>
        <input
          className="input"
          type="password"
          autoComplete="new-password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          required
        />
      </label>

      {message && !["name", "email", "password"].includes(apiError?.field ?? "") && (
        <p className="auth-error" role="alert">
          {message}
        </p>
      )}

      <button type="submit" className="btn btn-primary auth-submit" disabled={signup.isPending}>
        {signup.isPending ? "Creating account…" : "Create account"}
      </button>

      <p className="auth-switch">
        Already have an account?{" "}
        <button type="button" className="auth-link" onClick={onSwitchToSignIn}>
          Sign in
        </button>
      </p>
    </form>
  );
}
