import { expect, test, type Page } from "@playwright/test";

const APPROVER = "anishatiwari695@gmail.com";
const ANALYST = "analyst@campaigntracker.test";
const PASSWORD = "demo-password-change-me";

async function signIn(page: Page, email: string, password = PASSWORD) {
  await page.fill("input[type=email]", email);
  await page.fill("input[type=password]", password);
  await page.getByRole("button", { name: "Sign in" }).click();
}

test("a signed-out visitor gets the sign-in screen, not the dashboard", async ({ page }) => {
  await page.goto("/overview");
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
  // The dashboard must not render at all — not even briefly behind a modal.
  await expect(page.locator(".rail")).toHaveCount(0);
});

test("a wrong password is rejected without revealing whether the account exists", async ({ page }) => {
  await page.goto("/");
  await signIn(page, APPROVER, "not-the-password");
  await expect(page.getByRole("alert")).toHaveText("invalid email or password");

  // The same message for an address that does not exist at all, so the
  // response cannot be used to enumerate accounts.
  await page.reload();
  await signIn(page, "nobody@campaigntracker.test");
  await expect(page.getByRole("alert")).toHaveText("invalid email or password");
});

test("signing out revokes the session, and back does not restore it", async ({ page }) => {
  await page.goto("/");
  await signIn(page, APPROVER);
  await expect(page.locator(".rail-user-name")).toBeVisible();

  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();

  // The session is gone server-side, so returning to a dashboard URL must
  // not get back in.
  await page.goto("/campaigns");
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
});

test("an analyst can read everything but is shown no approval controls", async ({ page }) => {
  await page.goto("/");
  await signIn(page, ANALYST);
  await expect(page.locator(".rail-user-role")).toHaveText("Analyst");

  // Reading is fine.
  await expect(page.locator(".in-focus-card")).toBeVisible();

  // Deciding is not offered anywhere it would be for an approver.
  await expect(page.locator('.tile-actions button:has-text("Approve")')).toHaveCount(0);
  await page.goto("/approvals");
  await expect(page.locator(".approval-card")).not.toHaveCount(0);
  await expect(page.locator('.approval-card-actions button')).toHaveCount(0);
});
