import { paceClassOf } from "../lib/metrics";
import { formatPct } from "../lib/format";
import "./Meter.css";

// A pace bar: fill width is pace% clamped to 100, colored by pace class,
// with an optional marker at the 100% budget line so "over" reads clearly
// even when the fill itself is clipped.
export function PaceMeter({ pace, showLabel = true }: { pace: number; showLabel?: boolean }) {
  const cls = paceClassOf(pace);
  const width = Math.min(100, Math.max(0, pace));
  return (
    <div className="meter-row">
      <div className={`meter meter-${cls}`} role="img" aria-label={`Budget pace ${formatPct(pace)}`}>
        <div className="meter-fill" style={{ width: `${width}%` }} />
      </div>
      {showLabel && <span className={`meter-label meter-label-${cls}`}>{formatPct(pace)}</span>}
    </div>
  );
}

// A larger budget meter with a marker at the "100% of budget" line, used on
// the campaign detail page's sidebar. The track's domain extends past 100%
// so an over-pace fill visibly crosses the marker instead of just clipping
// at the container edge.
export function BudgetMeter({ spend, budget }: { spend: number; budget: number }) {
  const pace = budget === 0 ? 0 : Math.round((spend / budget) * 100);
  const cls = paceClassOf(pace);
  const domainMax = Math.max(120, pace + 10);
  const fillWidth = Math.min(100, (pace / domainMax) * 100);
  const markerPos = (100 / domainMax) * 100;
  return (
    <div className="budget-meter">
      <div className={`meter meter-lg meter-${cls}`} role="img" aria-label={`${formatPct(pace)} of budget spent`}>
        <div className="meter-fill" style={{ width: `${fillWidth}%` }} />
        <div className="meter-marker" style={{ left: `${markerPos}%` }} />
      </div>
    </div>
  );
}
