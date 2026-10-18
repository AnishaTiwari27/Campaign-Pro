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
  // YouTube's own chrome is off: a scrub bar and a centre play button
  // sat permanently over the ad and over this card's header.
  await expect(player).toHaveAttribute("src", /[?&]controls=0/);
});

test("the player's chrome stays off, and our controls only appear on hover", async ({ page }) => {
  await openSpotlight(page);
  const controls = page.locator(".ad-film-controls");
  await expect(controls).toHaveCount(1);

  // Park the pointer away from the film.
  await page.mouse.move(5, 5);
  await expect(controls).toHaveCSS("opacity", "0");

  const box = await page.locator(".ad-film-frame").boundingBox();
  await page.mouse.move(box!.x + box!.width / 2, box!.y + box!.height / 2);
  await expect(controls).toHaveCSS("opacity", "1");

  // The iframe must not take the pointer, or YouTube throws its title
  // and centre controls back up the moment you hover — which no CSS
  // of ours can hide, the player being cross-origin.
  await expect(page.locator(".ad-film-player")).toHaveCSS("pointer-events", "none");
});

test("the mute button reaches the player", async ({ page }) => {
  await openSpotlight(page);
  const video = page.frameLocator(".ad-film-player").locator("video").first();
  await expect(video).toHaveCount(1, { timeout: 20000 });
  expect(await video.evaluate((v: HTMLVideoElement) => v.muted)).toBe(true);

  await page.getByRole("button", { name: /Unmute/ }).click();
  await expect
    .poll(async () => video.evaluate((v: HTMLVideoElement) => v.muted), { timeout: 10000 })
    .toBe(false);
});

// The claim is that it plays, not merely that an iframe exists, so
// this asks the player how far through it is.
test("the film is actually running, muted", async ({ page }) => {
  await openSpotlight(page);
  await expect(page.locator(".ad-film-player")).toHaveCount(1);

  const video = page.frameLocator(".ad-film-player").locator("video").first();
  await expect(video).toHaveCount(1, { timeout: 20000 });

  // Waited for rather than sampled at a fixed moment. The film is
  // fetched from YouTube, so under a full suite run it can still be
  // buffering when a two-second window closes — and "it had not
  // started yet" is not the same failure as "it does not play". The
  // assertion is unchanged: the time has to move.
  const at = async () => video.evaluate((v: HTMLVideoElement) => v.currentTime);
  await expect.poll(at, { timeout: 30000, intervals: [500] }).toBeGreaterThan(0);
  const first = await at();
  await expect
    .poll(at, { timeout: 30000, intervals: [500] })
    .toBeGreaterThan(first);

  const state = await video.evaluate((v: HTMLVideoElement) => ({
    muted: v.muted,
    paused: v.paused,
  }));
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
