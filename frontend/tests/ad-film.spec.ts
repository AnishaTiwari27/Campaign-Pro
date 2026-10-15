import { expect, test } from "@playwright/test";

const PASSWORD = "demo-password-change-me";

async function openSpotlight(page: import("@playwright/test").Page) {
  await page.goto("/");
  await page.fill("input[type=email]", "anishatiwari695@gmail.com");
  await page.fill("input[type=password]", PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.waitForSelector(".in-focus");
  // Hover holds the 30s rotation still for the length of the test.
  await page.hover(".in-focus");
}

async function selectCampaign(page: import("@playwright/test").Page, name: RegExp) {
  for (const tab of await page.locator(".in-focus-thumb").all()) {
    if (name.test((await tab.getAttribute("aria-label")) ?? "")) {
      await tab.click();
      return true;
    }
  }
  return false;
}

test("the spotlight plays the brand's own ad film, and loads nothing until asked", async ({ page }) => {
  await openSpotlight(page);
  test.skip(!(await selectCampaign(page, /Dream11/)), "Dream11 is not in the spotlight right now");

  const film = page.locator(".ad-film");
  await expect(film).toBeVisible();

  // Attribution is the licensing argument, so it is not optional
  // decoration — it states whose upload this is.
  await expect(film).toContainText("Published by Dream11 on YouTube");

  // The facade: no YouTube iframe exists before anyone asks for one, so
  // a visitor who never watches loads no player and gets no cookies.
  await expect(page.locator(".ad-film-player")).toHaveCount(0);

  await page.locator(".ad-film-poster").click();
  const player = page.locator(".ad-film-player");
  await expect(player).toHaveCount(1);
  await expect(player).toHaveAttribute("src", /youtube-nocookie\.com\/embed\//);
});

test("a campaign with no verified film keeps the card it had", async ({ page }) => {
  await openSpotlight(page);
  test.skip(!(await selectCampaign(page, /Blinkit/)), "Blinkit is not in the spotlight right now");

  // No stand-in, no placeholder: the chart is simply the whole column.
  await expect(page.locator(".ad-film")).toHaveCount(0);
  await expect(page.locator(".in-focus-chart")).toBeVisible();
});

test("rotation holds while a film is playing", async ({ page }) => {
  await openSpotlight(page);
  test.skip(!(await selectCampaign(page, /Dream11/)), "Dream11 is not in the spotlight right now");

  await page.locator(".ad-film-poster").click();
  await expect(page.locator(".ad-film-player")).toHaveCount(1);

  const title = page.locator(".in-focus-title-row h2");
  const before = await title.innerText();
  // Leave the card: hover no longer holds it, so only the playing film
  // can. Without that the carousel would rotate out from under a video
  // the person is watching.
  await page.mouse.move(5, 5);
  await page.waitForTimeout(3000);
  await expect(title).toHaveText(before);
});
