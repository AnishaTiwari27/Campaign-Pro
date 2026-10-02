import { adTypeColorVar } from "./AdTypeTag";
import type { AdTypeCount } from "../api/types";
import "./StackedBar.css";

// A single-row stacked bar showing a region's ad-type split, with a small
// swatch legend beneath it.
export function StackedBar({ split }: { split: AdTypeCount[] }) {
  const total = split.reduce((sum, s) => sum + s.count, 0) || 1;
  return (
    <div className="stacked-bar-wrap">
      <div className="stacked-bar" role="img" aria-label="Ad type split">
        {split.map((s) => (
          <span
            key={s.adType}
            className="stacked-bar-segment"
            style={{ width: `${(s.count / total) * 100}%`, background: adTypeColorVar(s.adType) }}
            title={`${s.adType}: ${s.count}`}
          />
        ))}
      </div>
      <div className="stacked-bar-legend">
        {split.map((s) => (
          <span key={s.adType} className="stacked-bar-legend-item">
            <span className="stacked-bar-legend-swatch" style={{ background: adTypeColorVar(s.adType) }} />
            {s.adType} · {s.count}
          </span>
        ))}
      </div>
    </div>
  );
}
