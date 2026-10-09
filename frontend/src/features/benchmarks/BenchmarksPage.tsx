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

  // The untagged row is the baseline, and is shown as context rather than
  // listed among the festivals — comparing evergreen work with itself is
  // not a finding.
  const all = data.festivals ?? [];
  const baselineRow = all.find((f) => f.festival === "");
  const baseline = baselineRow?.avgReach ?? 0;
  const baselineN = baselineRow?.creatives ?? 0;
  const festivals = all.filter((f) => f.festival !== "");

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

      {/* Festival lift. The question is never "what did Diwali do" — a
          brand puts its best assets behind Diwali whether or not the
          festival helps. It is what Diwali did that evergreen work did
          not, so every row is scored against the untagged baseline. */}
      {festivals.length > 0 && (
        <div className="card benchmarks-panel">
          <h4>Festival lift</h4>
          <p className="benchmarks-panel-hint">
            Average reach of creatives cut for each festival, against evergreen work
            {baseline > 0 && <> (<strong className="mono">{formatReach(baseline)}</strong> across {baselineN} creatives)</>}.
            Above 1.0× means the festival cut out-reached ordinary work.
          </p>
          <ul className="benchmarks-list">
            {festivals.map((f) => (
              <li key={f.festival} className="benchmarks-list-row">
                <span>
                  <span className="benchmarks-list-category">{f.festival}</span>
                  <span className="benchmarks-list-sub"> · {f.creatives} creative{f.creatives === 1 ? "" : "s"}</span>
                </span>
                <span>
                  <span className="mono benchmarks-festival-reach">{formatReach(f.avgReach)}</span>
                  <span className={`mono benchmarks-festival-lift${f.lift >= 1.1 ? " benchmarks-festival-lift-up" : f.lift > 0 && f.lift <= 0.9 ? " benchmarks-festival-lift-down" : ""}`}>
                    {f.lift > 0 ? `${formatIndex(f.lift)} vs evergreen` : "baseline"}
                  </span>
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
