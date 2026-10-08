import { useMemo } from "react";
import { PACE_MIDLINE, INDEX_MIDLINE, QUADRANTS, type SignalPoint } from "../lib/signals";
import { formatIndex, formatMoney } from "../lib/format";
import "./QuadrantChart.css";

const W = 720;
const H = 440;
const PAD = { top: 26, right: 22, bottom: 46, left: 56 };
const PLOT_W = W - PAD.left - PAD.right;
const PLOT_H = H - PAD.top - PAD.bottom;

// Radius carries budget. Area, not radius, is proportional to it — a
// radius-linear scale makes a twice-as-big budget look four times the dot.
const R_MIN = 4;
const R_MAX = 13;

function radiusOf(budget: number, maxBudget: number): number {
  if (maxBudget <= 0) return R_MIN;
  return R_MIN + (R_MAX - R_MIN) * Math.sqrt(Math.max(0, budget) / maxBudget);
}

/**
 * Pace against reach index, with the two midlines dividing the fleet into
 * Scale / Protect / Fix / Cut. Both axes start at zero: this chart is read
 * by where a point sits relative to a line, so a truncated axis would move
 * campaigns between quadrants visually without moving them actually.
 *
 * The dots are one colour on purpose. Position already says which quadrant
 * a campaign is in — that is the entire premise of the form — so colouring
 * them four ways would encode the same fact twice, and the two statuses it
 * would need (warn and crit) separate by ΔE 12.7 in normal vision, under
 * the readable floor. The faint zone tints carry the region instead, where
 * they are large areas rather than adjacent 8px marks.
 */
export function QuadrantChart({
  points,
  onPointClick,
}: {
  points: SignalPoint[];
  onPointClick?: (campaignId: string) => void;
}) {
  const { xMax, yMax, maxBudget } = useMemo(() => {
    // Domains always contain the midline with room past it, so an empty or
    // one-sided fleet still renders four readable quadrants.
    const paces = points.map((p) => p.pace);
    const indices = points.map((p) => p.index);
    return {
      xMax: Math.max(PACE_MIDLINE * 1.4, ...paces.map((v) => v * 1.08)),
      yMax: Math.max(INDEX_MIDLINE * 2, ...indices.map((v) => v * 1.12)),
      maxBudget: Math.max(0, ...points.map((p) => p.campaign.budget)),
    };
  }, [points]);

  const x = (pace: number) => PAD.left + (Math.min(pace, xMax) / xMax) * PLOT_W;
  const y = (index: number) => PAD.top + PLOT_H - (Math.min(index, yMax) / yMax) * PLOT_H;

  const midX = x(PACE_MIDLINE);
  const midY = y(INDEX_MIDLINE);

  // Corner label per quadrant, placed in the corner furthest from the axes
  // crossing so it never sits under the densest cluster.
  const corners = [
    { q: QUADRANTS.scale, tx: PAD.left + 10, ty: PAD.top + 18, anchor: "start" as const },
    { q: QUADRANTS.protect, tx: W - PAD.right - 10, ty: PAD.top + 18, anchor: "end" as const },
    { q: QUADRANTS.fix, tx: PAD.left + 10, ty: PAD.top + PLOT_H - 10, anchor: "start" as const },
    { q: QUADRANTS.cut, tx: W - PAD.right - 10, ty: PAD.top + PLOT_H - 10, anchor: "end" as const },
  ];

  return (
    <svg className="quadrant-chart" viewBox={`0 0 ${W} ${H}`} role="img"
      aria-label={`${points.length} campaigns plotted by budget pace against reach index`}>
      {/* quadrant tints */}
      <rect className="quadrant-zone quadrant-zone-good" x={PAD.left} y={PAD.top}
        width={midX - PAD.left} height={midY - PAD.top} />
      <rect className="quadrant-zone quadrant-zone-accent" x={midX} y={PAD.top}
        width={PAD.left + PLOT_W - midX} height={midY - PAD.top} />
      <rect className="quadrant-zone quadrant-zone-warn" x={PAD.left} y={midY}
        width={midX - PAD.left} height={PAD.top + PLOT_H - midY} />
      <rect className="quadrant-zone quadrant-zone-crit" x={midX} y={midY}
        width={PAD.left + PLOT_W - midX} height={PAD.top + PLOT_H - midY} />

      {/* frame + midlines */}
      <rect className="quadrant-frame" x={PAD.left} y={PAD.top} width={PLOT_W} height={PLOT_H} />
      <line className="quadrant-midline" x1={midX} y1={PAD.top} x2={midX} y2={PAD.top + PLOT_H} />
      <line className="quadrant-midline" x1={PAD.left} y1={midY} x2={PAD.left + PLOT_W} y2={midY} />

      {corners.map((c) => (
        <text key={c.q.key} className="quadrant-corner"
          x={c.tx} y={c.ty} textAnchor={c.anchor}>
          {c.q.label.toUpperCase()}
        </text>
      ))}

      {/* axes */}
      <text className="quadrant-axis-title" x={PAD.left + PLOT_W / 2} y={H - 8} textAnchor="middle">
        Budget pace →
      </text>
      <text className="quadrant-axis-title" x={14} y={PAD.top + PLOT_H / 2}
        textAnchor="middle" transform={`rotate(-90 14 ${PAD.top + PLOT_H / 2})`}>
        Reach index →
      </text>
      <text className="quadrant-tick" x={midX} y={PAD.top + PLOT_H + 17} textAnchor="middle">
        {PACE_MIDLINE}% spent
      </text>
      <text className="quadrant-tick" x={PAD.left - 8} y={midY + 4} textAnchor="end">
        {formatIndex(INDEX_MIDLINE)}
      </text>
      <text className="quadrant-tick" x={PAD.left - 8} y={PAD.top + 4} textAnchor="end">
        {formatIndex(yMax)}
      </text>
      <text className="quadrant-tick" x={PAD.left - 8} y={PAD.top + PLOT_H + 4} textAnchor="end">
        0
      </text>

      {points.map((p) => {
        const clickable = Boolean(onPointClick);
        return (
          <circle
            key={p.campaign.id}
            className={`quadrant-dot${clickable ? " quadrant-dot-clickable" : ""}`}
            cx={x(p.pace)}
            cy={y(p.index)}
            r={radiusOf(p.campaign.budget, maxBudget)}
            tabIndex={clickable ? 0 : undefined}
            role={clickable ? "button" : undefined}
            onClick={clickable ? () => onPointClick?.(p.campaign.id) : undefined}
            onKeyDown={
              clickable
                ? (e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      onPointClick?.(p.campaign.id);
                    }
                  }
                : undefined
            }
          >
            {/* Native tooltip: no portal, works on touch-and-hold, and is
                read out by assistive tech as the element's accessible name. */}
            <title>
              {`${p.campaign.name} — ${QUADRANTS[p.quadrant].label}\n`}
              {`Pace ${Math.round(p.pace)}% · Index ${formatIndex(p.index)} · Budget ${formatMoney(p.campaign.budget)}`}
            </title>
          </circle>
        );
      })}
    </svg>
  );
}
