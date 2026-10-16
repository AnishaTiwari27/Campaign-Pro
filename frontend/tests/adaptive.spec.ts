import { expect, test, type Page } from "@playwright/test";

const PASSWORD = "demo-password-change-me";
const PHONE = { width: 390, height: 844 };

async function signIn(page: Page) {
  await page.goto("/");
  await page.fill("input[type=email]", "anishatiwari695@gmail.com");
  await page.fill("input[type=password]", PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  // Something only the signed-in app has. Waiting on `main` would
  // match the sign-in page's own, and pass without ever signing in.
  await page.waitForSelector("text=What needs you today");
}

/** How far the page sticks out past the viewport. */
async function overflow(page: Page) {
  return page.evaluate(() => {
    const de = document.documentElement;
    return de.scrollWidth - de.clientWidth;
  });
}

test.describe("on a phone", () => {
  test.use({ viewport: PHONE });

  test("no page scrolls sideways", async ({ page }) => {
    await signIn(page);
    for (const route of ["/overview", "/campaigns", "/influencers", "/approvals"]) {
      await page.goto(route);
      await page.waitForTimeout(400);
      expect(await overflow(page), `${route} overflows its viewport`).toBeLessThanOrEqual(1);
    }
  });

  test("a table becomes labelled cards, keeping the sort it would have lost", async ({ page }) => {
    await signIn(page);
    await page.goto("/campaigns");
    await expect(page.locator(".data-table-row").first()).toBeVisible();

    // The header row is gone, so each value carries its own label —
    // otherwise the numbers are unlabelled and the page is a list of
    // figures with nothing saying what they are.
    await expect(page.locator(".data-table thead")).toBeHidden();
    const firstRow = page.locator(".data-table-row").first();
    const labels = await firstRow.evaluate((row) =>
      [...row.querySelectorAll("td[data-label]:not([data-primary])")].map((td) =>
        td.getAttribute("data-label"),
      ),
    );
    expect(labels).toContain("Reach");
    expect(labels).toContain("Status");

    // Sorting lives in the header row on a desktop, so without this
    // the table would silently lose a feature on a phone.
    await expect(page.locator(".data-table-sortbar")).toBeVisible();
  });

  test("the spotlight caption sits below the film rather than over it", async ({ page }) => {
    await signIn(page);
    const film = page.locator(".ad-film-hero .ad-film-frame");
    await expect(film).toBeVisible();
    const frame = await film.boundingBox();
    const caption = await page.locator(".ad-film-hero .ad-film-caption").boundingBox();
    expect(frame && caption).toBeTruthy();
    // Clear of the picture: the film is only ~170px tall here, and an
    // overlaid caption was taking a quarter of it.
    expect(caption!.y).toBeGreaterThanOrEqual(frame!.y + frame!.height - 2);
  });
});

test("on a desktop the table is still a table", async ({ page }) => {
  await signIn(page);
  await page.goto("/campaigns");
  await expect(page.locator(".data-table thead")).toBeVisible();
  await expect(page.locator(".data-table-sortbar")).toBeHidden();
});
