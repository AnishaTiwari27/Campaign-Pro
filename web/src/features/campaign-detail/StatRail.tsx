import type { Campaign } from "../../api/types";
import { formatCPM, formatMoney, formatPct, formatReach } from "../../lib/format";
import "./StatRail.css";

export function StatRail({ campaign }: { campaign: Campaign }) {
  const overBudget = campaign.pace >= 100;
  return (
    <div className="stat-rail">
      <div className="stat-cell">
        <div className="stat-cell-label">Reach</div>
        <div className="stat-cell-value mono">{formatReach(campaign.reach)}</div>
      </div>
      <div className="stat-cell">
        <div className="stat-cell-label">Spend</div>
        <div className="stat-cell-value mono">{formatMoney(campaign.spend)}</div>
        <div className="stat-cell-sub">of {formatMoney(campaign.budget)}</div>
      </div>
      <div className="stat-cell">
        <div className="stat-cell-label">Budget used</div>
        <div className={`stat-cell-value mono${overBudget ? " stat-cell-value-crit" : ""}`}>{formatPct(campaign.pace)}</div>
      </div>
      <div className="stat-cell">
        <div className="stat-cell-label">CPM</div>
        <div className="stat-cell-value mono">{formatCPM(campaign.cpm)}</div>
      </div>
      <div className="stat-cell">
        <div className="stat-cell-label">Frequency</div>
        <div className="stat-cell-value mono">{campaign.frequency.toFixed(1)}x</div>
      </div>
      <div className="stat-cell">
        <div className="stat-cell-label">Flight</div>
        <div className="stat-cell-value mono">{campaign.daysRunning}d</div>
        <div className="stat-cell-sub">{campaign.status}</div>
      </div>
    </div>
  );
}
