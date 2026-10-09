import { useState, type FormEvent } from "react";
import { useSession } from "../../api/auth";
import { useUsers, useCreateUser, useSetUserRole, useResetPassword } from "../../api/users";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonBlock } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import { ApiError } from "../../api/client";
import { GrantEditor } from "./GrantEditor";
import { formatDate } from "../../lib/format";
import "./UsersPage.css";

/** A password the server will never show again. Kept in component state
 *  only — never cached, never refetched, gone on navigation, which is the
 *  honest lifetime for something shown once. */
type OneTimePassword = { email: string; password: string };

export function UsersPage() {
  useSetBreadcrumbs([{ label: "Team" }]);
  const { data: me } = useSession();
  const canManage = me?.canManageUsers ?? false;

  const { data, isLoading } = useUsers(canManage);
  const createUser = useCreateUser();
  const setRole = useSetUserRole();
  const resetPassword = useResetPassword();

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [role, setRole_] = useState("analyst");
  const [isAgency, setIsAgency] = useState(true);
  const [issued, setIssued] = useState<OneTimePassword | null>(null);
  // Which client's access is open for editing, if any.
  const [editing, setEditing] = useState<string | null>(null);

  if (!canManage) {
    return (
      <EmptyState
        title="Admins only"
        description="Managing accounts is an admin's job. Ask one of yours if you need a change."
      />
    );
  }
  if (isLoading || !data) return <SkeletonBlock height={420} />;

  function submit(e: FormEvent) {
    e.preventDefault();
    if (!name.trim() || !email.trim()) return;
    createUser.mutate(
      { name: name.trim(), email: email.trim(), role, isAgency },
      {
        onSuccess: (res) => {
          setIssued({ email: res.user.email, password: res.password });
          setName("");
          setEmail("");
        },
      },
    );
  }

  const createError =
    createUser.error instanceof ApiError
      ? createUser.error.message
      : createUser.error
        ? "Something went wrong. Please try again."
        : null;

  return (
    <div className="users-page">
      <div className="page-header">
        <div>
          <h1>Team</h1>
          <p className="users-lede">
            Who can sign in, and what each of them may do. Changing a role here changes it on the server, which is what
            every permission is actually checked against — the screens only hide what the server would refuse anyway.
          </p>
        </div>
      </div>

      {issued && (
        <div className="card users-issued" role="status">
          <div>
            <h4>Password for {issued.email}</h4>
            <p>
              Shown once and stored only as a hash. Send it to them out of band, and have them change it by asking for
              another reset once they are in.
            </p>
          </div>
          <code className="mono users-issued-password">{issued.password}</code>
          <button type="button" className="btn" onClick={() => setIssued(null)}>
            Done
          </button>
        </div>
      )}

      <form className="card users-new" onSubmit={submit}>
        <h4>Add someone</h4>
        <div className="users-new-row">
          <label className="users-field">
            <span>Full name</span>
            <input className="input" value={name} onChange={(e) => setName(e.target.value)} required />
          </label>
          <label className="users-field">
            <span>Email</span>
            <input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          </label>
          <label className="users-field">
            <span>Role</span>
            <select className="select" value={role} onChange={(e) => setRole_(e.target.value)}>
              {data.roles.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </select>
          </label>
          <label className="users-field users-field-check">
            <span>Agency staff</span>
            <input
              type="checkbox"
              checked={isAgency}
              onChange={(e) => setIsAgency(e.target.checked)}
              aria-describedby="agency-hint"
            />
          </label>
          <button type="submit" className="btn btn-primary users-new-submit" disabled={createUser.isPending}>
            {createUser.isPending ? "Creating…" : "Create account"}
          </button>
        </div>
        <p id="agency-hint" className="users-hint">
          Agency staff see every account. Uncheck for an external client, who never sees the approval queue or the
          cross-account benchmarks whatever role they hold.
        </p>
        {createError && (
          <p className="users-error" role="alert">
            {createError}
          </p>
        )}
      </form>

      <div className="card users-table-card">
        <h4>{data.items.length} accounts</h4>
        <table className="users-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Email</th>
              <th>Role</th>
              <th>Sees</th>
              <th>Added</th>
              <th className="users-col-action">Password</th>
            </tr>
          </thead>
          <tbody>
            {data.items.flatMap((u) => [
              <tr key={u.id}>
                <td>
                  {u.name}
                  {u.isSelf && <span className="users-you">you</span>}
                </td>
                <td className="users-email">{u.email}</td>
                <td>
                  <select
                    className="select users-role-select"
                    value={u.role}
                    disabled={u.isSelf || setRole.isPending}
                    aria-label={`Role for ${u.name}`}
                    title={u.isSelf ? "You cannot change your own role — ask another admin" : undefined}
                    onChange={(e) => setRole.mutate({ id: u.id, role: e.target.value, isAgency: u.isAgency })}
                  >
                    {data.roles.map((r) => (
                      <option key={r} value={r}>
                        {r}
                      </option>
                    ))}
                  </select>
                </td>
                <td>
                  {u.isAgency ? (
                    "Every account"
                  ) : (
                    <button
                      type="button"
                      className="users-grants-link"
                      onClick={() => setEditing(editing === u.id ? null : u.id)}
                    >
                      {editing === u.id ? "Close access" : "Only granted campaigns"}
                    </button>
                  )}
                </td>
                <td className="users-date">{formatDate(u.createdAt)}</td>
                <td className="users-col-action">
                  <button
                    type="button"
                    className="btn users-reset"
                    disabled={resetPassword.isPending}
                    onClick={() =>
                      resetPassword.mutate(u.id, {
                        onSuccess: (res) => setIssued({ email: u.email, password: res.password }),
                      })
                    }
                  >
                    Reset
                  </button>
                </td>
              </tr>,
              editing === u.id ? (
                <tr key={`${u.id}-grants`} className="users-grant-row">
                  <td colSpan={6}>
                    <GrantEditor userId={u.id} userName={u.name} onClose={() => setEditing(null)} />
                  </td>
                </tr>
              ) : null,
            ])}
          </tbody>
        </table>
      </div>
    </div>
  );
}
