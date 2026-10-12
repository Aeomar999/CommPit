import { expect, test } from "@playwright/test";

// A sent SMS shows up live in the open inbox without a page reload.
test("sent SMS arrives live in the inbox", async ({ page, request }) => {
  const body = `live arrival ${Date.now()}`;
  await page.goto("/inbox");

  const send = await request.post("/api/v1/sms", {
    headers: { "X-Mocksms": "true" },
    data: { from: "+15555550100", to: "+15005550111", body },
  });
  expect(send.status()).toBe(201);

  await expect(page.getByText(body).first()).toBeVisible({ timeout: 20_000 });
});
