import { expect, test } from "@playwright/test";

// Switching tabs used to re-run the session check on window focus, and
// while that was in flight the auth gate fell back to its loading state
// and unmounted the login form — losing anything typed. Reported by
// typing an email, switching tabs to copy a password, and coming back to
// an empty field.
test("switching away and back does not clear what you have typed", async ({ context, page }) => {
  await page.goto("/");
  await page.fill("input[type=email]", "anishatiwari695@gmail.com");

  const other = await context.newPage();
  await other.goto("about:blank");
  await other.bringToFront();
  await page.bringToFront();

  await expect(page.locator("input[type=email]")).toHaveValue("anishatiwari695@gmail.com");
  await other.close();
});

test("switching away and back inside the app keeps filters and the typed query", async ({ context, page }) => {
  await page.goto("/");
  await page.fill("input[type=email]", "anishatiwari695@gmail.com");
  await page.fill("input[type=password]", "demo-password-change-me");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.locator(".rail-user-name")).toBeVisible();

  await page.goto("/campaigns?category=Fintech");
  await page.fill("input[type=search]", "cred");

  const other = await context.newPage();
  await other.goto("about:blank");
  await other.bringToFront();
  await page.bringToFront();

  await expect(page.locator("input[type=search]")).toHaveValue("cred");
  expect(new URL(page.url()).search).toContain("category=Fintech");
  await other.close();
});
