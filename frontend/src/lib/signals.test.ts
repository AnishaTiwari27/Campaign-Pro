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
    index: 1, categoryIndex: 1, flightDays: 20, expectedPace: 50, paceVsPlan: 1,
    ...over,
  } as Campaign;
}

describe("quadrantOf", () => {
  // The x-axis is spending against plan: 1.0 is exactly on plan, so 0.7 is
  // behind the flight and 1.3 is burning ahead of it.
  it("places the four clear cases", () => {
    expect(quadrantOf(0.7, 1.6)).toBe("scale");
    expect(quadrantOf(1.3, 1.6)).toBe("protect");
    expect(quadrantOf(0.7, 0.4)).toBe("fix");
    expect(quadrantOf(1.3, 0.4)).toBe("cut");
  });

  // The whole reason the axis changed: half the budget gone at the halfway
  // point is ON plan, and used to read as "underspending" against a raw
  // 100%-of-budget line.
  it("reads a mid-flight campaign as on plan, not as underspending", () => {
    expect(quadrantOf(1, 1.6)).toBe("protect");
    expect(quadrantOf(1, 0.4)).toBe("cut");
  });

  // The boundaries are the whole definition, so they are pinned rather
  // than left to whichever comparison happened to be written.
  it("treats exactly-median delivery as meeting the median", () => {
    expect(quadrantOf(0.7, INDEX_MIDLINE)).toBe("scale");
    expect(quadrantOf(1.3, INDEX_MIDLINE)).toBe("protect");
  });

  it("treats exactly-on-plan as spending to plan", () => {
    expect(quadrantOf(PACE_MIDLINE, 1.6)).toBe("protect");
    expect(quadrantOf(PACE_MIDLINE, 0.4)).toBe("cut");
  });

  it("never lands an on-median campaign in Cut", () => {
    expect(quadrantOf(3, INDEX_MIDLINE)).toBe("protect");
  });
});

describe("signalPoints", () => {
  it("drops scheduled campaigns, which have delivered nothing yet", () => {
    const points = signalPoints([
      campaign({ id: "live", status: "live" }),
      campaign({ id: "sched", status: "scheduled", pace: 0, index: 0, paceVsPlan: 0 }),
      campaign({ id: "ended", status: "ended" }),
      campaign({ id: "paused", status: "paused" }),
    ]);
    expect(points.map((p) => p.campaign.id)).toEqual(["live", "ended", "paused"]);
  });

  // Placing one would mean inventing the plan it is measured against.
  it("drops campaigns with no recorded flight length", () => {
    const points = signalPoints([
      campaign({ id: "planned", paceVsPlan: 1.2 }),
      campaign({ id: "no-plan", flightDays: 0, expectedPace: 0, paceVsPlan: 0 }),
    ]);
    expect(points.map((p) => p.campaign.id)).toEqual(["planned"]);
  });
});

describe("summarise", () => {
  it("totals budget and spend per quadrant, in a fixed order", () => {
    const rows = summarise(signalPoints([
      campaign({ id: "a", paceVsPlan: 1.4, index: 0.5, budget: 200_000, spend: 240_000 }),
      campaign({ id: "b", paceVsPlan: 1.6, index: 0.6, budget: 100_000, spend: 130_000 }),
      campaign({ id: "c", paceVsPlan: 0.4, index: 2, budget: 50_000, spend: 15_000 }),
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
