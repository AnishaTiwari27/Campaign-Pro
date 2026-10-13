import { expect, test, type Page } from "@playwright/test";

// Credentials come from `make seed`, as in the other specs.
const APPROVER = "anishatiwari695@gmail.com";
const PASSWORD = "demo-password-change-me";

async function signIn(page: Page) {
  await page.goto("/");
  await page.fill("input[type=email]", APPROVER);
  await page.fill("input[type=password]", PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.locator(".rail-user-name")).toBeVisible();
}

// A tile is one campaign, but the person on it has a body of work. This
// walks the path someone actually takes — face, then person, then one of
// their ads, then back to the person — because each hop crosses a service
// boundary (campaigns knows a creator id; only creators knows the name)
// and nothing else would catch it if the resolution silently returned
// nothing. The byline simply would not render, and the page would look
// fine.
test("a tile's byline opens the person, their campaign list, and leads back", async ({ page }) => {
  await signIn(page);
  await page.goto("/overview");

  const tile = page.locator(".tile", { has: page.locator(".tile-byline") }).first();
  await expect(tile).toBeVisible();

  const byline = tile.locator(".tile-byline");
  const person = (await byline.innerText()).replace("›", "").trim();
  expect(person.length).toBeGreaterThan(0);

  // The byline must not also fire the tile's own navigation to the campaign.
  await byline.click();
  await expect(page).toHaveURL(/\/creators\/[a-z-]+$/);
  await expect(page.getByRole("heading", { level: 1, name: person })).toBeVisible();

  // Every campaign they have fronted, each showing the ad itself and not
  // just the brand it ran for.
  const rows = page.locator(".creator-campaign-row");
  const n = await rows.count();
  expect(n).toBeGreaterThan(0);
  await expect(page.locator(".creator-campaign-headline").first()).toBeVisible();

  // The headlines are per-campaign, not one template repeated.
  const headlines = await page.locator(".creator-campaign-headline").allTextContents();
  expect(new Set(headlines).size).toBe(headlines.length);

  await rows.first().click();
  await expect(page).toHaveURL(/\/campaigns\//);

  // And the campaign names the person, so the loop closes.
  await page.locator(".campaign-detail-subtitle a", { hasText: person }).click();
  await expect(page).toHaveURL(/\/creators\//);
  await expect(page.getByRole("heading", { level: 1, name: person })).toBeVisible();
});
