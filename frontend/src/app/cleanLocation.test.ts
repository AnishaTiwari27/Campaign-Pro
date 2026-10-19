import { describe, expect, it } from "vitest";
import { cleanPath } from "./cleanLocation";

describe("cleanPath", () => {
  // The one that actually happened: a URL pasted with its line break
  // still attached. React Router's catch-all compiles to `(.*)`, and
  // `.` does not match a newline, so this matched no route at all —
  // not the page, not the 404 — and the app rendered a blank screen.
  it("strips a trailing newline", () => {
    expect(cleanPath("/overview\n")).toBe("/overview");
    expect(cleanPath("/overview\r\n")).toBe("/overview");
  });

  // What the browser actually hands over. location.pathname is the
  // encoded form, so the newline arrives as the three printable
  // characters "%0A" — which is how the first version of this missed
  // it entirely while passing its own tests.
  it("strips control characters that are still percent-encoded", () => {
    expect(cleanPath("/overview%0A")).toBe("/overview");
    expect(cleanPath("/overview%0D%0A")).toBe("/overview");
    expect(cleanPath("/camp%09aigns")).toBe("/campaigns");
    expect(cleanPath("/settings%7F")).toBe("/settings");
    expect(cleanPath("/overview%20")).toBe("/overview");
  });

  it("strips control characters anywhere in the path", () => {
    expect(cleanPath("/camp\taigns")).toBe("/campaigns");
    expect(cleanPath("/\u0000billing")).toBe("/billing");
    expect(cleanPath("/settings\u007F")).toBe("/settings");
  });

  it("trims trailing whitespace", () => {
    expect(cleanPath("/overview  ")).toBe("/overview");
  });

  // Anything routable is left exactly as it was: this runs on every
  // load, so a path it rewrites needlessly is a navigation nobody
  // asked for.
  it("leaves an ordinary path alone", () => {
    for (const path of [
      "/",
      "/overview",
      "/campaigns/ms-dhoni-cars24",
      "/soon/ai-credits",
      // A legitimate encoded space mid-path survives: only a trailing
      // one is the pasted-whitespace case.
      "/creators/bhuvan%20bam",
    ]) {
      expect(cleanPath(path)).toBe(path);
    }
  });

  it("never returns an empty path", () => {
    // "/" is somewhere; "" is not, and would leave the router with
    // nothing to match either.
    expect(cleanPath("\n")).toBe("/");
    expect(cleanPath("")).toBe("/");
  });
});
