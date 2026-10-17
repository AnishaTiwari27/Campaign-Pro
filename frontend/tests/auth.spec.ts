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
  await expect(page.getByRole("heading", { name: "Welcome back" })).toBeVisible();
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
  await expect(page.locator(".account-name")).toBeVisible();

  // Signing out moved into the account menu with everything else
  // about the account.
  await page.locator(".account-chip").click();
  await page.getByRole("menuitem", { name: "Log out" }).click();
  await expect(page.getByRole("heading", { name: "Welcome back" })).toBeVisible();

  // The session is gone server-side, so returning to a dashboard URL must
  // not get back in.
  await page.goto("/campaigns");
  await expect(page.getByRole("heading", { name: "Welcome back" })).toBeVisible();
});

test("an analyst can read everything but is shown no approval controls", async ({ page }) => {
  await page.goto("/");
  await signIn(page, ANALYST);
  await expect(page.locator(".account-role")).toHaveText("Analyst");

  // Reading is fine.
  await expect(page.locator(".in-focus-card")).toBeVisible();

  // Deciding is not offered anywhere it would be for an approver.
  await expect(page.locator('.tile-actions button:has-text("Approve")')).toHaveCount(0);
  await page.goto("/approvals");
  await expect(page.locator(".approval-card")).not.toHaveCount(0);
  await expect(page.locator('.approval-card-actions button')).toHaveCount(0);
});

// Registering and then signing in, through the form, because that is how
// the failure was reported: correct details, and a 500. The server-side
// cause is covered by cmd/server's own test; this is the half that proves
// the form reaches it and shows what came back.
test("a stranger can register and then sign in with the password they chose", async ({ page }) => {
  const email = `e2e-signup-${Date.now()}@example.test`;
  const password = "correct horse battery staple";

  await page.goto("/");
  await page.getByRole("button", { name: "Create an account" }).click();
  await expect(page.getByRole("heading", { name: "Create your account" })).toBeVisible();

  await page.fill("input[type=text]", "E2E Signup");
  await page.fill("input[type=email]", email);
  const passwords = page.locator("input[type=password]");
  await passwords.nth(0).fill(password);
  await passwords.nth(1).fill(password);
  await page.getByRole("button", { name: "Create account" }).click();

  // Signup deliberately issues no session: it hands the address back to
  // the sign-in form, which is where the password gets proved.
  await expect(page.getByRole("status")).toContainText("Account created");
  await expect(page.locator("input[type=email]")).toHaveValue(email);

  await signIn(page, email, password);
  await expect(page.locator(".account-name")).toBeVisible();

  // An analyst, which is the whole security question for self-registration.
  await expect(page.getByRole("button", { name: "Approve" })).toHaveCount(0);
});
