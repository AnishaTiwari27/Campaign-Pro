import { Link, useNavigate } from "react-router-dom";
import type { Campaign } from "../api/types";
import { Pill } from "./Pill";
import { adTypeStripeStyle } from "./AdTypeTag";
import { SubjectImage } from "./SubjectImage";
import { useDecision } from "../api/campaigns";
import { useSession } from "../api/auth";
import { useCreatorIndex } from "../api/creatorPhotos";
import { adFilmFor, adFilmThumb, filmCropStyle } from "../data/adFilms";
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
  // The same film the spotlight plays, as a still. On a creator
  // campaign this is not a departure from showing the person — the
  // film chosen for them is one they front, so the frame is their face
  // in the brand's actual ad rather than a portrait beside a logo.
  const film = adFilmFor(campaign.id);

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
      {/* Nothing is drawn over the picture at all — no headline, no
          byline, no status badge. Everything that was on it had to be
          legible against an arbitrary photograph, which meant a scrim,
          and the scrim dimmed the face the tile exists to show. It all
          reads better as text under the image, where it needs no
          treatment to be readable. */}
      <div className="tile-art">
        {film ? (
          // Two copies of the same still: one blurred and cropped to
          // fill the box, the real one contained on top. The tile is
          // taller than 16:9, so cropping the still to fit was cutting
          // the ends off the ad's own copy — "Diwali gifts, lights…"
          // arrived as "li gifts, lights". Nothing is lost this way and
          // the box is still filled.
          <>
            <img
              className="tile-film-fill"
              src={adFilmThumb(film.id, "tile")}
              alt=""
              aria-hidden="true"
              loading="lazy"
              // Cropped too. Blurring the uncropped frame put the
              // film's own black bars back into the backdrop, so the
              // band around the still came out black rather than
              // carrying its colours. 1.18 is the bleed the CSS would
              // otherwise apply on its own.
              style={filmCropStyle(film, "fill", 1.18)}
            />
            <img
              className="tile-film"
              src={adFilmThumb(film.id, "tile")}
              alt=""
              loading="lazy"
              style={filmCropStyle(film, "clip")}
            />
          </>
        ) : (
          <SubjectImage
            name={campaign.name}
            imageUrl={creator?.imageUrl}
            initials={campaign.initials}
            kind={campaign.subjectType}
            domain={campaign.subjectType === "brand" ? campaign.brandDomain : undefined}
            seed={campaign.creatorId ?? campaign.brandDomain ?? campaign.name}
            fill
          />
        )}
      </div>

      <div className="tile-body">
        <div className="tile-name truncate">{campaign.name}</div>
        <div className="tile-subtitle truncate">
          {campaign.role || campaign.category} · {campaign.region}
        </div>
      </div>

      {/* Status and the route to the person, on one row. The flag dot
          that used to sit on the image is gone with them: the reason is
          spelled out just below, so the dot only ever repeated it. */}
      <div className="tile-meta">
        <Pill status={campaign.status} />
        {creator && (
          <Link
            to={`/creators/${creator.id}`}
            className="tile-profile"
            title={`${creator.name} — ${creator.campaigns} campaign${creator.campaigns === 1 ? "" : "s"}`}
            // The whole tile is a link to the campaign; this one is not it.
            onClick={(e) => e.stopPropagation()}
            onKeyDown={(e) => e.stopPropagation()}
          >
            View profile <span aria-hidden="true">›</span>
          </Link>
        )}
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
