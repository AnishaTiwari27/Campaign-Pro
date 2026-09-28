import "@testing-library/jest-dom/vitest";

// jsdom's own localStorage isn't reliably available in every Node/jsdom
// version combination (a real gap hit while wiring up this test suite —
// it came back `undefined` here despite a proper environmentOptions.jsdom.url).
// auth/session.js only needs the plain Storage interface
// (getItem/setItem/removeItem/clear), so a small in-memory polyfill
// standing in for it is simpler and more portable than chasing jsdom
// internals — same approach many projects use for exactly this reason.
if (typeof globalThis.localStorage === "undefined" || typeof globalThis.localStorage.clear !== "function") {
  const store = new Map();
  const memoryLocalStorage = {
    getItem: (key) => (store.has(key) ? store.get(key) : null),
    setItem: (key, value) => store.set(key, String(value)),
    removeItem: (key) => store.delete(key),
    clear: () => store.clear(),
  };
  Object.defineProperty(globalThis, "localStorage", { value: memoryLocalStorage, configurable: true });
  if (typeof window !== "undefined") {
    Object.defineProperty(window, "localStorage", { value: memoryLocalStorage, configurable: true });
  }
}
