import { expect, test, type Page } from "@playwright/test";

const APPROVER = "anishatiwari695@gmail.com";
const PASSWORD = "demo-password-change-me";

async function signIn(page: Page) {
  await page.goto("/");
  await page.fill("input[type=email]", APPROVER);
  await page.fill("input[type=password]", PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.locator(".account-name")).toBeVisible();
}

// These are real, named people, so this spec guards the two things that
// make showing them defensible: the photo credit the licence requires,
// and the absence of any invented metric beside their name.
test("the public record shows real people, their sourced brand work and the photo credit", async ({ page }) => {
  await signIn(page);
  await page.goto("/endorsements");

  const cards = page.locator(".endorsement-card");
  await expect(cards.first()).toBeVisible();
  expect(await cards.count()).toBeGreaterThan(0);

  // Every photograph actually resolves — a broken one here is a person
  // with no face and a credit for nothing. Polled rather than sampled:
  // read once straight after navigation this is a measurement of how far
  // the loads have got, not of whether they succeed.
  await expect
    .poll(
      () =>
        page.evaluate(() => {
          const imgs = [...document.querySelectorAll<HTMLImageElement>(".endorsement-photo")];
          return imgs.length > 0 && imgs.every((i) => i.complete && i.naturalWidth > 0);
        }),
      { timeout: 15000 },
    )
    .toBe(true);

  await page.locator(".endorsement-card", { hasText: "Virat Kohli" }).click();
  await expect(page.getByRole("heading", { level: 1, name: "Virat Kohli" })).toBeVisible();
  await expect(page.locator(".endorsement-row").first()).toBeVisible();

  // Attribution is a licence condition, not decoration.
  const credit = page.locator(".endorsement-credit");
  await expect(credit).toContainText("Photograph by");
  await expect(credit.getByRole("link", { name: "Wikimedia Commons" })).toBeVisible();

  // And nothing from the agency's invented book has leaked onto a real
  // person's page.
  const body = (await page.locator(".endorsement-detail-page").innerText()).toLowerCase();
  for (const forbidden of ["spend to review", "under-delivering", "over budget", "reach outlier", "approve"]) {
    expect(body).not.toContain(forbidden);
  }
});
