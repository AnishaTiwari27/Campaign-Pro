import { Link } from "react-router-dom";
import type { Campaign } from "../../api/types";
import { Pill } from "../../components/Pill";
import { ApprovalTag } from "../../components/Tag";
import { adTypeStripeStyle } from "../../components/AdTypeTag";
import { AreaChart } from "../../components/AreaChart";
import { reachCurve } from "../../lib/metrics";
import { formatIndex, formatMoney, formatReach } from "../../lib/format";
import "./SpotlightCard.css";

export function SpotlightCard({ campaign, medAll, categoryMedian }: { campaign: Campaign; medAll: number; categoryMedian?: number }) {
  const curve = reachCurve(campaign.reach, campaign.curveShape);
  const points = curve.map((v, i) => ({ label: `Day ${Math.round((i / 7) * campaign.daysRunning)}`, value: v }));
  const eyebrow = campaign.flagReason
    ? `Flagged ${campaign.adType} · ${campaign.region}`
    : `${campaign.adType} · ${campaign.region}`;

  return (
    <Link to={`/campaigns/${campaign.id}`} className="spotlight-card card card-hover" style={adTypeStripeStyle(campaign.adType)}>
      <span className="spotlight-stripe" />
      <div className="spotlight-main">
        <div className="spotlight-header">
          <div className="spotlight-eyebrow">{eyebrow}</div>
          <div className="spotlight-title-row">
            <h2>{campaign.name}</h2>
            <Pill status={campaign.status} />
            <ApprovalTag approval={campaign.approval} />
          </div>
          <div className="spotlight-subtitle">
            {campaign.role || campaign.category} · {campaign.platform} · Day {campaign.daysRunning}
          </div>
          {campaign.flagReason && <div className="spotlight-flag-reason">{campaign.flagReason}</div>}
        </div>

        <div className="spotlight-stats">
          <div>
            <div className="spotlight-stat-label">Reach</div>
            <div className="spotlight-stat-value mono">{formatReach(campaign.reach)}</div>
          </div>
          <div>
            <div className="spotlight-stat-label">Spend</div>
            <div className="spotlight-stat-value mono">{formatMoney(campaign.spend)}</div>
          </div>
          <div>
            <div className="spotlight-stat-label">All-campaign median</div>
            <div className="spotlight-stat-value mono">{formatReach(medAll)}</div>
          </div>
          <div>
            <div className="spotlight-stat-label">Index</div>
            <div className="spotlight-stat-value spotlight-stat-index mono">{formatIndex(campaign.index)}</div>
          </div>
        </div>
      </div>

      <div className="spotlight-chart">
        <AreaChart points={points} medianValue={categoryMedian} formatValue={formatReach} interactive={false} height={140} />
      </div>
    </Link>
  );
}
