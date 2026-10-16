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

test("the spotlight plays the brand's own ad film, without being asked", async ({ page }) => {
  await openSpotlight(page);
  test.skip(!(await selectCampaign(page, /Dream11/)), "Dream11 is not in the spotlight right now");

  const film = page.locator(".ad-film");
  await expect(film).toBeVisible();

  // Attribution is the licensing argument, so it is not optional
  // decoration — it states whose upload this is.
  await expect(film).toContainText("Published by Dream11 on YouTube");

  // Video-first: the spotlight is the film, so the chart and the stat
  // grid are not on the card at all.
  await expect(page.locator(".in-focus-card .in-focus-chart")).toHaveCount(0);
  await expect(page.locator(".in-focus-card .in-focus-stats")).toHaveCount(0);

  // No poster and no play button: the player is already there.
  const player = page.locator(".ad-film-player");
  await expect(player).toHaveCount(1);
  await expect(page.locator(".ad-film-poster")).toHaveCount(0);
  await expect(player).toHaveAttribute("src", /youtube-nocookie\.com\/embed\//);
  // Muted, or the browser refuses to start it and the film sits on
  // its first frame instead.
  await expect(player).toHaveAttribute("src", /[?&]mute=1/);
  await expect(player).toHaveAttribute("src", /[?&]autoplay=1/);
});

// The claim is that it plays, not merely that an iframe exists, so
// this asks the player how far through it is.
test("the film is actually running, muted", async ({ page }) => {
  await openSpotlight(page);
  await expect(page.locator(".ad-film-player")).toHaveCount(1);

  const video = page.frameLocator(".ad-film-player").locator("video").first();
  await expect(video).toHaveCount(1, { timeout: 20000 });
  await page.waitForTimeout(4000);
  const first = await video.evaluate((v: HTMLVideoElement) => v.currentTime);
  await page.waitForTimeout(2500);
  const state = await video.evaluate((v: HTMLVideoElement) => ({
    t: v.currentTime,
    muted: v.muted,
    paused: v.paused,
  }));
  expect(state.t, "the film is not advancing").toBeGreaterThan(first);
  expect(state.paused).toBe(false);
  expect(state.muted, "an unmuted autoplay would have been blocked").toBe(true);
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

// Playback used to hold the carousel, back when starting a film was a
// deliberate click. Every card plays on arrival now, so that rule
// would mean it never advanced at all — hover is what holds it.
test("rotation keeps going while a film plays, and hover still holds it", async ({ page }) => {
  // The carousel turns every 30s, which is the suite's whole per-test
  // budget — waiting for one turn needs more than the default.
  test.setTimeout(90_000);
  await openSpotlight(page);
  const active = page.locator(".in-focus-thumb-active .in-focus-thumb-name");
  const held = await active.innerText();

  // Still hovering: it must not move.
  await page.waitForTimeout(3000);
  await expect(active).toHaveText(held);

  // Off the card, and with a film playing, it must move on its own.
  await page.mouse.move(5, 5);
  await expect
    .poll(async () => active.innerText(), { timeout: 45000, intervals: [1000] })
    .not.toBe(held);
});
