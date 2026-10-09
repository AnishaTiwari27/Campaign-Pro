import { useNavigate } from "react-router-dom";
import { useCreators } from "../../api/creators";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonRows } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import { formatIndex, formatMoney, formatReach } from "../../lib/format";
import { TierBadge, IndexBar, ConsistencyDots, formatFollowers } from "./CreatorBits";
import { SubjectImage } from "../../components/SubjectImage";
import "./CreatorsPage.css";

export function CreatorsPage() {
  useSetBreadcrumbs([{ label: "Creators" }]);
  const { data, isLoading } = useCreators();
  const navigate = useNavigate();
  const items = data?.items ?? [];

  return (
    <div className="creators-page">
      <div className="page-header">
        <div>
          <h1>Creators</h1>
          <p className="creators-lede">
            Everyone who fronts a campaign — influencers, actors, cricketers, singers and founders. Ranked by how each
            performs <strong>for their own tier</strong>, not by follower count: a mega account doing big numbers is
            expected; a micro creator beating their tier median is a find.
          </p>
        </div>
      </div>

      {isLoading ? (
        <SkeletonRows count={6} height={72} />
      ) : items.length === 0 ? (
        <EmptyState title="No creators yet" description="Creator-led campaigns will appear here." />
      ) : (
        <div className="creator-list">
          <div className="creator-list-head">
            <span>Creator</span>
            <span>Tier</span>
            <span className="creator-col-num">vs tier</span>
            <span className="creator-col-num">Consistency</span>
            <span className="creator-col-num">Audience reach</span>
            <span className="creator-col-num">Cost / lakh</span>
            <span className="creator-col-num">Campaigns</span>
          </div>

          {items.map((c) => (
            <button key={c.id} type="button" className="creator-row" onClick={() => navigate(`/creators/${c.id}`)}>
              <span className="creator-identity">
                <SubjectImage name={c.name} initials={c.initials} kind="person" seed={c.id} size={34} />
                <span className="creator-identity-text">
                  <span className="creator-name">{c.name}</span>
                  <span className="creator-meta">
                    {c.role} · {c.region} · {c.languages.join(", ")}
                  </span>
                </span>
              </span>

              <span className="creator-tier-cell">
                <TierBadge tier={c.tier} label={c.tierLabel} />
                <span className="creator-followers mono">{formatFollowers(c.followers)}</span>
              </span>

              <span className="creator-col-num">
                <IndexBar index={c.tierIndex} />
              </span>

              <span className="creator-col-num">
                <ConsistencyDots score={c.consistency} n={c.consistencyN} />
              </span>

              <span className="creator-col-num mono creator-audience">
                {Math.round(c.audienceReachPct)}%
                {c.audienceReachPct >= 200 && <span className="creator-travel" title="Reaching well beyond their own following">travels</span>}
              </span>

              <span className="creator-col-num mono">{formatMoney(c.costPerLakh)}</span>

              <span className="creator-col-num mono creator-campaign-count">
                {c.campaigns}
                {c.flagged > 0 && <span className="creator-flag-count" title={`${c.flagged} flagged`}>{c.flagged}⚑</span>}
              </span>
            </button>
          ))}
        </div>
      )}

      {items.length > 0 && (
        <p className="creators-foot">
          Tier medians are computed across running campaigns only. Index of {formatIndex(1)} means a creator delivers exactly
          the median reach for their tier; consistency scores how repeatable that is across their{" "}
          {items.reduce((n, c) => n + c.campaigns, 0)} tracked campaigns, totalling {formatReach(items.reduce((n, c) => n + c.totalReach, 0))} reach.
        </p>
      )}
    </div>
  );
}
