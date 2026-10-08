import { expect, test } from "@playwright/test";

// Minimal smoke coverage for the embedded UI. M3-11 grows this into the
// full live-arrival / reply / inspector / batch-progress suite. The server
// under test is started by CI (or locally: bin/mocksms --memory).

test("landing page loads", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveTitle(/mocksms/);
});

test("native API answers under /api/v1", async ({ request }) => {
  const health = await request.get("/api/v1/healthz");
  expect(health.ok()).toBeTruthy();
  const send = await request.post("/api/v1/sms", {
    headers: { "X-Mocksms": "true" },
    data: { from: "+15555550100", to: "+15005550006", body: "playwright smoke" },
  });
  expect(send.status()).toBe(201);
  const created = await send.json();
  expect(created.id).toMatch(/^msg_/);
});
