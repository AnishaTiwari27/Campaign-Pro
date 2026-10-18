import { useState, type FormEvent } from "react";
import { useChangePassword, useSession, useUpdateName } from "../../api/auth";
import { ApiError } from "../../api/client";

/** The part of Settings you can actually change.
 *
 *  Everything else on this page reports something; these two write.
 *  They are the only account details a person owns outright — role,
 *  rights and organisation are an admin's to set, not theirs.
 */
export function ProfilePanel() {
  const { data: me } = useSession();
  const updateName = useUpdateName();
  const changePassword = useChangePassword();

  const [name, setName] = useState(me?.name ?? "");
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  if (!me) return null;

  const dirty = name.trim() !== me.name && name.trim() !== "";

  function saveName(e: FormEvent) {
    e.preventDefault();
    if (!dirty) return;
    updateName.mutate(name.trim());
  }

  function savePassword(e: FormEvent) {
    e.preventDefault();
    setLocalError(null);
    // Checked here and nowhere else: the server has no second field to
    // compare against, and should not be told about one.
    if (next !== confirm) {
      setLocalError("Those passwords do not match.");
      return;
    }
    changePassword.mutate(
      { currentPassword: current, newPassword: next },
      {
        onSuccess: () => {
          setCurrent("");
          setNext("");
          setConfirm("");
        },
      },
    );
  }

  const nameError = updateName.error instanceof ApiError ? updateName.error.message : null;
  const pwError =
    localError ?? (changePassword.error instanceof ApiError ? changePassword.error.message : null);

  return (
    <section className="card settings-panel">
      <h4>Your profile</h4>

      <form className="settings-form" onSubmit={saveName}>
        <label className="settings-field">
          <span>Display name</span>
          <input
            className="input"
            value={name}
            onChange={(e) => setName(e.target.value)}
            autoComplete="name"
          />
          <small className="settings-hint">
            Shown on every approval and audit line you write.
          </small>
        </label>
        {nameError && (
          <p className="settings-error" role="alert">
            {nameError}
          </p>
        )}
        <div className="settings-form-actions">
          {updateName.isSuccess && !dirty && (
            <span className="settings-saved" role="status">
              Saved
            </span>
          )}
          <button
            type="submit"
            className="btn btn-primary btn-sm"
            disabled={!dirty || updateName.isPending}
          >
            {updateName.isPending ? "Saving…" : "Save name"}
          </button>
        </div>
      </form>

      <div className="settings-form-divider" />

      <form className="settings-form" onSubmit={savePassword}>
        <label className="settings-field">
          <span>Current password</span>
          <input
            className="input"
            type="password"
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
            autoComplete="current-password"
            required
          />
        </label>
        <label className="settings-field">
          <span>New password</span>
          <input
            className="input"
            type="password"
            value={next}
            onChange={(e) => setNext(e.target.value)}
            autoComplete="new-password"
            required
          />
        </label>
        <label className="settings-field">
          <span>Confirm new password</span>
          <input
            className="input"
            type="password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            autoComplete="new-password"
            required
          />
          {/* There is no reset flow in this product, so a typo here
              would lock someone out for good. Hence the second field. */}
          <small className="settings-hint">
            There is no password reset, so keep this somewhere safe.
          </small>
        </label>
        {pwError && (
          <p className="settings-error" role="alert">
            {pwError}
          </p>
        )}
        <div className="settings-form-actions">
          {changePassword.isSuccess && (
            <span className="settings-saved" role="status">
              Password changed
            </span>
          )}
          <button
            type="submit"
            className="btn btn-sm"
            disabled={changePassword.isPending || !current || !next}
          >
            {changePassword.isPending ? "Changing…" : "Change password"}
          </button>
        </div>
      </form>
    </section>
  );
}
