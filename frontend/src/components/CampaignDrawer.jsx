import React, { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { X, AlertTriangle } from "lucide-react";
import SubjectTypeIcon from "./SubjectTypeIcon";
import { fmtDate, fmtMoney } from "../format";
import { hasAnyRole } from "../auth/session";
import { updateCampaignBudget, updateCampaignApprovalStatus, getAnomalies, listCreatives, createCreative } from "../api/client";
import { PACING_LABEL, PACING_COLOR } from "../pacing";
import { APPROVAL_LABEL, APPROVAL_COLOR } from "../approval";

const CREATIVE_TYPES = ["image", "video", "carousel", "text"];

// Detail drawer for one campaign — opened from a row click in any tab
// (Overview's feed, the Campaigns table). Lives here once instead of
// duplicated per tab. Budget and approval status are the two editable
// fields here — there's no campaign-creation UI yet (see
// docs/ROADMAP.md), so these inline controls are the only way an admin
// assigns/changes them through the product.
export default function CampaignDrawer({ campaign, onClose, onFocusSubject, onBudgetUpdated, onApprovalUpdated }) {
  const [editingBudget, setEditingBudget] = useState(false);
  const [budgetInput, setBudgetInput] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(null);
  const [approvalSaving, setApprovalSaving] = useState(false);
  const [approvalError, setApprovalError] = useState(null);
  const [addingCreative, setAddingCreative] = useState(false);
  const [creativeHeadline, setCreativeHeadline] = useState("");
  const [creativeType, setCreativeType] = useState(CREATIVE_TYPES[0]);
  const [creativeReach, setCreativeReach] = useState("");
  const [creativeSpend, setCreativeSpend] = useState("");
  const [creativeSaving, setCreativeSaving] = useState(false);
  const [creativeError, setCreativeError] = useState(null);
  const queryClient = useQueryClient();

  // Same ["anomalies"] key OverviewTab's own query uses — React Query
  // dedupes/caches this rather than each place issuing its own request.
  // Enabled only once a campaign is open, since this drawer can mount
  // before one is selected (campaign is null on first render below).
  const anomaliesQuery = useQuery({ queryKey: ["anomalies"], queryFn: getAnomalies, enabled: !!campaign });
  const anomaly = anomaliesQuery.data?.campaigns?.find((a) => a.id === campaign?.id);

  const creativesQuery = useQuery({
    queryKey: ["creatives", campaign?.id],
    queryFn: () => listCreatives(campaign.id),
    enabled: !!campaign,
  });
  const creatives = creativesQuery.data?.data ?? [];

  if (!campaign) return null;

  // UI hints only — see session.js's hasAnyRole comment; the server
  // independently re-checks via platform.RequireRole on every write.
  // Budget is editor/admin's tier, approval is approver/admin's — real
  // separation of duties, see docs/ARCHITECTURE.md's RBAC section.
  const canEditBudget = hasAnyRole("admin", "editor");
  const canApprove = hasAnyRole("admin", "approver");
  const canAddCreative = hasAnyRole("admin", "editor"); // same tier as creating the campaign itself

  async function handleAddCreative() {
    if (!creativeHeadline.trim()) {
      setCreativeError("Headline is required");
      return;
    }
    setCreativeSaving(true);
    setCreativeError(null);
    try {
      await createCreative(campaign.id, {
        headline: creativeHeadline.trim(),
        creativeType,
        reach: Number(creativeReach) || 0,
        spend: Number(creativeSpend) || 0,
      });
      setAddingCreative(false);
      setCreativeHeadline("");
      setCreativeReach("");
      setCreativeSpend("");
      queryClient.invalidateQueries({ queryKey: ["creatives", campaign.id] });
    } catch (e) {
      setCreativeError(e.message);
    } finally {
      setCreativeSaving(false);
    }
  }

  async function handleSetApproval(status) {
    setApprovalSaving(true);
    setApprovalError(null);
    try {
      const updated = await updateCampaignApprovalStatus(campaign.id, status);
      onApprovalUpdated?.(updated);
    } catch (e) {
      setApprovalError(e.message);
    } finally {
      setApprovalSaving(false);
    }
  }

  const rows = [
    ["Status", campaign.status],
    ["Ad format", campaign.adType],
    ["Platform", campaign.platform],
    ["Region", campaign.region],
    ["Start date", fmtDate(campaign.start)],
    ["End date", fmtDate(campaign.end)],
    ["Estimated reach", campaign.reach.toLocaleString("en-IN")],
    ["Estimated spend", fmtMoney(campaign.spend)],
  ];

  function startEditingBudget() {
    setBudgetInput(campaign.budget != null ? String(campaign.budget) : "");
    setError(null);
    setEditingBudget(true);
  }

  async function handleSaveBudget() {
    const trimmed = budgetInput.trim();
    const parsed = trimmed === "" ? null : Number(trimmed);
    if (parsed !== null && (Number.isNaN(parsed) || parsed < 0)) {
      setError("Enter a non-negative number, or leave blank to clear it");
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const updated = await updateCampaignBudget(campaign.id, parsed);
      setEditingBudget(false);
      onBudgetUpdated?.(updated);
    } catch (e) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <div className="ctp-drawer-backdrop" onClick={onClose} />
      <div className="ctp-drawer p-6">
        <div className="flex items-start justify-between mb-6">
          <div>
            <div className="ctp-display text-xl font-bold inline-flex items-center gap-2">
              <SubjectTypeIcon type={campaign.subjectType} />{campaign.subject}
            </div>
            <div className="flex items-center gap-1.5 mt-1">
              <span className="ctp-tag px-2 py-0.5 text-xs inline-block">{campaign.category}</span>
              <span className="text-xs" style={{ color: "var(--muted)" }}>{campaign.subjectType === "person" ? "Person" : "Brand"}</span>
            </div>
          </div>
          <button onClick={onClose} style={{ color: "var(--muted)" }}><X size={18} /></button>
        </div>
        <div className="flex flex-col gap-4 text-sm">
          {rows.map(([label, val]) => (
            <div key={label} className="flex items-center justify-between" style={{ borderBottom: "1px solid var(--line)", paddingBottom: 8 }}>
              <span style={{ color: "var(--muted)" }}>{label}</span>
              <span className={`font-medium ${label === "Status" ? (val === "Live" ? "ctp-status-live" : "ctp-status-completed") : ""}`}>{val}</span>
            </div>
          ))}

          <div className="flex items-center justify-between" style={{ borderBottom: editingBudget ? "none" : "1px solid var(--line)", paddingBottom: editingBudget ? 0 : 8 }}>
            <span style={{ color: "var(--muted)" }}>Budget</span>
            {!editingBudget && (
              <span className="inline-flex items-center gap-2">
                <span className="font-medium">{campaign.budget != null ? fmtMoney(campaign.budget) : "Not set"}</span>
                {canEditBudget && (
                  <button onClick={startEditingBudget} className="text-xs underline" style={{ color: "var(--accent)" }}>
                    {campaign.budget != null ? "Edit" : "Set"}
                  </button>
                )}
              </span>
            )}
          </div>

          {editingBudget && (
            <div className="flex flex-col gap-2" style={{ borderBottom: "1px solid var(--line)", paddingBottom: 8 }}>
              <div className="flex items-center gap-2">
                <input
                  type="number" min="0" autoFocus value={budgetInput}
                  onChange={(e) => setBudgetInput(e.target.value)}
                  placeholder="Rupees, blank to clear" className="ctp-select px-2 py-1 text-xs flex-1"
                />
                <button onClick={handleSaveBudget} disabled={saving} className="text-xs font-medium px-2 py-1 disabled:opacity-50" style={{ background: "var(--ink)", color: "var(--base)" }}>
                  {saving ? "Saving…" : "Save"}
                </button>
                <button onClick={() => setEditingBudget(false)} className="text-xs" style={{ color: "var(--muted)" }}>Cancel</button>
              </div>
              {error && <span className="text-xs" style={{ color: "var(--alert)" }}>{error}</span>}
            </div>
          )}

          {campaign.pacing && (
            <div className="flex items-center justify-between" style={{ borderBottom: "1px solid var(--line)", paddingBottom: 8 }}>
              <span style={{ color: "var(--muted)" }}>Pacing</span>
              <span className="font-medium" style={{ color: PACING_COLOR[campaign.pacing] }}>
                {PACING_LABEL[campaign.pacing] ?? campaign.pacing}
              </span>
            </div>
          )}

          <div className="flex items-center justify-between" style={{ borderBottom: "1px solid var(--line)", paddingBottom: 8 }}>
            <span style={{ color: "var(--muted)" }}>Approval</span>
            <span className="inline-flex items-center gap-2">
              <span className="font-medium" style={{ color: APPROVAL_COLOR[campaign.approvalStatus] }}>
                {APPROVAL_LABEL[campaign.approvalStatus] ?? campaign.approvalStatus}
              </span>
              {canApprove && (
                <span className="inline-flex items-center gap-2">
                  {campaign.approvalStatus !== "approved" && (
                    <button onClick={() => handleSetApproval("approved")} disabled={approvalSaving} className="text-xs underline disabled:opacity-50" style={{ color: "var(--accent)" }}>
                      Approve
                    </button>
                  )}
                  {campaign.approvalStatus !== "rejected" && (
                    <button onClick={() => handleSetApproval("rejected")} disabled={approvalSaving} className="text-xs underline disabled:opacity-50" style={{ color: "var(--muted)" }}>
                      Reject
                    </button>
                  )}
                </span>
              )}
            </span>
          </div>
          {approvalError && <span className="text-xs -mt-3" style={{ color: "var(--alert)" }}>{approvalError}</span>}

          {anomaly && (
            <div className="flex items-start gap-2 text-xs" style={{ background: "var(--base)", border: "1px solid var(--line)", borderRadius: 8, padding: 8 }}>
              <AlertTriangle size={14} style={{ color: "var(--alert)", flexShrink: 0, marginTop: 1 }} />
              <span>
                Flagged as an outlier against its category peers — <span className="font-medium">{anomaly.reason}</span>.
              </span>
            </div>
          )}

          <div className="flex flex-col gap-2" style={{ paddingTop: 4 }}>
            <div className="flex items-center justify-between">
              <span style={{ color: "var(--muted)" }}>Creatives</span>
              {canAddCreative && !addingCreative && (
                <button onClick={() => setAddingCreative(true)} className="text-xs underline" style={{ color: "var(--accent)" }}>
                  Add
                </button>
              )}
            </div>

            {creatives.map((c) => (
              <div key={c.id} className="flex items-center justify-between text-xs" style={{ borderBottom: "1px solid var(--line)", paddingBottom: 6 }}>
                <span className="inline-flex items-center gap-1.5">
                  <span className="ctp-tag px-1.5 py-0.5" style={{ fontSize: 10 }}>{c.creativeType}</span>
                  {c.headline}
                </span>
                <span style={{ color: "var(--muted)" }}>{c.reach.toLocaleString("en-IN")} · {fmtMoney(c.spend)}</span>
              </div>
            ))}
            {creatives.length === 0 && !addingCreative && (
              <span className="text-xs" style={{ color: "var(--muted)" }}>No creatives tracked yet.</span>
            )}

            {addingCreative && (
              <div className="flex flex-col gap-2 mt-1">
                <input
                  autoFocus value={creativeHeadline} onChange={(e) => setCreativeHeadline(e.target.value)}
                  placeholder="Headline" className="ctp-select px-2 py-1 text-xs"
                />
                <div className="flex items-center gap-2">
                  <select value={creativeType} onChange={(e) => setCreativeType(e.target.value)} className="ctp-select px-2 py-1 text-xs">
                    {CREATIVE_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
                  </select>
                  <input
                    type="number" min="0" value={creativeReach} onChange={(e) => setCreativeReach(e.target.value)}
                    placeholder="Reach" className="ctp-select px-2 py-1 text-xs flex-1"
                  />
                  <input
                    type="number" min="0" value={creativeSpend} onChange={(e) => setCreativeSpend(e.target.value)}
                    placeholder="Spend (₹)" className="ctp-select px-2 py-1 text-xs flex-1"
                  />
                </div>
                <div className="flex items-center gap-2">
                  <button onClick={handleAddCreative} disabled={creativeSaving} className="text-xs font-medium px-2 py-1 disabled:opacity-50" style={{ background: "var(--ink)", color: "var(--base)" }}>
                    {creativeSaving ? "Saving…" : "Save"}
                  </button>
                  <button onClick={() => setAddingCreative(false)} className="text-xs" style={{ color: "var(--muted)" }}>Cancel</button>
                </div>
                {creativeError && <span className="text-xs" style={{ color: "var(--alert)" }}>{creativeError}</span>}
              </div>
            )}
          </div>
        </div>
        <button
          onClick={() => onFocusSubject(campaign.subject)}
          className="w-full mt-6 py-2 text-xs font-medium"
          style={{ background: "var(--ink)", color: "var(--base)" }}
        >
          View all {campaign.subject} campaigns
        </button>
      </div>
    </>
  );
}
