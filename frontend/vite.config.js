import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: { port: 5173 }, // matches the Go backend's default ALLOWED_ORIGIN
  test: {
    environment: "jsdom",
    // jsdom's localStorage throws/no-ops on the default "about:blank"
    // document (no proper origin) — a real URL is what makes it behave
    // like a browser's localStorage, which session.js depends on.
    environmentOptions: { jsdom: { url: "http://localhost" } },
    // @testing-library/react's automatic per-test cleanup (unmounting
    // whatever the previous test rendered) hooks into a global `afterEach`
    // — needed so component tests don't accumulate DOM across tests in the
    // same file.
    globals: true,
    setupFiles: "./src/setupTests.js",
  },
});
