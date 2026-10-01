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
import { SpotlightCard } from "./SpotlightCard";
import "./OverviewPage.css";

export function OverviewPage() {
  useSetBreadcrumbs([{ label: "Overview" }]);

  const { data, isLoading } = useOverview();
  const { data: benchmark } = useBenchmark();

  if (isLoading || !data) {
    return <SkeletonBlock height={500} />;
  }

  const spotlightCategoryMedian = data.spotlight
    ? benchmark?.categories.find((c) => c.category === data.spotlight!.category)?.medianReach
    : undefined;

  const hasAnything =
    data.needsDecision.length + data.flagged.length + data.movers.length + data.people.length > 0 || !!data.spotlight;

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
          {data.spotlight && <SpotlightCard campaign={data.spotlight} medAll={data.medAll} categoryMedian={spotlightCategoryMedian} />}

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

            <Ribbon
              title="Faces, not logos"
              subtitle="Creator- and talent-led campaigns, where the subject is a person"
              count={data.people.length}
              accent="accent"
              emptyMessage="No people-led campaigns are running."
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
