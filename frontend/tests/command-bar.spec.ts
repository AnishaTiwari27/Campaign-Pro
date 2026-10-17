import { expect, test, type Page } from "@playwright/test";

const PASSWORD = "demo-password-change-me";

async function signIn(page: Page) {
  await page.goto("/");
  await page.fill("input[type=email]", "anishatiwari695@gmail.com");
  await page.fill("input[type=password]", PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.waitForSelector(".command-bar-search");
}

test("the search hint cycles through what is searchable", async ({ page }) => {
  await signIn(page);
  const word = page.locator(".command-bar-search-word");
  const first = await word.innerText();
  await expect.poll(async () => word.innerText(), { timeout: 15000 }).not.toBe(first);

  // The accessible name must not be the cycling word, or a screen
  // reader is told the control renamed itself every few seconds.
  await expect(page.getByRole("button", { name: /Search campaigns, creators/ })).toBeVisible();
});

test.describe("on a phone", () => {
  test.use({ viewport: { width: 390, height: 844 } });

  // The theme switch used to live here in three labelled options,
  // which cost about 130px and squeezed the breadcrumb to "Camp". It
  // is in the account menu now, so the bar has the room either way.
  test("the breadcrumb is not squeezed", async ({ page }) => {
    await signIn(page);
    await expect(page.locator(".command-bar-breadcrumbs")).toContainText("Overview");
    const over = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    );
    expect(over).toBeLessThanOrEqual(1);
  });

  test("the account menu is still reachable where the rail is collapsed", async ({ page }) => {
    await signIn(page);
    await page.locator(".account-chip").click();
    await expect(page.locator(".account-menu")).toBeVisible();
    // Positioned from the chip's measured rectangle, so it has to land
    // on screen at this width too rather than off the left edge.
    const box = await page.locator(".account-menu").boundingBox();
    expect(box!.x).toBeGreaterThanOrEqual(0);
    expect(box!.x + box!.width).toBeLessThanOrEqual(390);
  });
});
