import { Link } from "react-router-dom";
import { useRegions } from "../../api/regions";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonRows } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import { StackedBar } from "../../components/StackedBar";
import { formatMoney, formatReach } from "../../lib/format";
import "./RegionsPage.css";

export function RegionsPage() {
  useSetBreadcrumbs([{ label: "Regions" }]);
  const { data, isLoading } = useRegions();
  const items = data?.items ?? [];

  return (
    <div className="regions-page">
      <div className="page-header">
        <h1>Regions</h1>
      </div>

      {isLoading ? (
        <SkeletonRows count={4} height={160} />
      ) : items.length === 0 ? (
        <EmptyState title="No regions yet" />
      ) : (
        <div className="regions-grid">
          {items.map((r) => (
            <Link key={r.region} to={`/regions/${encodeURIComponent(r.region)}`} className="region-card card card-hover">
              <h3>{r.region}</h3>
              <div className="region-card-stats">
                <div>
                  <div className="region-card-stat-label">Campaigns</div>
                  <div className="region-card-stat-value mono">{r.count}</div>
                </div>
                <div>
                  <div className="region-card-stat-label">Live</div>
                  <div className="region-card-stat-value mono">{r.liveCount}</div>
                </div>
                <div>
                  <div className="region-card-stat-label">Reach</div>
                  <div className="region-card-stat-value mono">{formatReach(r.totalReach)}</div>
                </div>
                <div>
                  <div className="region-card-stat-label">Spend</div>
                  <div className="region-card-stat-value mono">{formatMoney(r.totalSpend)}</div>
                </div>
              </div>
              <StackedBar split={r.adTypeSplit} />
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
