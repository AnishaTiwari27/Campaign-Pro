import "./BarChart.css";

export interface BarChartRow {
  label: string;
  value: number;
  markerValue?: number;
  sublabel?: string;
}

// A horizontal bar chart: bar length is `value`, an optional thin marker
// shows `markerValue` (e.g. the category's top campaign) against the same
// scale. Rows are buttons so a category can be clicked through to Campaigns.
export function BarChart({
  rows,
  formatValue,
  onRowClick,
}: {
  rows: BarChartRow[];
  formatValue: (v: number) => string;
  onRowClick?: (label: string) => void;
}) {
  const max = Math.max(...rows.map((r) => Math.max(r.value, r.markerValue ?? 0)), 1);

  return (
    <div className="bar-chart">
      {rows.map((r) => {
        const widthPct = (r.value / max) * 100;
        const markerPct = r.markerValue !== undefined ? (r.markerValue / max) * 100 : null;
        return (
          <button
            key={r.label}
            type="button"
            className="bar-chart-row"
            onClick={() => onRowClick?.(r.label)}
            disabled={!onRowClick}
          >
            <span className="bar-chart-label truncate">
              {r.label}
              {r.sublabel && <span className="bar-chart-sublabel"> {r.sublabel}</span>}
            </span>
            <span className="bar-chart-track">
              <span className="bar-chart-fill" style={{ width: `${widthPct}%` }} />
              {markerPct !== null && <span className="bar-chart-marker" style={{ left: `${markerPct}%` }} />}
            </span>
            <span className="bar-chart-value mono">{formatValue(r.value)}</span>
          </button>
        );
      })}
    </div>
  );
}
