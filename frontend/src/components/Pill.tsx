import type { CampaignStatus } from "../api/types";

const LABELS: Record<CampaignStatus, string> = {
  live: "Live",
  ended: "Ended",
  scheduled: "Scheduled",
  paused: "Paused",
};

export function Pill({ status }: { status: CampaignStatus }) {
  return <span className={`pill pill-${status}`}>{LABELS[status]}</span>;
}
