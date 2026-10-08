import { describe, expect, it } from "vitest";
import { quadrantOf, signalPoints, summarise, PACE_MIDLINE, INDEX_MIDLINE } from "./signals";
import type { Campaign } from "../api/types";

function campaign(over: Partial<Campaign>): Campaign {
  return {
    id: "c1", name: "A campaign", subjectType: "brand", initials: "AC",
    category: "Retail", region: "Mumbai", adType: "Social", platform: "Meta",
    status: "live", daysRunning: 10, reach: 5, spend: 50_000, budget: 100_000,
    frequency: 2, approval: "approved", curveShape: "steady",
    createdAt: "", updatedAt: "", pace: 50, paceClass: "on", cpm: 100,
    index: 1, categoryIndex: 1,
    ...over,
  } as Campaign;
}

describe("quadrantOf", () => {
  it("places the four clear cases", () => {
    expect(quadrantOf(40, 1.6)).toBe("scale");
    expect(quadrantOf(120, 1.6)).toBe("protect");
    expect(quadrantOf(40, 0.4)).toBe("fix");
    expect(quadrantOf(120, 0.4)).toBe("cut");
  });

  // The boundaries are the whole definition, so they are pinned rather
  // than left to whichever comparison happened to be written.
  it("treats exactly-median delivery as meeting the median", () => {
    expect(quadrantOf(40, INDEX_MIDLINE)).toBe("scale");
    expect(quadrantOf(120, INDEX_MIDLINE)).toBe("protect");
  });

  it("treats exactly-full pace as spent", () => {
    expect(quadrantOf(PACE_MIDLINE, 1.6)).toBe("protect");
    expect(quadrantOf(PACE_MIDLINE, 0.4)).toBe("cut");
  });

  it("never lands an on-median campaign in Cut", () => {
    expect(quadrantOf(300, INDEX_MIDLINE)).toBe("protect");
  });
});

describe("signalPoints", () => {
  it("drops scheduled campaigns, which have delivered nothing yet", () => {
    const points = signalPoints([
      campaign({ id: "live", status: "live" }),
      campaign({ id: "sched", status: "scheduled", pace: 0, index: 0 }),
      campaign({ id: "ended", status: "ended" }),
      campaign({ id: "paused", status: "paused" }),
    ]);
    expect(points.map((p) => p.campaign.id)).toEqual(["live", "ended", "paused"]);
  });
});

describe("summarise", () => {
  it("totals budget and spend per quadrant, in a fixed order", () => {
    const rows = summarise(signalPoints([
      campaign({ id: "a", pace: 120, index: 0.5, budget: 200_000, spend: 240_000 }),
      campaign({ id: "b", pace: 130, index: 0.6, budget: 100_000, spend: 130_000 }),
      campaign({ id: "c", pace: 30, index: 2, budget: 50_000, spend: 15_000 }),
    ]));
    expect(rows.map((r) => r.key)).toEqual(["scale", "protect", "fix", "cut"]);
    const cut = rows.find((r) => r.key === "cut")!;
    expect(cut.points).toHaveLength(2);
    expect(cut.budget).toBe(300_000);
    expect(cut.spend).toBe(370_000);
    expect(rows.find((r) => r.key === "scale")!.budget).toBe(50_000);
    expect(rows.find((r) => r.key === "fix")!.points).toHaveLength(0);
  });
});
