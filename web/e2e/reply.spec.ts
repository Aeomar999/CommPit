import { expect, test } from "@playwright/test";

// The reply box under an SMS thread sends as the handset and the reply
// appears back in the inbox.
test("reply box sends an inbound reply", async ({ page, request }) => {
  const stamp = Date.now();
  const outbound = `reply seed ${stamp}`;
  const send = await request.post("/api/v1/sms", {
    headers: { "X-Mocksms": "true" },
    data: { from: "+15555550100", to: "+15005550112", body: outbound },
  });
  expect(send.status()).toBe(201);

  await page.goto("/inbox");
  await page.getByRole("button").filter({ hasText: outbound }).first().click();

  const replyBody = `reply back ${stamp}`;
  await page.getByLabel("Reply text").fill(replyBody);
  await page.getByRole("button", { name: "Send reply" }).click();

  await expect(page.getByText(replyBody).first()).toBeVisible({ timeout: 10_000 });
});
