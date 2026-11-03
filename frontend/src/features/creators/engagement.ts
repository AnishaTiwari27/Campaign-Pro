import type { CreatorPerformance, Tier } from "../../api/types";

/** The median engagement rate within each tier, computed from whoever
 *  has actually been scanned.
 *
 *  Computed here rather than fetched because it is a property of the
 *  set on screen: comparing a nano creator's rate against a mega
 *  account's says nothing except that small audiences engage harder,
 *  which is true of every creator and tells a buyer nothing.
 *
 *  A tier with fewer than three scanned creators gets no median. Two
 *  data points do not make a benchmark, and a creator compared against
 *  one other creator would be shown a verdict built from a sample of
 *  one.
 */
const MIN_PEERS = 3;

export function engagementMediansByTier(items: CreatorPerformance[]): Partial<Record<Tier, number>> {
  const byTier: Partial<Record<Tier, number[]>> = {};
  for (const c of items) {
    if (!c.signals.hasRate) continue;
    (byTier[c.tier] ??= []).push(c.signals.engagementRate);
  }
  const out: Partial<Record<Tier, number>> = {};
  for (const [tier, rates] of Object.entries(byTier)) {
    if (!rates || rates.length < MIN_PEERS) continue;
    const sorted = [...rates].sort((a, b) => a - b);
    const mid = Math.floor(sorted.length / 2);
    out[tier as Tier] =
      sorted.length % 2 === 1 ? sorted[mid] : (sorted[mid - 1] + sorted[mid]) / 2;
  }
  return out;
}
