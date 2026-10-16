import { expect, test, type Page } from "@playwright/test";

const PASSWORD = "demo-password-change-me";

async function signIn(page: Page) {
  await page.goto("/");
  await page.fill("input[type=email]", "anishatiwari695@gmail.com");
  await page.fill("input[type=password]", PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.waitForSelector(".command-bar-theme");
}

test("the theme thumb slides to whichever option is chosen", async ({ page }) => {
  await signIn(page);
  const thumb = page.locator(".command-bar-theme-thumb");
  const at = async () => (await thumb.boundingBox())!.x;

  await page.getByRole("radio", { name: "Auto" }).click();
  const auto = await at();
  await page.getByRole("radio", { name: "Light" }).click();
  await expect.poll(at).toBeGreaterThan(auto);
  const light = await at();
  await page.getByRole("radio", { name: "Dark" }).click();
  await expect.poll(at).toBeGreaterThan(light);

  // And it is driving the real thing, not just sliding.
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.getByRole("radio", { name: "Dark" })).toHaveAttribute("aria-checked", "true");
});

test("the search hint cycles through what is searchable", async ({ page }) => {
  await signIn(page);
  const word = page.locator(".command-bar-search-word");
  const first = await word.innerText();
  await expect.poll(async () => word.innerText(), { timeout: 15000 }).not.toBe(first);

  // The button's accessible name must not be the cycling word, or a
  // screen reader is told the control renamed itself every few
  // seconds.
  await expect(page.getByRole("button", { name: /Search campaigns, creators/ })).toBeVisible();
});

test.describe("on a phone", () => {
  test.use({ viewport: { width: 390, height: 844 } });

  // The old narrow-screen hack hid two of three options and made the
  // survivor cycle, because three words would not fit. Three icons
  // do, so all three must be reachable directly.
  test("all three theme options are reachable, and the breadcrumb is not squeezed", async ({ page }) => {
    await signIn(page);
    for (const name of ["Auto", "Light", "Dark"]) {
      await expect(page.getByRole("radio", { name })).toBeVisible();
    }
    await page.getByRole("radio", { name: "Light" }).click();
    await expect(page.locator("html")).toHaveAttribute("data-theme", "light");

    // "Overview" fits now rather than being clipped to "Overv".
    await expect(page.locator(".command-bar-breadcrumbs")).toContainText("Overview");
    const over = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    );
    expect(over).toBeLessThanOrEqual(1);
  });
});
