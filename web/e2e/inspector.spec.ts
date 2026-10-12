import { expect, test } from "@playwright/test";

// Adapter traffic shows up in the inspector with a raw request/response
// detail view.
test("twilio adapter request is logged in the inspector", async ({ page, request }) => {
  const accountSid = `AC${"0".repeat(32)}`;
  const stamp = Date.now();

  const projects = await request.get("/api/v1/projects", {
    headers: { "X-Mocksms": "true" },
  });
  const listed = await projects.json();
  const project = listed.projects.find((p: { name: string }) => p.name === "default") || listed.projects[0];

  const link = await request.post(`/api/v1/projects/${project.id}/credentials`, {
    headers: { "X-Mocksms": "true" },
    data: { provider: "twilio", key: accountSid },
  });
  expect(link.ok()).toBeTruthy();

  const form = new URLSearchParams({
    To: "+15005550117",
    From: "+15555550100",
    Body: `inspector check ${stamp}`,
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

  await page.goto("/inspector");
  const row = page
    .getByRole("button")
    .filter({ hasText: `/2010-04-01/Accounts/${accountSid}/Messages.json` })
    .first();
  await expect(row).toBeVisible({ timeout: 10_000 });
  await row.click();
  await expect(page.getByText("Response · 201").first()).toBeVisible({
    timeout: 10_000,
  });
  await expect(page.getByText("+15005550117").first()).toBeVisible({
    timeout: 10_000,
  });
});
