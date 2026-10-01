import type { Approval } from "../api/types";

const LABELS: Record<Approval, string> = {
  pending: "Pending",
  approved: "Approved",
  rejected: "Rejected",
};

export function ApprovalTag({ approval }: { approval: Approval }) {
  return <span className={`tag tag-${approval}`}>{LABELS[approval]}</span>;
}

export function FlaggedTag() {
  return <span className="tag tag-flagged">Flagged</span>;
}
