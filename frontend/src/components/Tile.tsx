import { Link, useNavigate } from "react-router-dom";
import type { Campaign } from "../api/types";
import { Pill } from "./Pill";
import { adTypeStripeStyle } from "./AdTypeTag";
import { SubjectImage } from "./SubjectImage";
import { useDecision } from "../api/campaigns";
import { useSession } from "../api/auth";
import { useCreatorIndex } from "../api/creatorPhotos";
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
  const creatorIndex = useCreatorIndex();
  const creator = campaign.creatorId ? creatorIndex[campaign.creatorId] : undefined;
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
      {/* The picture is the whole panel and nothing is written across
          it. The ad's headline used to sit over the lower half, which
          put two lines of white text across the face of the person the
          tile is about — the one thing the image is there to show. The
          copy still lives on the campaign and on the creator's page,
          where it has room to be read rather than squinted past. */}
      <div className="tile-art">
        <SubjectImage
          name={campaign.name}
          imageUrl={creator?.imageUrl}
          initials={campaign.initials}
          kind={campaign.subjectType}
          domain={campaign.subjectType === "brand" ? campaign.brandDomain : undefined}
          seed={campaign.creatorId ?? campaign.brandDomain ?? campaign.name}
          fill
        />
        {/* The byline. A tile is one campaign, but the face on it has a
            body of work, and the person is what someone wants next —
            so their name on the picture goes to them, not to this ad. */}
        {creator && (
          <Link
            to={`/creators/${creator.id}`}
            className="tile-byline"
            title={`${creator.name} — ${creator.campaigns} campaign${creator.campaigns === 1 ? "" : "s"}`}
            // The whole tile is a link to the campaign; this one is not it.
            onClick={(e) => e.stopPropagation()}
            onKeyDown={(e) => e.stopPropagation()}
          >
            <span className="tile-byline-name truncate">{creator.name}</span>
            <span className="tile-byline-go" aria-hidden="true">›</span>
          </Link>
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
