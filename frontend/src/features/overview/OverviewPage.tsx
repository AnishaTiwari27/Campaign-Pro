import { Link } from "react-router-dom";
import { useOverview, useBenchmark } from "../../api/overview";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonBlock } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import { Ribbon } from "../../components/Ribbon";
import { Tile } from "../../components/Tile";
import { KpiTile } from "../../components/KpiTile";
import { exportUrl } from "../../api/client";
import { formatMoney, formatPct, formatReach } from "../../lib/format";
import { InFocus } from "./InFocus";
import "./OverviewPage.css";

export function OverviewPage() {
  useSetBreadcrumbs([{ label: "Overview" }]);

  const { data, isLoading } = useOverview();
  const { data: benchmark } = useBenchmark();

  if (isLoading || !data) {
    return <SkeletonBlock height={500} />;
  }

  const hasAnything =
    data.needsDecision.length + data.flagged.length + data.movers.length + data.people.length > 0 ||
    data.spotlightCandidates.length > 0;

  return (
    <div className="overview-page">
      <div className="page-header">
        <div>
          <h1>What needs you today</h1>
          <p className="overview-lede">
            {data.pendingCount} campaign{data.pendingCount === 1 ? "" : "s"} waiting on a decision, {data.flaggedCount} flagged by
            anomaly detection.
          </p>
        </div>
        <div className="page-header-actions">
          <a className="btn" href={exportUrl({})}>
            Export CSV
          </a>
          {data.pendingCount > 0 && (
            <Link to="/approvals" className="btn btn-primary">
              Review {data.pendingCount} pending
            </Link>
          )}
        </div>
      </div>

      {!hasAnything ? (
        <EmptyState title="All quiet" description="No campaigns are waiting on you or flagged right now." />
      ) : (
        <>
          <InFocus candidates={data.spotlightCandidates} medAll={data.medAll} benchmarks={benchmark?.categories} />

          <div className="overview-ribbons">
            <Ribbon
              title="Your call"
              subtitle={`Approve or reject — biggest spend first · ${formatMoney(data.pendingSpend)} unreviewed`}
              count={data.needsDecision.length}
              accent="warn"
              action={
                data.needsDecision.length > 0 ? (
                  <Link to="/approvals" className="btn btn-sm">
                    Open queue
                  </Link>
                ) : undefined
              }
              emptyMessage="Inbox zero — nothing is waiting on you."
            >
              {data.needsDecision.map((c) => (
                <Tile key={c.id} campaign={c} statLabel="Spend to review" stat={formatMoney(c.spend)} showDecide />
              ))}
            </Ribbon>

            <Ribbon
              title="Off track"
              subtitle="Anomaly detection flagged these — overspending, or delivery well off the norm"
              count={data.flagged.length}
              accent="crit"
              emptyMessage="Nothing is flagged. Every campaign is pacing and delivering normally."
            >
              {data.flagged.map((c) => (
                <Tile key={c.id} campaign={c} statLabel="Reach" stat={formatReach(c.reach)} />
              ))}
            </Ribbon>

            <Ribbon
              title="Outperformers"
              subtitle={`Reach at least 1.3× the all-campaign median of ${formatReach(data.medAll)}`}
              count={data.movers.length}
              accent="good"
              emptyMessage="No campaign is beating the median by enough to call out yet."
            >
              {data.movers.map((c) => (
                <Tile key={c.id} campaign={c} statLabel="vs median" stat={`${c.index.toFixed(1)}x`} />
              ))}
            </Ribbon>

            {/* Named for what it is. "Faces, not logos" read as a design
                slogan; this ribbon is the endorsement book — the campaigns
                a person fronts rather than the brand, which is the half of
                the market this product exists to measure. */}
            <Ribbon
              title="Influencer & celebrity campaigns"
              subtitle="Fronted by a creator, actor, cricketer or singer — not by the brand itself"
              count={data.people.length}
              accent="accent"
              emptyMessage="No influencer or celebrity campaigns are running."
            >
              {data.people.map((c) => (
                <Tile key={c.id} campaign={c} statLabel="Reach" stat={formatReach(c.reach)} />
              ))}
            </Ribbon>
          </div>
        </>
      )}

      <div className="overview-kpis">
        <KpiTile
          label="Live campaigns"
          value={String(data.liveCount)}
          growth={data.liveSparkline.growth}
          sparkValues={data.liveSparkline.points.map((p) => p.LiveCount)}
          to="/campaigns?status=live"
        />
        <KpiTile
          label="Waiting on you"
          value={String(data.pendingCount)}
          growth={data.pendingSparkline.growth}
          sparkValues={data.pendingSparkline.points.map((p) => p.LiveCount)}
          to="/approvals"
          accent="var(--warn)"
        />
        <KpiTile
          label="Reach live"
          value={formatReach(data.reachLive)}
          growth={data.reachSparkline.growth}
          sparkValues={data.reachSparkline.points.map((p) => p.Reach)}
          to="/campaigns?sort=reach&dir=desc"
        />
        <KpiTile
          label={`Spend in window · ${formatPct(data.spendPctBudget)} of approved`}
          value={formatMoney(data.spendWindow)}
          growth={data.spendSparkline.growth}
          sparkValues={data.spendSparkline.points.map((p) => p.Spend)}
          to="/regions"
        />
      </div>
    </div>
  );
}
