import { Link, useNavigate, useParams } from "react-router-dom";
import { useCreatorDetail } from "../../api/creators";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonBlock } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import { BarChart } from "../../components/BarChart";
import { Pill } from "../../components/Pill";
import { formatMoney, formatReach } from "../../lib/format";
import { ConsistencyDots, IndexBar, TierBadge, formatFollowers } from "./CreatorBits";
import "./CreatorDetailPage.css";

export function CreatorDetailPage() {
  const { id = "" } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data, isLoading, isError } = useCreatorDetail(id);

  useSetBreadcrumbs([{ label: "Creators", href: "/creators" }, { label: data?.name ?? "…" }]);

  if (isLoading) return <SkeletonBlock height={400} />;

  if (isError || !data) {
    return (
      <EmptyState
        title="Creator not found"
        description="This creator doesn't exist, or the link is out of date."
        action={
          <Link to="/creators" className="btn btn-primary">
            Back to Creators
          </Link>
        }
      />
    );
  }

  const verdict =
    data.tierIndex >= 1.3
      ? `Over-delivering for ${data.tierLabel.toLowerCase()} tier`
      : data.tierIndex >= 0.9
        ? `About typical for ${data.tierLabel.toLowerCase()} tier`
        : `Under-delivering for ${data.tierLabel.toLowerCase()} tier`;
  const verdictTone = data.tierIndex >= 1.3 ? "good" : data.tierIndex >= 0.9 ? "neutral" : "crit";

  const langRows = data.languageBreakdown.map((l) => ({
    label: l.language,
    value: l.reach,
    sublabel: `${l.count} creative${l.count === 1 ? "" : "s"}`,
  }));

  return (
    <div className="creator-detail-page">
      <div className="page-header">
        <div className="creator-detail-head">
          <span className="creator-detail-avatar">{data.initials}</span>
          <div>
            <div className="creator-detail-title">
              <h1>{data.name}</h1>
              <TierBadge tier={data.tier} label={data.tierLabel} />
            </div>
            <p className="creator-detail-sub">
              {data.role} · {data.region} · {data.primaryPlatform} · {formatFollowers(data.followers)} followers ·{" "}
              {data.languages.join(", ")}
            </p>
          </div>
        </div>
      </div>

      <div className={`creator-verdict creator-verdict-${verdictTone}`}>
        <div className="creator-verdict-main">
          <IndexBar index={data.tierIndex} />
          <div>
            <div className="creator-verdict-text">{verdict}</div>
            <div className="creator-verdict-detail">
              Averages {formatReach(data.avgReach)} against a {data.tierLabel.toLowerCase()}-tier median of{" "}
              {formatReach(data.tierMedian)}, across {data.tierPeers} creator{data.tierPeers === 1 ? "" : "s"} at this size.
            </div>
          </div>
        </div>
        <div className="creator-verdict-consistency">
          <span className="creator-verdict-label">Consistency</span>
          <ConsistencyDots score={data.consistency} n={data.consistencyN} />
          <span className="mono creator-verdict-pct">{data.consistencyN < 2 ? "" : `${Math.round(data.consistency * 100)}%`}</span>
        </div>
      </div>

      <div className="stat-rail creator-stat-rail">
        <div className="stat-cell">
          <div className="stat-cell-label">Campaigns</div>
          <div className="stat-cell-value mono">{data.campaigns}</div>
          <div className="stat-cell-sub">{data.liveCampaigns} live</div>
        </div>
        <div className="stat-cell">
          <div className="stat-cell-label">Total reach</div>
          <div className="stat-cell-value mono">{formatReach(data.totalReach)}</div>
        </div>
        <div className="stat-cell">
          <div className="stat-cell-label">Total spend</div>
          <div className="stat-cell-value mono">{formatMoney(data.totalSpend)}</div>
        </div>
        <div className="stat-cell">
          <div className="stat-cell-label">Cost per lakh</div>
          <div className="stat-cell-value mono">{formatMoney(data.costPerLakh)}</div>
        </div>
        <div className="stat-cell">
          <div className="stat-cell-label">Audience reach</div>
          <div className="stat-cell-value mono">{Math.round(data.audienceReachPct)}%</div>
          <div className="stat-cell-sub">{data.audienceReachPct >= 100 ? "beyond following" : "of following"}</div>
        </div>
        <div className="stat-cell">
          <div className="stat-cell-label">Flagged</div>
          <div className={`stat-cell-value mono${data.flagged > 0 ? " stat-cell-value-crit" : ""}`}>{data.flagged}</div>
        </div>
      </div>

      <div className="creator-detail-body">
        <div className="card creator-panel">
          <h4>Campaigns</h4>
          <div className="creator-campaign-list">
            {data.campaignRows.map((c) => (
              <button
                key={c.id}
                type="button"
                className="creator-campaign-row"
                onClick={() => navigate(`/campaigns/${c.id}`)}
              >
                <span className="creator-campaign-name truncate">{c.name}</span>
                <Pill status={c.status} />
                <span className="mono creator-campaign-reach">{formatReach(c.reach)}</span>
                <span className="mono creator-campaign-spend">{formatMoney(c.spend)}</span>
              </button>
            ))}
          </div>
        </div>

        <div className="creator-detail-side">
          <div className="card creator-panel">
            <h4>Reach by creative language</h4>
            {langRows.length === 0 ? (
              <p className="side-panel-muted">No creatives have been language-tagged yet.</p>
            ) : (
              <>
                <BarChart rows={langRows} formatValue={formatReach} />
                <p className="creator-panel-note">
                  Language mix is the axis most global ad-intelligence tools don't break out.
                </p>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
