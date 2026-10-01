import { describe, expect, it } from "vitest";
import { cpmOf, indexOf, median, paceClassOf, paceOf, reachCurve, valueAtAge } from "./metrics";

describe("reachCurve", () => {
  it("matches the fast preset", () => {
    const curve = reachCurve(100, "fast");
    const want = [10, 34, 55, 70, 81, 89, 95, 100];
    curve.forEach((v, i) => expect(v).toBeCloseTo(want[i]));
  });
  it("ends at the full reach", () => {
    expect(reachCurve(50, "steady")[7]).toBe(50);
  });
  it("falls back to steady for an unknown shape", () => {
    expect(reachCurve(100, "bogus" as never)).toEqual(reachCurve(100, "steady"));
  });
});

describe("valueAtAge", () => {
  it("is 0 at age 0", () => {
    expect(valueAtAge(100, "steady", 14, 0)).toBe(0);
  });
  it("is the full total at age == daysRunning", () => {
    expect(valueAtAge(100, "steady", 14, 14)).toBe(100);
  });
  it("clamps beyond daysRunning", () => {
    expect(valueAtAge(100, "steady", 14, 20)).toBe(100);
  });
  it("interpolates at the midpoint", () => {
    // age=7 of 14 -> fraction 0.5 -> curve position 3.5 -> halfway between
    // steady[3]=46 and steady[4]=59 -> 52.5.
    expect(valueAtAge(100, "steady", 14, 7)).toBeCloseTo(52.5);
  });
});

describe("paceOf / paceClassOf", () => {
  it("computes pace percentage", () => {
    expect(paceOf(50, 100)).toBe(50);
    expect(paceOf(0, 0)).toBe(0);
  });
  it("classes pace correctly", () => {
    expect(paceClassOf(100)).toBe("over");
    expect(paceClassOf(90)).toBe("warn");
    expect(paceClassOf(50)).toBe("good");
  });
});

describe("cpmOf", () => {
  it("computes cost per 1000 impressions", () => {
    // reach=10L, frequency=2 -> impressions=2,000,000; spend=200000 -> cpm=100.
    expect(cpmOf(200000, 10, 2)).toBeCloseTo(100);
  });
  it("returns 0 with no impressions", () => {
    expect(cpmOf(1000, 0, 0)).toBe(0);
  });
});

describe("median / indexOf", () => {
  it("computes median for odd and even lengths", () => {
    expect(median([3, 1, 2])).toBe(2);
    expect(median([1, 2, 3, 4])).toBe(2.5);
    expect(median([])).toBe(0);
  });
  it("computes index against a baseline", () => {
    expect(indexOf(140, 100)).toBeCloseTo(1.4);
    expect(indexOf(50, 0)).toBe(0);
  });
});
