import { expect, test } from "@playwright/test";

// The estimate view prices observed traffic scaled to a volume.
test("estimate view prices twilio traffic", async ({ page, request }) => {
  const accountSid = `AC${"1".repeat(32)}`;

  const seed = await request.post("/api/v1/sms", {
    headers: { "X-Mocksms": "true" },
    data: { from: "+15555550100", to: "+15005550121", body: "project probe" },
  });
  expect(seed.status()).toBe(201);
  const projectId = ((await seed.json()) as { project_id: string }).project_id;

  const link = await request.post(`/api/v1/projects/${projectId}/credentials`, {
    headers: { "X-Mocksms": "true" },
    data: { provider: "twilio", key: accountSid },
  });
  expect(link.ok()).toBeTruthy();

  const form = new URLSearchParams({
    To: "+15005550122",
    From: "+15555550100",
    Body: "estimate seed",
  });
  const send = await request.post(
    `/twilio/2010-04-01/Accounts/${accountSid}/Messages.json`,
    {
      headers: {
        Authorization: `Basic ${Buffer.from(`${accountSid}:token`).toString("base64")}`,
        "Content-Type": "application/x-www-form-urlencoded",
      },
      data: form.toString(),
    }
  );
  expect(send.status()).toBe(201);

  await page.goto("/estimate");
  await expect(page.getByText("twilio", { exact: true }).first()).toBeVisible({
    timeout: 20_000,
  });
  await expect(page.getByText("estimate only").first()).toBeVisible({ timeout: 10_000 });
});
