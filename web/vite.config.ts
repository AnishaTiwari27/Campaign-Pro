/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// In compose the API is another service; natively it's on localhost.
const apiTarget = process.env.VITE_API_PROXY_TARGET ?? "http://localhost:8090";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": apiTarget,
      "/health": apiTarget,
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./tests/setup.ts"],
    // tests/ holds the Playwright suite, which has its own runner.
    include: ["src/**/*.test.{ts,tsx}"],
  },
});
