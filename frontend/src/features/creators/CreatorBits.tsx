import type { Tier } from "../../api/types";
import "./CreatorBits.css";

export function formatFollowers(n: number): string {
  if (n >= 10_000_000) return `${(n / 10_000_000).toFixed(1)}Cr`;
  if (n >= 100_000) return `${(n / 100_000).toFixed(1)}L`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(0)}K`;
  return String(n);
}

export function TierBadge({ tier, label }: { tier: Tier; label: string }) {
  return <span className={`tier-badge tier-badge-${tier}`}>{label}</span>;
}

// A bar centred on 1.0x: left of centre is under-delivering for the tier,
// right of centre is over. Reading a bare number against a baseline is
// harder than seeing which side of the line it falls on.
export function IndexBar({ index }: { index: number }) {
  const clamped = Math.max(0, Math.min(2, index));
  const pct = (clamped / 2) * 100;
  const over = index >= 1;
  return (
    <span className="index-bar-wrap">
      <span className={`index-bar-value mono ${over ? "index-over" : "index-under"}`}>{index.toFixed(2)}x</span>
      <span className="index-bar" role="img" aria-label={`${index.toFixed(2)} times the tier median`}>
        <span className="index-bar-baseline" />
        <span className={`index-bar-fill ${over ? "index-bar-over" : "index-bar-under"}`} style={{ width: `${pct}%` }} />
      </span>
    </span>
  );
}

// A single campaign can't demonstrate repeatability, so below two we show
// nothing rather than a misleading full score.
export function ConsistencyDots({ score, n }: { score: number; n?: number }) {
  if (n !== undefined && n < 2) {
    return (
      <span className="consistency-unknown" title="Needs at least two campaigns to measure">
        &ndash;
      </span>
    );
  }
  const filled = Math.round(score * 5);
  const tone = score >= 0.8 ? "good" : score >= 0.55 ? "warn" : "crit";
  return (
    <span className="consistency" role="img" aria-label={`Consistency ${Math.round(score * 100)} percent`}>
      {[0, 1, 2, 3, 4].map((i) => (
        <span key={i} className={`consistency-dot${i < filled ? ` consistency-dot-${tone}` : ""}`} />
      ))}
    </span>
  );
}
