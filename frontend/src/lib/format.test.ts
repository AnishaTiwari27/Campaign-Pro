import { describe, expect, it } from "vitest";
import { formatCPM, formatIndex, formatMoney, formatReach } from "./format";

describe("formatReach", () => {
  it("stays in L under 100", () => {
    expect(formatReach(34.2)).toBe("34.2L");
  });
  it("rolls up to Cr at 100+", () => {
    expect(formatReach(123)).toBe("1.23Cr");
  });
});

describe("formatMoney", () => {
  it("formats plain rupees", () => {
    expect(formatMoney(900)).toBe("₹900");
  });
  it("formats thousands", () => {
    expect(formatMoney(44000)).toBe("₹44K");
  });
  it("formats lakh", () => {
    expect(formatMoney(440000)).toBe("₹4.4L");
  });
  it("formats crore", () => {
    expect(formatMoney(12300000)).toBe("₹1.23Cr");
  });
});

describe("formatIndex", () => {
  it("renders a multiple", () => {
    expect(formatIndex(1.4)).toBe("1.4x");
  });
});

describe("formatCPM", () => {
  it("renders an em dash for zero", () => {
    expect(formatCPM(0)).toBe("—");
  });
  it("rounds to whole rupees", () => {
    expect(formatCPM(123.6)).toBe("₹124");
  });
});
