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

  // Video-first: the spotlight is the film, so the chart and the stat
  // grid are not on the card at all.
  await expect(page.locator(".in-focus-card .in-focus-chart")).toHaveCount(0);
  await expect(page.locator(".in-focus-card .in-focus-stats")).toHaveCount(0);

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

  // Which campaigns reach the spotlight depends on the data, and every
  // one of them may well have a film — that is the good case, not a
  // broken test. So find one without, and skip honestly if there is
  // none rather than assert against whatever happens to be showing.
  let found = false;
  for (const tab of await page.locator(".in-focus-thumb").all()) {
    await tab.click();
    if ((await page.locator(".ad-film").count()) === 0) {
      found = true;
      break;
    }
  }
  test.skip(!found, "every campaign in the spotlight has a verified film");

  // No stand-in and no placeholder: the card it always had, numbers
  // and chart intact.
  await expect(page.locator(".in-focus-chart")).toBeVisible();
  await expect(page.locator(".in-focus-stats")).toBeVisible();
});

test("rotation holds while a film is playing", async ({ page }) => {
  await openSpotlight(page);
  test.skip(!(await selectCampaign(page, /Dream11/)), "Dream11 is not in the spotlight right now");

  await page.locator(".ad-film-poster").click();
  await expect(page.locator(".ad-film-player")).toHaveCount(1);

  // Read the name off the active tab rather than the card: the card's
  // markup differs between the film and no-film layouts, the strip
  // does not.
  const title = page.locator(".in-focus-thumb-active .in-focus-thumb-name");
  const before = await title.innerText();
  // Leave the card: hover no longer holds it, so only the playing film
  // can. Without that the carousel would rotate out from under a video
  // the person is watching.
  await page.mouse.move(5, 5);
  await page.waitForTimeout(3000);
  await expect(title).toHaveText(before);
});
