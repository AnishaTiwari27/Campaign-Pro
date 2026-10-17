import { expect, test, type Page } from "@playwright/test";

const PASSWORD = "demo-password-change-me";

async function signIn(page: Page) {
  await page.goto("/");
  await page.fill("input[type=email]", "anishatiwari695@gmail.com");
  await page.fill("input[type=password]", PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.waitForSelector(".account-chip");
}

test("the account menu opens, nests, and closes", async ({ page }) => {
  await signIn(page);
  await expect(page.locator(".account-menu")).toHaveCount(0);

  await page.locator(".account-chip").click();
  await expect(page.locator(".account-menu")).toBeVisible();

  await page.getByRole("menuitem", { name: "Account" }).hover();
  const flyout = page.locator(".account-flyout");
  await expect(flyout).toBeVisible();

  // The rail scrolls, so it clips anything leaving it. The popover is
  // portalled to the body for exactly this reason; rendered inside the
  // rail the flyout is invisible even though it exists in the DOM.
  const box = await flyout.boundingBox();
  const rail = await page.locator(".rail").boundingBox();
  expect(box!.x).toBeGreaterThan(rail!.x + rail!.width - 60);
  await expect(flyout.getByText("Plan & billing")).toBeVisible();

  await page.keyboard.press("Escape");
  await expect(page.locator(".account-menu")).toHaveCount(0);
});

test("what is built goes somewhere; what is not says so", async ({ page }) => {
  await signIn(page);
  await page.locator(".account-chip").click();
  await page.getByRole("menuitem", { name: "Account" }).hover();

  await page.getByRole("menuitem", { name: "Plan & billing" }).click();
  await expect(page.locator(".billing-plans")).toBeVisible();

  // An unbuilt item is listed, but never dressed up as working.
  await page.locator(".account-chip").click();
  await expect(page.getByRole("menuitem", { name: /Notifications/ })).toContainText("Soon");
  await page.getByRole("menuitem", { name: /Notifications/ }).click();
  await expect(page.getByText("Not built yet")).toBeVisible();
});

test("login history shows real sessions and marks the current one", async ({ page }) => {
  await signIn(page);
  await page.locator(".account-chip").click();
  await page.getByRole("menuitem", { name: "Account" }).hover();
  await page.getByRole("menuitem", { name: "Login history" }).click();

  const rows = page.locator(".signin");
  await expect(rows.first()).toBeVisible();

  // Exactly one row is this session — it is matched by token hash, so
  // more than one would mean the comparison is wrong.
  await expect(page.locator(".signin-current")).toHaveCount(1);
});

test("the theme switch moved into the menu, and still works", async ({ page }) => {
  await signIn(page);
  // Two controls for one setting, both on screen, is how a settings
  // menu rots — the command bar's copy is gone.
  await expect(page.locator(".command-bar-theme")).toHaveCount(0);

  await page.locator(".account-chip").click();
  await page.locator(".account-theme").getByRole("radio", { name: "Light" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.locator(".account-theme").getByRole("radio", { name: "Dark" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});
