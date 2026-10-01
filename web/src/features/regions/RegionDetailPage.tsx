import { Link, useNavigate, useParams } from "react-router-dom";
import { useRegionDetail } from "../../api/regions";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonBlock } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import { StackedBar } from "../../components/StackedBar";
import { BarChart } from "../../components/BarChart";
import { formatMoney, formatPct, formatReach } from "../../lib/format";
import "./RegionDetailPage.css";

export function RegionDetailPage() {
  const { id = "" } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data, isLoading, isError } = useRegionDetail(id);

  useSetBreadcrumbs([{ label: "Regions", href: "/regions" }, { label: id }]);

  if (isLoading) return <SkeletonBlock height={400} />;

  if (isError || !data) {
    return (
      <EmptyState
        title="Region not found"
        description="This region doesn't exist, or the link is out of date."
        action={
          <Link to="/regions" className="btn btn-primary">
            Back to Regions
          </Link>
        }
      />
    );
  }

  const campaignRows = data.campaigns.map((c) => ({ label: c.name, value: c.reach, sublabel: c.category }));
  const categoryRows = data.categoryBreakdown.map((c) => ({ label: c.category, value: c.reach, sublabel: `n=${c.count}` }));

  return (
    <div className="region-detail-page">
      <div className="page-header">
        <h1>{data.region}</h1>
      </div>

      <div className="region-stat-rail">
        <div className="stat-cell">
          <div className="stat-cell-label">Campaigns</div>
          <div className="stat-cell-value mono">{data.count}</div>
        </div>
        <div className="stat-cell">
          <div className="stat-cell-label">Reach</div>
          <div className="stat-cell-value mono">{formatReach(data.totalReach)}</div>
        </div>
        <div className="stat-cell">
          <div className="stat-cell-label">Spend</div>
          <div className="stat-cell-value mono">{formatMoney(data.totalSpend)}</div>
        </div>
        <div className="stat-cell">
          <div className="stat-cell-label">Share of tracked budget</div>
          <div className="stat-cell-value mono">{formatPct(data.shareOfBudget)}</div>
        </div>
        <div className="stat-cell">
          <div className="stat-cell-label">Pending</div>
          <div className="stat-cell-value mono">{data.pendingCount}</div>
        </div>
      </div>

      <div className="region-detail-body">
        <div className="card region-panel">
          <h4>Campaigns by reach</h4>
          <BarChart rows={campaignRows} formatValue={formatReach} onRowClick={(label) => {
            const match = data.campaigns.find((c) => c.name === label);
            if (match) navigate(`/campaigns/${match.id}`);
          }} />
        </div>

        <div className="region-detail-side">
          <div className="card region-panel">
            <h4>Ad type split</h4>
            <StackedBar split={data.adTypeSplit} />
          </div>
          <div className="card region-panel">
            <h4>Category breakdown</h4>
            <BarChart rows={categoryRows} formatValue={formatReach} />
          </div>
        </div>
      </div>
    </div>
  );
}
