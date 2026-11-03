import { describe, expect, it } from "vitest";
import { engagementMediansByTier } from "./engagement";
import type { CreatorPerformance, Signals, Tier } from "../../api/types";

function creator(tier: Tier, rate: number | null): CreatorPerformance {
  const signals: Signals = {
    scanned: rate !== null,
    stage: rate === null ? "unknown" : "none",
    stageLabel: "",
    brandPosts: 0,
    engagementRate: rate ?? 0,
    hasRate: rate !== null,
    postsScanned: rate === null ? 0 : 25,
  };
  return { tier, signals } as unknown as CreatorPerformance;
}

describe("engagementMediansByTier", () => {
  it("takes the median within each tier, not across the whole roster", () => {
    const out = engagementMediansByTier([
      creator("nano", 6), creator("nano", 8), creator("nano", 10),
      creator("mega", 0.4), creator("mega", 0.6), creator("mega", 0.8),
    ]);
    // Small audiences engage harder. Pooling the tiers would judge
    // every mega account as failing and every nano one as excellent,
    // which tells a buyer nothing.
    expect(out.nano).toBe(8);
    expect(out.mega).toBeCloseTo(0.6);
  });

  it("withholds a median from a tier with too few scanned peers", () => {
    const out = engagementMediansByTier([creator("micro", 2), creator("micro", 4)]);
    // Two points do not make a benchmark, and a creator compared
    // against one other creator is shown a verdict built from a
    // sample of one.
    expect(out.micro).toBeUndefined();
  });

  it("ignores creators with no measurable rate", () => {
    const out = engagementMediansByTier([
      creator("mid", 1), creator("mid", 3), creator("mid", 5),
      creator("mid", null), creator("mid", null),
    ]);
    // The unscanned must not drag the median towards zero — they are
    // absent data, not zero engagement.
    expect(out.mid).toBe(3);
  });

  it("gives no median for a tier where nobody has been scanned", () => {
    expect(engagementMediansByTier([creator("macro", null), creator("macro", null)]).macro)
      .toBeUndefined();
  });

  it("averages the middle pair on an even count", () => {
    const out = engagementMediansByTier([
      creator("nano", 1), creator("nano", 2), creator("nano", 4), creator("nano", 5),
    ]);
    expect(out.nano).toBe(3);
  });

  it("handles an empty roster", () => {
    expect(engagementMediansByTier([])).toEqual({});
  });
});
