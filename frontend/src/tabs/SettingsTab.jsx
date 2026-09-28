import React, { useState } from "react";
import { useOutletContext } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ShieldAlert, ChevronLeft, ChevronRight } from "lucide-react";
import { listUsers, updateUserRole, getAuditLog } from "../api/client";
import { hasAnyRole, getUserId } from "../auth/session";
import { fmtDate } from "../format";

const ROLE_OPTIONS = ["viewer", "editor", "approver", "admin"];
const AUDIT_PAGE_SIZE = 25;

// fmtDate expects YYYY-MM-DD; audit entries carry a full RFC3339 instant
// instead (they're events, not calendar dates) — this formats the whole
// thing (date + time), reusing fmtDate's day for the date half.
function fmtDateTime(iso) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return `${fmtDate(iso.slice(0, 10))} ${d.toLocaleTimeString("en-IN", { hour: "2-digit", minute: "2-digit" })}`;
}

// Admin-only: manage every user's role. Settings had a placeholder route
// with no content until this — see docs/ROADMAP.md's Phase C.
export default function SettingsTab() {
  const { onError } = useOutletContext();
  const isAdmin = hasAnyRole("admin"); // UI hint only — server re-checks on every write
  const myID = getUserId();
  const queryClient = useQueryClient();
  const [savingID, setSavingID] = useState(null);
  const [rowError, setRowError] = useState(null);
  const [auditPage, setAuditPage] = useState(1);

  const usersQuery = useQuery({ queryKey: ["users"], queryFn: listUsers, enabled: isAdmin });
  const users = usersQuery.data?.data ?? [];

  const auditQuery = useQuery({
    queryKey: ["audit-log", auditPage],
    queryFn: () => getAuditLog({ page: auditPage, limit: AUDIT_PAGE_SIZE }),
    enabled: isAdmin,
  });
  const auditEntries = auditQuery.data?.data ?? [];
  const auditTotal = auditQuery.data?.total ?? 0;
  const auditLastPage = Math.max(1, Math.ceil(auditTotal / AUDIT_PAGE_SIZE));

  async function handleRoleChange(userID, role) {
    setSavingID(userID);
    setRowError(null);
    try {
      await updateUserRole(userID, role);
      queryClient.invalidateQueries({ queryKey: ["users"] });
    } catch (e) {
      setRowError(`${userID}: ${e.message}`);
      onError?.(e.message);
    } finally {
      setSavingID(null);
    }
  }

  if (!isAdmin) {
    return (
      <div className="px-6 pb-8">
        <div className="ctp-panel p-5 flex items-center gap-2 text-sm" style={{ color: "var(--muted)" }}>
          <ShieldAlert size={16} />Settings are admin-only.
        </div>
      </div>
    );
  }

  return (
    <div className="px-6 pb-8 flex flex-col gap-4">
      <div className={`ctp-panel p-5 ${usersQuery.isFetching ? "ctp-fetching" : ""}`}>
        <h2 className="ctp-display font-semibold mb-4">Users &amp; roles</h2>
        <table className="w-full text-left text-xs">
          <thead>
            <tr style={{ color: "var(--muted)", borderBottom: "1px solid var(--line)" }}>
              <th className="py-2 font-medium">Email</th>
              <th className="py-2 font-medium">Role</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id} style={{ borderBottom: "1px solid var(--line)" }}>
                <td className="py-2.5 font-medium">
                  {u.email}
                  {u.id === myID && <span style={{ color: "var(--muted)" }}> (you)</span>}
                </td>
                <td className="py-2.5">
                  <select
                    className="ctp-select px-2 py-1 text-xs"
                    value={u.role}
                    disabled={u.id === myID || savingID === u.id}
                    onChange={(e) => handleRoleChange(u.id, e.target.value)}
                    title={u.id === myID ? "Ask another admin to change your own role" : undefined}
                  >
                    {ROLE_OPTIONS.map((r) => <option key={r} value={r}>{r}</option>)}
                  </select>
                </td>
              </tr>
            ))}
            {users.length === 0 && !usersQuery.isFetching && (
              <tr><td colSpan={2} className="py-6 text-center" style={{ color: "var(--muted)" }}>No users found.</td></tr>
            )}
          </tbody>
        </table>
        {rowError && <div className="mt-2 text-xs" style={{ color: "var(--alert)" }}>{rowError}</div>}
      </div>

      <div className={`ctp-panel p-5 ${auditQuery.isFetching ? "ctp-fetching" : ""}`}>
        <div className="flex items-center justify-between mb-4">
          <h2 className="ctp-display font-semibold">Audit log</h2>
          <div className="flex items-center gap-3 text-xs" style={{ color: "var(--muted)" }}>
            <span>{auditTotal} total</span>
            <div className="flex items-center gap-1">
              <button onClick={() => setAuditPage((p) => Math.max(1, p - 1))} disabled={auditPage <= 1} className="disabled:opacity-30">
                <ChevronLeft size={14} />
              </button>
              <span>Page {auditPage} of {auditLastPage}</span>
              <button onClick={() => setAuditPage((p) => Math.min(auditLastPage, p + 1))} disabled={auditPage >= auditLastPage} className="disabled:opacity-30">
                <ChevronRight size={14} />
              </button>
            </div>
          </div>
        </div>
        <table className="w-full text-left text-xs">
          <thead>
            <tr style={{ color: "var(--muted)", borderBottom: "1px solid var(--line)" }}>
              <th className="py-2 font-medium">When</th>
              <th className="py-2 font-medium">Action</th>
              <th className="py-2 font-medium">Actor</th>
              <th className="py-2 font-medium">Campaign</th>
              <th className="py-2 font-medium">Details</th>
            </tr>
          </thead>
          <tbody>
            {auditEntries.map((e) => (
              <tr key={e.id} style={{ borderBottom: "1px solid var(--line)" }}>
                <td className="py-2.5" style={{ color: "var(--muted)" }}>{fmtDateTime(e.createdAt)}</td>
                <td className="py-2.5"><span className="ctp-tag px-2 py-0.5 text-xs">{e.action}</span></td>
                <td className="py-2.5">{e.actorEmail || e.actorId}</td>
                <td className="py-2.5" style={{ color: "var(--muted)" }}>{e.campaignId ?? "—"}</td>
                <td className="py-2.5">{e.details}</td>
              </tr>
            ))}
            {auditEntries.length === 0 && !auditQuery.isFetching && (
              <tr><td colSpan={5} className="py-6 text-center" style={{ color: "var(--muted)" }}>No audit entries yet.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
