import { useMemo } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useCampaignsList } from "../../api/campaigns";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonBlock } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import { QuadrantChart } from "../../components/QuadrantChart";
import { signalPoints, summarise } from "../../lib/signals";
import { formatIndex, formatMoney } from "../../lib/format";
import "./SignalsPage.css";

// One page of everything. The quadrant is a portfolio view: a paginated
// slice of the fleet would put a campaign in a quadrant whose boundary was
// drawn from a different set, which is worse than no chart.
const PER_PAGE = 500;

/** The three campaigns most worth looking at in a quadrant: the biggest
 *  budgets, because that is where a decision moves the most money. */
const SHOWN_PER_QUADRANT = 3;

export function SignalsPage() {
  useSetBreadcrumbs([{ label: "Signals" }]);
  const navigate = useNavigate();
  const { data, isLoading } = useCampaignsList({ per: PER_PAGE, page: 1 });

  const points = useMemo(() => signalPoints(data?.items ?? []), [data]);
  const quadrants = useMemo(() => summarise(points), [points]);

  if (isLoading || !data) return <SkeletonBlock height={460} />;

  const scale = quadrants.find((q) => q.key === "scale")!;
  const cut = quadrants.find((q) => q.key === "cut")!;
  // Unspent budget behind campaigns already beating the median: the
  // cheapest reach available without asking anyone for more money.
  const headroom = scale.points.reduce(
    (sum, p) => sum + Math.max(0, p.campaign.budget - p.campaign.spend), 0,
  );

  // Said plainly rather than silently truncated, because a quadrant drawn
  // from part of the fleet would be quietly wrong.
  const truncated = data.total > data.items.length;
  // Said out loud rather than quietly dropped: a campaign with no recorded
  // flight length has no plan to be measured against, so it cannot be
  // placed on this chart without inventing one.
  const unplanned = (data.items ?? []).filter(
    (c) => c.status !== "scheduled" && c.paceVsPlan <= 0,
  ).length;

  return (
    <div className="signals-page">
      <div className="page-header">
        <div>
          <h1>Signals</h1>
          <p className="signals-lede">
            Every running campaign plotted by whether it is spending to plan against what it has delivered. Vertical
            line is exactly on plan — its own flight curve, at today's point in the flight; horizontal line is{" "}
            <Link to="/benchmarks">the all-campaign median</Link>. Which quarter a campaign lands in is the decision
            waiting to be made about it.
          </p>
        </div>
      </div>

      {truncated && (
        <p className="signals-warning" role="status">
          Showing {data.items.length} of {data.total} campaigns — the quadrant below is drawn from a partial fleet.
        </p>
      )}

      {points.length === 0 ? (
        <EmptyState
          title="Nothing running yet"
          description="Campaigns appear here once they have started delivering."
        />
      ) : (
        <>
          {/* Stat tiles rather than the KPI tile used elsewhere: that one
              draws a sparkline, and there is no time series behind these
              numbers to draw. An invented one would be the worst of both. */}
          <div className="signals-stats">
            <div className="card signals-stat">
              <span className="signals-stat-label">Unspent behind winners</span>
              <strong className="signals-stat-value mono">{formatMoney(headroom)}</strong>
              <span className="signals-stat-sub">
                {scale.points.length} campaign{scale.points.length === 1 ? "" : "s"} beating the median with budget left
              </span>
            </div>
            <div className="card signals-stat">
              <span className="signals-stat-label">Spent below the median</span>
              <strong className="signals-stat-value mono">{formatMoney(cut.spend)}</strong>
              <span className="signals-stat-sub">
                {cut.points.length} campaign{cut.points.length === 1 ? "" : "s"} out of budget and under-delivering
              </span>
            </div>
            <div className="card signals-stat">
              <span className="signals-stat-label">Campaigns plotted</span>
              <strong className="signals-stat-value mono">{points.length}</strong>
              <span className="signals-stat-sub">
                {unplanned > 0
                  ? `${unplanned} excluded — no planned flight length recorded`
                  : "Scheduled excluded — nothing delivered yet"}
              </span>
            </div>
          </div>

          <div className="card signals-chart-card">
            <QuadrantChart points={points} onPointClick={(id) => navigate(`/campaigns/${id}`)} />
            <p className="signals-chart-hint">
              Dot size is budget. Select a campaign to open it.
            </p>
          </div>

          <div className="signals-grid">
            {quadrants.map((q) => (
              <section key={q.key} className={`card signals-quadrant signals-quadrant-${q.tone}`}>
                <header className="signals-quadrant-head">
                  <h4>
                    <span className={`signals-dot signals-dot-${q.tone}`} aria-hidden="true" />
                    {q.label}
                  </h4>
                  <span className="mono signals-count">{q.points.length}</span>
                </header>
                <p className="signals-quadrant-blurb">{q.blurb}</p>

                {q.points.length === 0 ? (
                  <p className="signals-quadrant-empty">Nothing here right now.</p>
                ) : (
                  <ul className="signals-list">
                    {[...q.points]
                      .sort((a, b) => b.campaign.budget - a.campaign.budget)
                      .slice(0, SHOWN_PER_QUADRANT)
                      .map((p) => (
                        <li key={p.campaign.id}>
                          <Link to={`/campaigns/${p.campaign.id}`} className="signals-list-name">
                            {p.campaign.name}
                          </Link>
                          <span className="mono signals-list-figures">
                            {formatIndex(p.paceVsPlan)} · {formatIndex(p.index)}
                          </span>
                        </li>
                      ))}
                  </ul>
                )}

                <footer className="signals-quadrant-foot">
                  <span>{formatMoney(q.budget)} budgeted</span>
                  {q.points.length > SHOWN_PER_QUADRANT && (
                    <span>+{q.points.length - SHOWN_PER_QUADRANT} more</span>
                  )}
                </footer>
              </section>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
