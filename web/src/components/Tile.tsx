import { useNavigate } from "react-router-dom";
import type { Campaign } from "../api/types";
import { Pill } from "./Pill";
import { adTypeStripeStyle } from "./AdTypeTag";
import { useDecision } from "../api/campaigns";
import "./Tile.css";

export function Tile({
  campaign,
  stat,
  statLabel,
  showDecide = false,
}: {
  campaign: Campaign;
  stat: string;
  /** What the number means — an unlabelled "₹18.0L" tells you nothing. */
  statLabel: string;
  /** Renders approve/reject inline so a call can be made without leaving Overview. */
  showDecide?: boolean;
}) {
  const navigate = useNavigate();
  const decision = useDecision();

  return (
    <div
      className="tile card"
      style={adTypeStripeStyle(campaign.adType)}
      role="link"
      tabIndex={0}
      onClick={() => navigate(`/campaigns/${campaign.id}`)}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          navigate(`/campaigns/${campaign.id}`);
        }
      }}
    >
      <div className="tile-art">
        <span className="tile-initials">{campaign.initials}</span>
        <span className="tile-status">
          <Pill status={campaign.status} />
        </span>
        {campaign.flagReason && <span className="tile-flag-dot" aria-label={`Flagged: ${campaign.flagReason}`} title={campaign.flagReason} />}
      </div>

      <div className="tile-body">
        <div className="tile-name truncate">{campaign.name}</div>
        <div className="tile-subtitle truncate">
          {campaign.role || campaign.category} · {campaign.region}
        </div>
      </div>

      {campaign.flagReason && <div className="tile-flag-reason truncate">{campaign.flagReason}</div>}

      <div className="tile-stat-block">
        <span className="tile-stat-label">{statLabel}</span>
        <span className="tile-stat mono">{stat}</span>
      </div>

      {showDecide && (
        <div className="tile-actions" onClick={(e) => e.stopPropagation()}>
          <button
            type="button"
            className="btn btn-sm btn-primary tile-action-btn"
            disabled={decision.isPending}
            onClick={() => decision.mutate({ id: campaign.id, action: "approve" })}
          >
            Approve
          </button>
          <button
            type="button"
            className="btn btn-sm btn-danger tile-action-btn"
            disabled={decision.isPending}
            onClick={() => decision.mutate({ id: campaign.id, action: "reject" })}
          >
            Reject
          </button>
        </div>
      )}
    </div>
  );
}
