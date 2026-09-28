import { beforeEach, describe, expect, it, vi } from "vitest";
import { clearSession, getAccessToken, getRefreshToken, getRole, getUserId, hasAnyRole, isAuthenticated, setSession, subscribe } from "./session";

beforeEach(() => {
  clearSession();
  localStorage.clear();
});

function base64url(str) {
  return btoa(str).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

// Builds a real-shaped (unsigned) JWT — getRole() only ever reads the
// payload segment, never verifies the signature (that's the server's job,
// on every write — see session.js's getRole comment), so a fake signature
// is enough to exercise the decode path.
function fakeJWT(claims) {
  return `${base64url(JSON.stringify({ alg: "HS256", typ: "JWT" }))}.${base64url(JSON.stringify(claims))}.fake-signature`;
}

describe("session", () => {
  it("starts unauthenticated with no tokens", () => {
    expect(isAuthenticated()).toBe(false);
    expect(getAccessToken()).toBeNull();
  });

  it("setSession stores the access token in memory and the refresh token in localStorage", () => {
    setSession({ accessToken: "access-1", refreshToken: "refresh-1" });
    expect(isAuthenticated()).toBe(true);
    expect(getAccessToken()).toBe("access-1");
    expect(getRefreshToken()).toBe("refresh-1");
  });

  it("clearSession removes both", () => {
    setSession({ accessToken: "access-1", refreshToken: "refresh-1" });
    clearSession();
    expect(isAuthenticated()).toBe(false);
    expect(getAccessToken()).toBeNull();
    expect(getRefreshToken()).toBeNull();
  });

  it("a later setSession call replaces the refresh token (rotation)", () => {
    setSession({ accessToken: "a1", refreshToken: "r1" });
    setSession({ accessToken: "a2", refreshToken: "r2" });
    expect(getAccessToken()).toBe("a2");
    expect(getRefreshToken()).toBe("r2");
  });

  it("subscribe is notified with the new isAuthenticated() value on every change", () => {
    const fn = vi.fn();
    const unsubscribe = subscribe(fn);

    setSession({ accessToken: "a1", refreshToken: "r1" });
    expect(fn).toHaveBeenLastCalledWith(true);

    clearSession();
    expect(fn).toHaveBeenLastCalledWith(false);

    unsubscribe();
    setSession({ accessToken: "a2", refreshToken: "r2" });
    expect(fn).toHaveBeenCalledTimes(2); // not called again after unsubscribing
  });

  it("getRole is null when unauthenticated", () => {
    expect(getRole()).toBeNull();
  });

  it("getRole decodes the access token's own role claim", () => {
    setSession({ accessToken: fakeJWT({ sub: "u1", role: "admin" }), refreshToken: "r1" });
    expect(getRole()).toBe("admin");
  });

  it("getRole reflects whatever the current token actually claims", () => {
    setSession({ accessToken: fakeJWT({ sub: "u1", role: "viewer" }), refreshToken: "r1" });
    expect(getRole()).toBe("viewer");
  });

  it("getRole is null for a malformed access token, not a thrown error", () => {
    setSession({ accessToken: "not-a-real-jwt", refreshToken: "r1" });
    expect(getRole()).toBeNull();
  });

  it("hasAnyRole matches when the current role is in the allowed set", () => {
    setSession({ accessToken: fakeJWT({ sub: "u1", role: "editor" }), refreshToken: "r1" });
    expect(hasAnyRole("admin", "editor")).toBe(true);
  });

  it("hasAnyRole is false when the current role isn't in the allowed set", () => {
    setSession({ accessToken: fakeJWT({ sub: "u1", role: "viewer" }), refreshToken: "r1" });
    expect(hasAnyRole("admin", "editor")).toBe(false);
  });

  it("hasAnyRole is false when unauthenticated", () => {
    expect(hasAnyRole("admin", "editor")).toBe(false);
  });

  it("getUserId decodes the access token's subject claim", () => {
    setSession({ accessToken: fakeJWT({ sub: "u1", role: "admin" }), refreshToken: "r1" });
    expect(getUserId()).toBe("u1");
  });

  it("getUserId is null when unauthenticated", () => {
    expect(getUserId()).toBeNull();
  });
});
