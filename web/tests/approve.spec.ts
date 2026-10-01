import { expect, test } from "@playwright/test";

// The one end-to-end smoke test the spec calls for: approve a campaign
// starting from Overview, and prove the decision propagates everywhere —
// the approvals queue, the rail badge, and the overview's own counts.
// Assumes a freshly seeded database (`make seed`), which leaves 5 pending.
test("approving from overview updates the queue, the rail badge and the overview counts", async ({ page }) => {
  await page.goto("/overview");

  const lede = page.locator(".overview-lede");
  await expect(lede).toBeVisible();

  const ledeText = (await lede.textContent()) ?? "";
  const pendingBefore = Number(ledeText.match(/^(\d+)\s+campaign/)?.[1] ?? "0");
  expect(pendingBefore, "seed the database before running e2e (make seed)").toBeGreaterThan(0);

  const badge = page.locator(".rail-item-badge");
  await expect(badge).toHaveText(String(pendingBefore));

  // Overview's primary action takes you into the queue.
  await page.getByRole("link", { name: `Review ${pendingBefore} pending` }).click();
  await expect(page).toHaveURL(/\/approvals$/);

  const cards = page.locator(".approval-card");
  await expect(cards).toHaveCount(pendingBefore);

  const firstCardName = await cards.first().locator(".approval-card-name").textContent();

  await cards.first().getByRole("button", { name: "Approve" }).click();

  // Optimistic update removes the card immediately and toasts.
  await expect(page.locator(".toast-positive")).toContainText("Campaign approved.");
  await expect(cards).toHaveCount(pendingBefore - 1);
  await expect(page.locator(".approval-card-name").first()).not.toHaveText(firstCardName ?? "");

  // The rail badge reflects the new pending count.
  if (pendingBefore - 1 > 0) {
    await expect(badge).toHaveText(String(pendingBefore - 1));
  } else {
    await expect(badge).toHaveCount(0);
  }

  // And the overview agrees after a refetch.
  await page.goto("/overview");
  await expect(page.locator(".overview-lede")).toContainText(`${pendingBefore - 1} campaign`);
});

// The "Your call" ribbon decides in place, so a decision never requires
// leaving Overview at all.
test("approving inline from the Your call ribbon updates the ribbon, badge and lede", async ({ page }) => {
  await page.goto("/overview");

  const ribbon = page.locator("section.ribbon").filter({ hasText: "Your call" });
  const tiles = ribbon.locator(".tile");

  const before = await tiles.count();
  expect(before, "seed the database before running e2e (make seed)").toBeGreaterThan(0);
  await expect(ribbon.locator(".ribbon-count")).toHaveText(String(before));

  const firstName = await tiles.first().locator(".tile-name").textContent();

  await tiles.first().getByRole("button", { name: "Approve" }).click();

  await expect(page.locator(".toast-positive")).toContainText("Campaign approved.");
  await expect(tiles).toHaveCount(before - 1);
  if (before - 1 > 0) {
    await expect(tiles.first().locator(".tile-name")).not.toHaveText(firstName ?? "");
    await expect(ribbon.locator(".ribbon-count")).toHaveText(String(before - 1));
  }
  await expect(page.locator(".overview-lede")).toContainText(`${before - 1} campaign`);
  await expect(page.locator(".rail-item-badge")).toHaveText(String(before - 1));
});
