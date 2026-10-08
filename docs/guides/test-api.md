# Test API Guide

Deterministic assertions for E2E tests: no sleeps, no races, parallel-safe.
All endpoints live under `/api/v1` and accept `Authorization: Bearer <key>`
or `?project=<id>` (reads and test helpers; sends without a key use the
`default` project).

Start the server with a fixed OTP code to make verification flows fully
deterministic:

```bash
mocksms --memory --otp-code 123456 &
```

## `GET /messages/wait` — long-poll for a message

Returns the first message matching the filters created after `since`, or
`408 wait_timeout`. `timeout` accepts seconds (`10`) or a Go duration
(`10s`), defaults to 10 s, caps at 60 s. `since` defaults to the moment the
request arrives — record it **before** triggering the flow under test.

```bash
SINCE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
# ... trigger your app to send an SMS ...
curl "http://127.0.0.1:4010/api/v1/messages/wait?to=%2B15005550006&channel=sms&since=$SINCE&timeout=10"
```

## `GET /otp/latest` — read the newest OTP

Newest outbound code for a recipient: the verification code with linkage
(`source: verification`, plus `verification_id`) when the message carries
one, else the first extracted code (`source: extracted`). An omitted `since`
means no lower bound. `404` when nothing matches.

```bash
curl "http://127.0.0.1:4010/api/v1/otp/latest?to=%2B15005550006"
# {"code":"123456","source":"verification","message_id":"msg_…","verification_id":"vrf_…"}
```

## `GET /emails/latest` — read the newest email

Newest outbound email with its codes, links and primary link. `404` when
nothing matches.

```bash
curl "http://127.0.0.1:4010/api/v1/emails/latest?to=user%40example.com"
```

## Reset and expiry helpers

```bash
# Expire a verification to test the expired path
curl -X POST http://127.0.0.1:4010/api/v1/verifications/vrf_…/expire
# Wipe a project between tests (requires explicit project)
curl -X DELETE "http://127.0.0.1:4010/api/v1/messages?project=<id>"
```

## Playwright

```ts
// otp.spec.ts
import { expect, test } from "@playwright/test";

const API = process.env.MOCKSMS_URL ?? "http://127.0.0.1:4010";
const TO = "+15005550006";

test("signup OTP flow", async ({ page, request }) => {
  const since = new Date().toISOString();
  await page.goto(process.env.APP_URL ?? "http://localhost:3000/signup");
  await page.getByLabel("Phone").fill(TO);
  await page.getByRole("button", { name: "Send code" }).click();

  const wait = await request.get(`${API}/api/v1/messages/wait`, {
    params: { to: TO, channel: "sms", since, timeout: 10 },
  });
  expect(wait.ok()).toBeTruthy();

  const latest = await request.get(`${API}/api/v1/otp/latest`, { params: { to: TO } });
  const { code } = await latest.json();
  await page.getByLabel("Code").fill(code);
  await page.getByRole("button", { name: "Verify" }).click();
  await expect(page.getByText("Welcome")).toBeVisible();
});
```

## Cypress

```ts
// cypress/e2e/otp.cy.ts
const API = Cypress.env("MOCKSMS_URL") ?? "http://127.0.0.1:4010";
const TO = "+15005550006";

it("signup OTP flow", () => {
  const since = new Date().toISOString();
  cy.visit("/signup");
  cy.get("[aria-label=Phone]").type(TO);
  cy.contains("button", "Send code").click();

  cy.request({
    url: `${API}/api/v1/messages/wait`,
    qs: { to: TO, channel: "sms", since, timeout: 10 },
  }).then((wait) => {
    expect(wait.status).to.eq(200);
    cy.request(`${API}/api/v1/otp/latest?to=${encodeURIComponent(TO)}`).then(
      (latest) => {
        cy.get("[aria-label=Code]").type(latest.body.code);
        cy.contains("button", "Verify").click();
        cy.contains("Welcome").should("be.visible");
      }
    );
  });
});
```

## Jest

```ts
// otp.test.ts (server started with --otp-code 123456)
const API = process.env.MOCKSMS_URL ?? "http://127.0.0.1:4010";
const TO = "+15005550006";

test("verification approves with the latest OTP", async () => {
  const since = new Date().toISOString();
  await triggerSignupSms(TO); // your app's send path

  const wait = await fetch(
    `${API}/api/v1/messages/wait?to=${encodeURIComponent(TO)}&since=${since}&timeout=10`
  );
  expect(wait.status).toBe(200);

  const latest = await fetch(`${API}/api/v1/otp/latest?to=${encodeURIComponent(TO)}`);
  expect(latest.status).toBe(200);
  const { code, source, verification_id } = await latest.json();
  expect(source).toBe("verification");

  const check = await fetch(`${API}/api/v1/verifications/${verification_id}/check`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ code }),
  });
  expect((await check.json()).valid).toBe(true);
});
```

Calls without credentials share the `default` project; pass `?project=<id>`
(or `Authorization: Bearer <key>`) everywhere to isolate parallel workers.
