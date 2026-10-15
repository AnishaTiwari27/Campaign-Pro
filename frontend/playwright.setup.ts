import { execFileSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

/**
 * Seed the database before the suite runs.
 *
 * Several specs approve campaigns, which drains the pending queue they
 * assert on, so a second run against the same data fails on counts that
 * were correct the first time. That has been mistaken for a regression
 * more than once: the suite reports a failure, the failure is not real,
 * and the next genuine one is easy to wave away as "just the seed".
 *
 * The Makefile already seeded first and the specs documented the
 * assumption in a comment, which worked right up until anyone ran
 * `npx playwright test` directly. Doing it here makes the suite
 * self-sufficient however it is started.
 *
 * Skipped when E2E_NO_SEED is set, for running against a deployment
 * where reseeding would be destructive.
 *
 * Lives beside playwright.config.ts rather than in tests/ because it is
 * configuration, not a spec: it uses node builtins, and tests/ is
 * typechecked by the app's tsconfig, which has no node types.
 */
export default function globalSetup() {
  if (process.env.E2E_NO_SEED) {
    console.log("[e2e] E2E_NO_SEED set — using the database as it stands");
    return;
  }
  // import.meta.url rather than __dirname: this config is ESM.
  const here = path.dirname(fileURLToPath(import.meta.url));
  const backend = path.resolve(here, "..", "backend");
  const databaseURL =
    process.env.DATABASE_URL ??
    "postgres://campaign_tracker_pro:campaign_tracker_pro_dev_pw@localhost:5432/campaign_tracker_pro?sslmode=disable";

  try {
    execFileSync("go", ["run", "./cmd/seed"], {
      cwd: backend,
      stdio: ["ignore", "pipe", "pipe"],
      env: {
        ...process.env,
        DATABASE_URL: databaseURL,
        // The specs sign in as fixture users, so they need the password
        // the seeder would otherwise generate.
        SEED_PASSWORD: "demo-password-change-me",
      },
    });
    console.log("[e2e] database seeded");
  } catch (err) {
    const e = err as { stderr?: Buffer; stdout?: Buffer };
    // Fail loudly rather than letting the suite run on whatever state
    // happens to be there and report confusing count mismatches.
    throw new Error(
      "[e2e] could not seed the database. Start Postgres, or set E2E_NO_SEED=1 " +
        "to run against the current data.\n" +
        (e.stderr?.toString() ?? e.stdout?.toString() ?? String(err)),
    );
  }
}
