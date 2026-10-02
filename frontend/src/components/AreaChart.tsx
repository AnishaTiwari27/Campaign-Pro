import { useId, useMemo, useState } from "react";
import "./AreaChart.css";

export interface AreaChartPoint {
  label: string;
  value: number;
}

interface AreaChartProps {
  points: AreaChartPoint[];
  medianValue?: number;
  medianLabel?: string;
  formatValue: (v: number) => string;
  height?: number;
  interactive?: boolean;
  color?: string;
}

// A hand-written SVG cumulative area chart: gradient fill, y ticks, an
// optional dashed baseline (the category median), and — when interactive —
// a hover crosshair with a tooltip showing the point plus its delta from
// the previous one.
export function AreaChart({
  points,
  medianValue,
  medianLabel = "Category median",
  formatValue,
  height = 220,
  interactive = true,
  color = "var(--accent)",
}: AreaChartProps) {
  const gradientId = useId();
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);

  const width = 640;
  const padY = 16;
  const padX = 4;

  const values = points.map((p) => p.value);
  const maxVal = Math.max(...values, medianValue ?? 0, 1);
  const minVal = 0;
  const range = maxVal - minVal || 1;

  const stepX = (width - padX * 2) / Math.max(1, points.length - 1);
  const coords = useMemo(
    () =>
      points.map((p, i) => ({
        x: padX + i * stepX,
        y: padY + (height - padY * 2) * (1 - (p.value - minVal) / range),
      })),
    [points, stepX, height, range],
  );

  const linePath = coords.length ? `M${coords.map((c) => `${c.x},${c.y}`).join(" L")}` : "";
  const areaPath = coords.length
    ? `${linePath} L${coords[coords.length - 1].x},${height - padY} L${coords[0].x},${height - padY} Z`
    : "";

  const medianY =
    medianValue !== undefined ? padY + (height - padY * 2) * (1 - (medianValue - minVal) / range) : null;

  const ticks = 4;
  const tickValues = Array.from({ length: ticks + 1 }, (_, i) => (maxVal / ticks) * i);

  function handleMove(e: React.MouseEvent<SVGRectElement>) {
    if (!interactive || coords.length === 0) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const relX = ((e.clientX - rect.left) / rect.width) * width;
    let nearest = 0;
    let nearestDist = Infinity;
    coords.forEach((c, i) => {
      const dist = Math.abs(c.x - relX);
      if (dist < nearestDist) {
        nearestDist = dist;
        nearest = i;
      }
    });
    setHoverIndex(nearest);
  }

  const hovered = hoverIndex !== null ? points[hoverIndex] : null;
  const hoveredCoord = hoverIndex !== null ? coords[hoverIndex] : null;
  const delta =
    hoverIndex !== null && hoverIndex > 0 ? points[hoverIndex].value - points[hoverIndex - 1].value : null;

  return (
    <div className="area-chart" style={{ height }}>
      <svg
        viewBox={`0 0 ${width} ${height}`}
        width="100%"
        height={height}
        role="img"
        aria-label="Cumulative reach over time"
        preserveAspectRatio="none"
      >
        <defs>
          <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity={0.32} />
            <stop offset="100%" stopColor={color} stopOpacity={0} />
          </linearGradient>
        </defs>

        {tickValues.map((tv, i) => {
          const y = padY + (height - padY * 2) * (1 - tv / (maxVal || 1));
          return <line key={i} x1={0} x2={width} y1={y} y2={y} className="area-chart-gridline" />;
        })}

        {medianY !== null && (
          <line x1={0} x2={width} y1={medianY} y2={medianY} className="area-chart-median-line" />
        )}

        {areaPath && <path d={areaPath} fill={`url(#${gradientId})`} />}
        {linePath && <path d={linePath} fill="none" stroke={color} strokeWidth={2} />}

        {hoveredCoord && (
          <line
            x1={hoveredCoord.x}
            x2={hoveredCoord.x}
            y1={padY}
            y2={height - padY}
            className="area-chart-crosshair"
          />
        )}
        {hoveredCoord && <circle cx={hoveredCoord.x} cy={hoveredCoord.y} r={4} fill={color} />}

        {interactive && (
          <rect
            x={0}
            y={0}
            width={width}
            height={height}
            fill="transparent"
            onMouseMove={handleMove}
            onMouseLeave={() => setHoverIndex(null)}
          />
        )}
      </svg>

      <div className="area-chart-y-labels" aria-hidden="true">
        {[...tickValues].reverse().map((tv, i) => (
          <span key={i}>{formatValue(tv)}</span>
        ))}
      </div>

      {medianValue !== undefined && (
        <div className="area-chart-legend">
          <span className="area-chart-legend-swatch area-chart-legend-line" />
          <span>Reach</span>
          <span className="area-chart-legend-swatch area-chart-legend-dashed" />
          <span>{medianLabel}</span>
        </div>
      )}

      {hovered && hoveredCoord && (
        <div
          className="area-chart-tooltip"
          style={{ left: `${(hoveredCoord.x / width) * 100}%`, top: `${(hoveredCoord.y / height) * 100}%` }}
        >
          <div className="area-chart-tooltip-value">{formatValue(hovered.value)} cumulative</div>
          <div className="area-chart-tooltip-meta">
            {hovered.label}
            {delta !== null && <> · {delta >= 0 ? "+" : ""}{formatValue(delta)} that day</>}
          </div>
        </div>
      )}
    </div>
  );
}
