# CI Guide

Run your messaging tests against mocksms in CI: start the binary, point
the app at it, assert through the test API, reset between suites.

## Minimal job (GitHub Actions)

```yaml
- name: Start mocksms
  run: |
    go run ./cmd/mocksms --otp-code 123456 --data-dir /tmp/mocksms-data serve &
    sleep 3
    curl -f http://127.0.0.1:4010/api/v1/healthz
```

`--otp-code` pins every verification code so assertions are
deterministic. Prefer a fresh `--data-dir` per run for isolation.

## Patterns

**Wait for a message instead of sleeping:**

```bash
curl "http://127.0.0.1:4010/api/v1/messages/wait?to=%2B15005550006&timeout=30" \
  -H "X-Mocksms: true"
```

**Read the latest OTP:**

```bash
mocksms otp latest --to +15005550006 --port 4010
```

**Reset between suites** so one test's STOP unsubscribes and leftovers
never leak into the next:

```bash
mocksms reset --project <prj_…> --port 4010
```

List projects with `GET /api/v1/projects` to find the id, or link
credentials so each suite resolves to its own project.

**Isolate suites with credentials:** give each suite its own API key or
sender credential. Traffic resolves to separate auto-provisioned
projects — no shared state, no cross-talk.

**Failure simulation in CI:** seed `sim.rules` in `mocksms.yaml` for
deterministic error paths (rejections, async failures, rate limits),
or drive magic recipients (`…999902` fails async, `…999905` fails hard,
`*@bounce.mocksms.test` bounces). See the
[simulation guide](simulation.md).

## Playwright

The repo's own suite (`web/e2e/`) is the reference: API calls set up
state, the UI asserts it, webhook receivers run in-process via
`node:http`, and unique bodies per run keep tests independent. Run
serially (`workers: 1` in `playwright.config.ts`) — every spec shares
one server and its default project.

## External pieces

- The `setup-mocksms` GitHub Action (M4-05) will wrap the start/health
  block above into one step; until it ships, inline the snippet.
- Testcontainers modules with `waitForOtp` helpers (M4-06, Node/Go/Python)
  will cover non-Go stacks; until they ship, start the binary directly
  as above.
