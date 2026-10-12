import { expect, test } from "@playwright/test";

// A bulk send shows up on the batches view with live status progress,
// and the detail view breaks down counts and rejections.
test("batch progress appears on the batches view", async ({ page, request }) => {
  const stamp = Date.now();
  const send = await request.post("/api/v1/sms", {
    headers: { "X-Mocksms": "true" },
    data: {
      from: "+15555550100",
      to: ["+15005550114", "+15005550115", "+15005550116"],
      body: `batch check ${stamp}`,
    },
  });
  expect(send.status()).toBe(202);
  const batchId = ((await send.json()) as { id: string }).id;
  expect(batchId).toMatch(/^bat_/);

  await page.goto("/batches");
  const row = page.getByRole("button").filter({ hasText: batchId }).first();
  await expect(row).toBeVisible({ timeout: 10_000 });
  await expect(row).toContainText("3 recipients");
  await row.click();
  await expect(page.getByText("Status counts").first()).toBeVisible({ timeout: 10_000 });
  await expect(page.getByText("delivered", { exact: true }).first()).toBeVisible({
    timeout: 15_000,
  });
});
