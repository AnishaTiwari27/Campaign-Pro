import type { Signals, Tier } from "../../api/types";
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

/** Whether anyone has paid this creator yet, as far as we can see.
 *
 *  Three states, and the third is the one that matters: a creator whose
 *  posts have never been read is "not yet checked", never "no brand
 *  work". Collapsing those would put our own backlog on screen as a
 *  finding about a named person. For the same reason the wording is
 *  about what we saw — "None in recent posts" — rather than about who
 *  they are: the scan reads recent media, never a full history.
 */
export function StageBadge({ signals }: { signals: Signals }) {
  const title =
    signals.stage === "proven"
      ? `${signals.brandPosts} disclosed brand ${signals.brandPosts === 1 ? "post" : "posts"} found in recent media`
      : signals.stage === "none"
        ? `Scanned${signals.scannedAt ? ` on ${signals.scannedAt}` : ""}; no disclosure found in recent posts`
        : "This creator's posts have not been read yet";
  return (
    <span className={`stage-badge stage-badge-${signals.stage}`} title={title}>
      {signals.stageLabel}
    </span>
  );
}

/** Engagement against the median for the creator's tier.
 *
 *  Shown as the measurement, never as a verdict about the person. A
 *  rate below the tier median is the honest proxy for a bought
 *  audience — "0.4% against 1.8%" is something we can stand behind,
 *  where "42% fake followers" is a claim about a named individual that
 *  no free data supports.
 */
export function EngagementRate({ signals, tierMedianRate }: { signals: Signals; tierMedianRate?: number }) {
  if (!signals.hasRate) {
    return (
      <span className="consistency-unknown" title="Needs at least three organic posts to measure">
        &ndash;
      </span>
    );
  }
  const ratio = tierMedianRate && tierMedianRate > 0 ? signals.engagementRate / tierMedianRate : 0;
  const tone = !ratio ? "" : ratio < 0.4 ? " engagement-crit" : ratio < 0.8 ? " engagement-warn" : "";
  return (
    <span
      className={`engagement-rate${tone}`}
      title={
        ratio
          ? `${signals.engagementRate.toFixed(2)}% against a tier median of ${tierMedianRate!.toFixed(2)}%`
          : `${signals.engagementRate.toFixed(2)}% across ${signals.postsScanned} recent posts`
      }
    >
      {signals.engagementRate.toFixed(2)}%
    </span>
  );
}
