import { expect, test, type Page } from "@playwright/test";

const PASSWORD = "demo-password-change-me";
const ORIGINAL_NAME = "Anisha T.";

async function signIn(page: Page) {
  await page.goto("/");
  await page.fill("input[type=email]", "anishatiwari695@gmail.com");
  await page.fill("input[type=password]", PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.waitForSelector(".account-chip");
  await page.goto("/settings");
  await page.waitForSelector(".settings-panel");
}

test("Appearance is gone; the theme lives in one place now", async ({ page }) => {
  await signIn(page);
  await expect(page.getByRole("heading", { name: "Appearance" })).toHaveCount(0);
  // The density toggle that shared that card survived the move.
  await expect(page.getByRole("switch", { name: "Compact tables" })).toBeVisible();
  // And the theme is not offered twice.
  await expect(page.locator(".settings-segmented")).toHaveCount(0);
});

test("the display name can actually be changed", async ({ page }) => {
  await signIn(page);
  const field = page.getByRole("textbox", { name: "Display name" });
  const save = page.getByRole("button", { name: "Save name" });

  // Nothing to save until something changes.
  await expect(save).toBeDisabled();

  await field.fill("Anisha Tiwari");
  await expect(save).toBeEnabled();
  await save.click();

  // It is the same name the rest of the app reads, not a local copy.
  await expect(page.locator(".account-name")).toHaveText("Anisha Tiwari");

  // Put it back, or every later test in the run sees the new one.
  await field.fill(ORIGINAL_NAME);
  await save.click();
  await expect(page.locator(".account-name")).toHaveText(ORIGINAL_NAME);
});

test("an empty name is refused", async ({ page }) => {
  await signIn(page);
  const field = page.getByRole("textbox", { name: "Display name" });
  await field.fill("   ");
  // Blank is not a change worth offering to save — the name is on
  // every audit line this account has written.
  await expect(page.getByRole("button", { name: "Save name" })).toBeDisabled();
  await expect(page.locator(".account-name")).toHaveText(ORIGINAL_NAME);
});

// Deliberately only the refusal: actually changing it would lock every
// other test in the run out of the account.
test("changing the password requires the current one", async ({ page }) => {
  await signIn(page);
  await page.getByRole("textbox", { name: "Current password" }).fill("not-the-password");
  await page.getByRole("textbox", { name: "New password", exact: true }).fill("a-long-enough-one");
  await page.getByRole("textbox", { name: /Confirm new password/ }).fill("a-long-enough-one");
  await page.getByRole("button", { name: "Change password" }).click();

  await expect(page.locator(".settings-error")).toContainText("not your current password");
});

test("a mismatched confirmation never reaches the server", async ({ page }) => {
  await signIn(page);
  await page.getByRole("textbox", { name: "Current password" }).fill(PASSWORD);
  await page.getByRole("textbox", { name: "New password", exact: true }).fill("one-long-password");
  await page.getByRole("textbox", { name: /Confirm new password/ }).fill("a-different-one");

  let sent = false;
  page.on("request", (r) => {
    if (r.url().includes("/api/me/password")) sent = true;
  });
  await page.getByRole("button", { name: "Change password" }).click();
  await expect(page.locator(".settings-error")).toContainText("do not match");
  expect(sent, "the mismatch was sent to the server, which has no second field to compare").toBe(false);
});
