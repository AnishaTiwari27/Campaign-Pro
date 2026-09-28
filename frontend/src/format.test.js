import { describe, expect, it } from "vitest";
import { fmtDate, fmtMoney, fmtReach } from "./format";

describe("fmtDate", () => {
  it("formats a YYYY-MM-DD date string", () => {
    expect(fmtDate("2026-08-20")).toBe("20 Aug 26");
  });

  it("tolerates a full ISO timestamp, not just a bare date", () => {
    expect(fmtDate("2026-08-20T14:30:00Z")).toBe("20 Aug 26");
  });
});

describe("fmtReach", () => {
  it("formats large numbers in lakhs", () => {
    expect(fmtReach(2140000)).toBe("21.4L");
  });

  it("formats numbers under a lakh in thousands", () => {
    expect(fmtReach(42000)).toBe("42K");
  });

  it("holds at the lakh boundary", () => {
    expect(fmtReach(100000)).toBe("1.0L");
    expect(fmtReach(99999)).toBe("100K"); // just under the boundary — still thousands-formatted, rounds up
  });
});

describe("fmtMoney", () => {
  it("prefixes the rupee sign and formats in lakhs", () => {
    expect(fmtMoney(381920)).toBe("₹3.8L");
  });

  it("formats small amounts in thousands", () => {
    expect(fmtMoney(5000)).toBe("₹5K");
  });
});
