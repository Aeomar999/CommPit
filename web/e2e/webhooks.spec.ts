import http from "http";
import { expect, test, type AddressInfo } from "@playwright/test";

interface CaughtWebhook {
  url: string;
  headers: Record<string, string | string[] | undefined>;
  body: string;
}

// An inbound message reaches the project's inbound webhook, and the
// inspector's replay button re-queues the delivery.
test("reply reaches the inbound webhook and replay re-queues", async ({
  page,
  request,
}) => {
  const received: CaughtWebhook[] = [];
  const server = http.createServer((req, res) => {
    let body = "";
    req.on("data", (chunk) => {
      body += chunk;
    });
    req.on("end", () => {
      received.push({ url: req.url || "", headers: req.headers, body });
      res.writeHead(200, { "Content-Type": "text/plain" });
      res.end("ok");
    });
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const port = (server.address() as AddressInfo).port;
  const hookUrl = `http://127.0.0.1:${port}/hook`;
  test.info().annotations.push({ type: "hook", description: hookUrl });

  try {
    const projects = await request.get("/api/v1/projects", {
      headers: { "X-Mocksms": "true" },
    });
    expect(projects.ok()).toBeTruthy();
    const listed = await projects.json();
    const project = listed.projects.find((p: { name: string }) => p.name === "default") || listed.projects[0];
    const projectId = project.id as string;

    // Give the default project a native API key so webhooks are signed,
    // then point its inbound URL at the catcher.
    const key = `pwkey-${Date.now()}`;
    const link = await request.post(`/api/v1/projects/${projectId}/credentials`, {
      headers: { "X-Mocksms": "true" },
      data: { provider: "native", key },
    });
    expect(link.ok()).toBeTruthy();
    const current = await request.get(`/api/v1/projects/${projectId}`, {
      headers: { "X-Mocksms": "true" },
    });
    const settings = ((await current.json()).settings || {}) as Record<string, unknown>;
    const patched = await request.patch(`/api/v1/projects/${projectId}`, {
      headers: { "X-Mocksms": "true" },
      data: { settings: { ...settings, "native.inbound_url": hookUrl } },
    });
    expect(patched.ok()).toBeTruthy();

    const stamp = Date.now();
    const inbound = await request.post("/api/v1/inbound", {
      headers: { "X-Mocksms": "true" },
      data: { from: "+15005550113", to: "+15555550100", body: `hook check ${stamp}` },
    });
    expect(inbound.status()).toBe(201);

    await expect
      .poll(() => received.length, { timeout: 15_000 })
      .toBeGreaterThan(0);
    const first = JSON.parse(received[0].body) as { event: string };
    expect(first.event).toBe("message.inbound");
    expect(received[0].headers["x-mocksms-signature"]).toMatch(/^sha256=/);

    // The delivery shows in the inspector log and replay re-queues it.
    await page.goto("/inspector");
    await page.getByRole("button", { name: "Webhooks Log" }).click();
    await expect(page.getByText(hookUrl).first()).toBeVisible({ timeout: 10_000 });
    const before = received.length;
    await page.getByRole("button", { name: "Replay" }).first().click();
    await expect(page.getByText("Re-queued as a fresh delivery.").first()).toBeVisible({
      timeout: 10_000,
    });
    await expect
      .poll(() => received.length, { timeout: 15_000 })
      .toBeGreaterThan(before);
  } finally {
    server.close();
  }
});
