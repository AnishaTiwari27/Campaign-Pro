import { describe, expect, it } from "vitest";
import { ssoFailureMessage } from "./SsoButton";

describe("ssoFailureMessage", () => {
  it("explains each way the provider round trip can end badly", () => {
    expect(ssoFailureMessage("?sso=denied")).toMatch(/cancelled/i);
    expect(ssoFailureMessage("?sso=expired")).toMatch(/too long/i);
    expect(ssoFailureMessage("?sso=unverified")).toMatch(/verified/i);
    expect(ssoFailureMessage("?sso=failed")).toMatch(/try again/i);
  });

  it("says nothing on an ordinary visit", () => {
    expect(ssoFailureMessage("")).toBeNull();
    expect(ssoFailureMessage("?from=email")).toBeNull();
  });

  // The value lands in an alert the person reads. Anything the server
  // did not send must not become a message, or the query string becomes
  // a way to put arbitrary text on the sign-in screen.
  it("ignores a value it does not recognise", () => {
    expect(ssoFailureMessage("?sso=your+account+is+suspended")).toBeNull();
    expect(ssoFailureMessage("?sso=")).toBeNull();
  });
});
