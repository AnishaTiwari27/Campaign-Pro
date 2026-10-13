import { useNavigate } from "react-router-dom";
import type { Campaign } from "../api/types";
import { Pill } from "./Pill";
import { adTypeStripeStyle } from "./AdTypeTag";
import { SubjectImage } from "./SubjectImage";
import { useDecision } from "../api/campaigns";
import { useSession } from "../api/auth";
import { useCreatorPhotos } from "../api/creatorPhotos";
import { approvalBlockOf } from "../lib/metrics";
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
  const creatorPhotos = useCreatorPhotos();
  const decision = useDecision();
  const { data: me } = useSession();
  // The server refuses to approve an over-budget campaign, so the inline
  // Approve is offered only where it would actually succeed.
  const approvalBlock = approvalBlockOf(campaign.spend, campaign.budget);

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
      {/* The art is the advertisement: its headline, its format, and the
          face fronting it — not a portrait floating on a gradient. The
          headline is the campaign's highest-reaching asset. */}
      <div className="tile-art">
        <SubjectImage
          name={campaign.name}
          imageUrl={campaign.creatorId ? creatorPhotos[campaign.creatorId] : undefined}
          initials={campaign.initials}
          kind={campaign.subjectType}
          domain={campaign.subjectType === "brand" ? campaign.brandDomain : undefined}
          seed={campaign.creatorId ?? campaign.brandDomain ?? campaign.name}
          size={30}
          className="tile-image"
        />
        {campaign.lead && (
          <span className="tile-creative">
            <span className="tile-creative-copy">{campaign.lead.headline}</span>
            <span className="tile-creative-meta">
              {campaign.lead.kind}
              {campaign.lead.duration && campaign.lead.duration !== "static" && campaign.lead.duration !== "article"
                ? ` · ${campaign.lead.duration}`
                : ""}
              {campaign.lead.festival ? ` · ${campaign.lead.festival}` : ""}
            </span>
          </span>
        )}
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

      {showDecide && me?.canApprove && (
        <div className="tile-actions" onClick={(e) => e.stopPropagation()}>
          <button
            type="button"
            className="btn btn-sm btn-primary tile-action-btn"
            disabled={decision.isPending || approvalBlock !== null}
            title={approvalBlock ? `${approvalBlock} — raise the budget or pause the campaign first` : undefined}
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
