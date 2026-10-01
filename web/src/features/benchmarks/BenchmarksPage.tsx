import { useNavigate } from "react-router-dom";
import { useBenchmark } from "../../api/overview";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonBlock } from "../../components/Skeleton";
import { BarChart } from "../../components/BarChart";
import { formatIndex, formatMoney, formatReach } from "../../lib/format";
import "./BenchmarksPage.css";

export function BenchmarksPage() {
  useSetBreadcrumbs([{ label: "Benchmarks" }]);
  const navigate = useNavigate();
  const { data, isLoading } = useBenchmark();

  if (isLoading || !data) return <SkeletonBlock height={400} />;

  const rows = data.categories.map((c) => ({
    label: c.category,
    value: c.medianReach,
    markerValue: c.topReach,
    sublabel: `n=${c.n}`,
  }));

  return (
    <div className="benchmarks-page">
      <div className="page-header">
        <div>
          <h1>Benchmarks</h1>
          <p className="benchmarks-lede">
            The all-campaign median reach — <strong className="mono">{formatReach(data.medAll)}</strong> right now — is the
            baseline every index in Campaign Tracker Pro is expressed against. A campaign at 1.0x is exactly typical; higher means
            it's outperforming the fleet.
          </p>
        </div>
      </div>

      <div className="card benchmarks-panel">
        <h4>Reach by category</h4>
        <p className="benchmarks-panel-hint">Bar is the category median; the marker is that category's top campaign.</p>
        <BarChart rows={rows} formatValue={formatReach} onRowClick={(category) => navigate(`/campaigns?category=${encodeURIComponent(category)}`)} />
      </div>

      <div className="benchmarks-columns">
        <div className="card benchmarks-panel">
          <h4>Category leaders</h4>
          <ul className="benchmarks-list">
            {data.categories.map((c) => (
              <li key={c.category} className="benchmarks-list-row">
                <span>
                  <span className="benchmarks-list-category">{c.category}</span>
                  <span className="benchmarks-list-sub"> · {c.topCampaign}</span>
                </span>
                <span className="mono">{formatIndex(c.medianReach > 0 ? c.topReach / c.medianReach : 0)}</span>
              </li>
            ))}
          </ul>
        </div>

        <div className="card benchmarks-panel">
          <h4>Spend per category</h4>
          <ul className="benchmarks-list">
            {data.categories.map((c) => (
              <li key={c.category} className="benchmarks-list-row">
                <span className="benchmarks-list-category">{c.category}</span>
                <span className="mono">{formatMoney(c.totalSpend)}</span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  );
}
