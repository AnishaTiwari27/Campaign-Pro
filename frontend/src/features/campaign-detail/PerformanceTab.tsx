import { AreaChart } from "../../components/AreaChart";
import { reachCurve } from "../../lib/metrics";
import { formatMoney, formatReach } from "../../lib/format";
import type { CampaignDetail } from "../../api/types";
import "./PerformanceTab.css";

export function PerformanceTab({ campaign, categoryMedian }: { campaign: CampaignDetail; categoryMedian?: number }) {
  const curve = reachCurve(campaign.reach, campaign.curveShape);
  const points = curve.map((v, i) => ({
    label: `Day ${Math.round((i / 7) * campaign.daysRunning)}`,
    value: v,
  }));

  const deltas = curve.slice(1).map((v, i) => v - curve[i]);
  const intervalDays = campaign.daysRunning / 7;
  const bestDayReach = intervalDays > 0 ? Math.max(...deltas) / intervalDays : 0;
  const avgPerDay = campaign.daysRunning > 0 ? campaign.reach / campaign.daysRunning : 0;
  const costPer1L = campaign.reach > 0 ? campaign.spend / campaign.reach : 0;
  const estImpressions = campaign.reach * 100000 * campaign.frequency;

  return (
    <div className="performance-tab">
      <AreaChart points={points} medianValue={categoryMedian} formatValue={(v) => formatReach(v)} />

      <div className="performance-insights card">
        <h4>How it got there</h4>
        <div className="performance-insights-grid">
          <div>
            <div className="performance-insight-label">Best single day</div>
            <div className="performance-insight-value mono">{formatReach(bestDayReach)}</div>
          </div>
          <div>
            <div className="performance-insight-label">Average per day</div>
            <div className="performance-insight-value mono">{formatReach(avgPerDay)}</div>
          </div>
          <div>
            <div className="performance-insight-label">Cost per 1L reach</div>
            <div className="performance-insight-value mono">{formatMoney(costPer1L)}</div>
          </div>
          <div>
            <div className="performance-insight-label">Estimated impressions</div>
            <div className="performance-insight-value mono">{Math.round(estImpressions).toLocaleString("en-IN")}</div>
          </div>
        </div>
      </div>
    </div>
  );
}
