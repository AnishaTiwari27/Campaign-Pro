import { defineConfig } from "@playwright/test";
import os from "node:os";

// This machine has no Docker and no `npx playwright install` browser set, so
// the suite points at the Chrome for Testing build already in the Playwright
// cache. Override with PLAYWRIGHT_CHROMIUM_PATH if yours lives elsewhere.
const executablePath =
  process.env.PLAYWRIGHT_CHROMIUM_PATH ??
  `${os.homedir()}/Library/Caches/ms-playwright/chromium-1234/chrome-mac-arm64/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing`;

const baseURL = process.env.E2E_BASE_URL ?? "http://localhost:5173";

export default defineConfig({
  testDir: "./tests",
  // These tests share one database and several of them approve campaigns,
  // draining the same pending queue. Running them in parallel makes the
  // counts race, so they run serially against a freshly seeded state
  // (`make test-e2e` seeds first).
  fullyParallel: false,
  workers: 1,
  testMatch: "**/*.spec.ts",
  timeout: 30_000,
  reporter: [["list"]],
  use: {
    baseURL,
    launchOptions: { executablePath },
    viewport: { width: 1440, height: 900 },
  },
});
