# mocksms — Tasks

Milestone breakdown for Wave 1. Each task is small enough for one focused commit (or a short series of commits) on its milestone branch. Detailed step-by-step implementation plans (test-first) are written per milestone in `docs/superpowers/plans/` before that milestone starts.

**How to use this file**
- **One branch per milestone.** Each milestone has its own branch (`milestone/m1` … `milestone/m4`), created from an up-to-date `main` when the milestone starts. Every task and commit for that milestone is committed and pushed to its branch. Cross-cutting tasks go on the branch of the milestone active at the time.
- **After every task:** run `task test` and `task lint` before committing and pushing. Never commit or push with failing tests.
- **After every milestone:** run the full suite (`task test`, `task lint`, `task e2e`, `task build`) on the milestone branch before the final commit and push, then open the PR to `main`. Work through the milestone gate under each table, in order.
- Mark a task `[~]` when you start it and `[x]` when its tests pass and it is committed (with a `Refs: <task-id>` footer) and pushed to the milestone branch.
- Requirement IDs (`REQ-…`) refer to [PRD.md](PRD.md).
- Record progress in [progress.md](progress.md) and user-visible changes in [CHANGELOG.md](../CHANGELOG.md) in the same commit as the task.
- Full git and test rules: [engineering.md](engineering.md) §8.

Legend: `[ ]` not started · `[~]` in progress · `[x]` done · `[!]` blocked

## M1: Core, native API, SMTP, inbox → v0.1.0

**Branch:** `milestone/m1`. Commit and push every M1 task here.

| | ID | Task | Depends on | REQ |
|---|---|---|---|---|
| [x] | M1-01 | Repo scaffolding: `go.mod`, `Taskfile.yml`, `.golangci.yml` (with `depguard` rules from architecture §3), `biome.json`, `.gitattributes`, `.editorconfig`, Apache-2.0 `LICENSE`, README stub, CI workflow (lint + test matrix) | Open Question 1 (module path) | — |
| [x] | M1-02 | `phone`: E.164 parsing with `valid` / `possible` / `off` modes; GSM-7 vs UCS-2 detection; segment counting (160/153, 70/67) | M1-01 | REQ-063 |
| [x] | M1-03 | `extract`: OTP codes (4–8 digits, keyword-adjacent first), links from text and HTML, `primary_link` | M1-01 | REQ-006 |
| [x] | M1-04 | `core` domain types, canonical errors, prefixed ULIDs, ports (`Store`, `BlobStore`, `Bus`, `Simulator`, `Clock`, `ProjectResolver`), fake clock for tests | M1-01 | REQ-013 |
| [x] | M1-05 | `store/storetest` conformance suite | M1-04 | — |
| [x] | M1-06 | `store/sqlite`: WAL, single writer + read pool, goose migrations, blobs, transactional batch insert; passes `storetest` | M1-05 | REQ-080 |
| [x] | M1-07 | `bus`: in-process pub/sub implementing `core.Bus` | M1-04 | — |
| [ ] | M1-08 | Core service: projects and credentials (auto-create), `SendMessage`, `SendBatch`, validation, extraction on save | M1-02, M1-03, M1-06, M1-07 | REQ-001, REQ-010, REQ-091 |
| [ ] | M1-09 | Lifecycle runner: Clock-driven timers, configurable step delay, resume on startup | M1-08 | REQ-050 |
| [ ] | M1-10 | Verification flow: start, check, attempts, expiry, `otp.fixed_code` | M1-08 | REQ-020, REQ-021, REQ-022 |
| [ ] | M1-11 | `config`: koanf with flags > env > YAML > defaults; all settings from spec §9 | M1-01 | REQ-092 |
| [ ] | M1-12 | `openapi/openapi.yaml` for M1 endpoints; oapi-codegen setup; `api` handlers for send, read, verifications, projects, `/healthz` | M1-08, M1-10 | REQ-010–REQ-014 |
| [ ] | M1-13 | SSE hub and `GET /api/v1/events` | M1-07, M1-12 | REQ-002 |
| [ ] | M1-14 | Security middleware: `Host` allow-list (421), `X-Mocksms` header, no CORS, optional `ui_auth`, non-loopback warning | M1-12 | — |
| [ ] | M1-15 | `smtpd`: listener, AUTH username → project, enmime parsing, 25 MB limit, optional STARTTLS | M1-08 | REQ-035 |
| [ ] | M1-16 | Retention prune job with cascading deletes | M1-06 | REQ-093 |
| [ ] | M1-17 | Web app scaffold: Vite, Tailwind, shadcn/ui, TanStack Router + Query, openapi-typescript/openapi-fetch, `web/embed.go`, Vite proxy for development; design tokens, fonts, Phosphor icons and themed browser surfaces from [design.md](design.md) §5–§9 and §15; record the direction contract (design.md §17) with `impeccable surface-brief write` | M1-12 | — |
| [ ] | M1-18 | Inbox UI per [design.md](design.md) §10–§11: sidebar, top bar with Sandbox pill, project switcher, message list, code tiles, delivery track, SMS threads, email plate (sandboxed iframe, remote-image toggle), filters, get-started checklist, settings page, live updates; finish with `impeccable detect`, the impeccable finish review, and the impeccable documenter writing root `DESIGN.md` + `.impeccable/design.json` | M1-13, M1-17 | REQ-002–REQ-007 |
| [ ] | M1-19 | `cmd/mocksms serve`: wiring, startup/shutdown order (architecture §7), banner | M1-09–M1-16 | — |
| [ ] | M1-20 | Release pipeline: goreleaser (binaries, Docker, Homebrew, Scoop), cosign, release workflow | M1-19 | — |
| [ ] | M1-21 | Docs: README quick start; SMTP setup for Laravel, Django, Rails, Nodemailer, Spring | M1-19 | — |

**M1 milestone gate** (in order, once every task above is `[x]`):
- [ ] Full suite passes on `milestone/m1`: `task test`, `task lint`, `task e2e`, `task build`
- [ ] Final commit (changelog `v0.1.0` heading, progress update) pushed to `milestone/m1`
- [ ] PR `milestone/m1` → `main` merged with a merge commit
- [ ] `v0.1.0` tagged on `main`; next milestone branch created from `main`

## M2: Twilio, Termii, test API, inspector → v0.2.0

**Branch:** `milestone/m2`. Commit and push every M2 task here.

| | ID | Task | Depends on | REQ |
|---|---|---|---|---|
| [ ] | M2-01 | `adapterkit`: panic recovery via `WriteError`, credential → project resolution, request logging with masking (64 KB cap) | M1 | REQ-090 |
| [ ] | M2-02 | RequestLog storage and `GET /api/v1/requests[/{id}]` | M2-01 | REQ-090 |
| [ ] | M2-03 | Twilio Messages API: create, list (filters, paging), fetch; Twilio response shape and error mapping; stores `StatusCallback` | M2-01 | REQ-030, REQ-032 |
| [ ] | M2-04 | Twilio Verify v2: services, verifications, checks, cancel/approve | M2-03 | REQ-031 |
| [ ] | M2-05 | Twilio golden fixtures and contract tests against a pinned twilio-oai spec; scheduled spec-refresh workflow | M2-04 | REQ-032 |
| [ ] | M2-06 | Twilio redirect snippets (Node, Python, PHP, Go, C#) in `examples/`; CI runs them against the built binary | M2-04 | REQ-030 |
| [ ] | M2-07 | Termii SMS: `/api/sms/send`, `/send/bulk`, `/number/send`; batches; sender allow-list | M2-01 | REQ-033 |
| [ ] | M2-08 | Termii Token: `otp/send`, `otp/verify`, `otp/generate`, `email/otp/send` | M2-07 | REQ-034 |
| [ ] | M2-09 | Termii golden fixtures and `docs/fidelity.md` listing unverified behavior | M2-08 | — |
| [ ] | M2-10 | Test API: `messages/wait`, `otp/latest`, `emails/latest`, `verifications/{id}/expire`, scoped `DELETE /messages` | M1 | REQ-040–REQ-043, REQ-023 |
| [ ] | M2-11 | Dedicated adapter ports (`adapters.<name>.port`) | M2-03 | REQ-036 |
| [ ] | M2-12 | Credential linking: YAML `projects[].credentials`, `POST /projects/{id}/credentials`, UI action | M1 | REQ-091 |
| [ ] | M2-13 | UI: OTPs view, request inspector | M2-02, M2-10 | REQ-090 |
| [ ] | M2-14 | Docs: Twilio and Termii guides, test-API guide with Playwright/Cypress/Jest examples | M2-06, M2-10 | — |

**M2 milestone gate** (in order, once every task above is `[x]`):
- [ ] Full suite passes on `milestone/m2`: `task test`, `task lint`, `task e2e`, `task build`
- [ ] Final commit (changelog `v0.2.0` heading, progress update) pushed to `milestone/m2`
- [ ] PR `milestone/m2` → `main` merged with a merge commit
- [ ] `v0.2.0` tagged on `main`; next milestone branch created from `main`

## M3: Webhooks, failure simulation, inbound, batches → v0.3.0

**Branch:** `milestone/m3`. Commit and push every M3 task here.

| | ID | Task | Depends on | REQ |
|---|---|---|---|---|
| [ ] | M3-01 | `webhooks` worker: persistent queue, bus wake-up, 10 s timeout, backoff (1s, 5s, 30s, 2m, 10m), attempt records | M2 | REQ-053 |
| [ ] | M3-02 | Twilio `StatusNotifier` with `X-Twilio-Signature` (auth-token rules from spec §7.4) | M3-01 | REQ-051, REQ-052 |
| [ ] | M3-03 | Termii delivery-report notifier (account-level URL) | M3-01 | REQ-051 |
| [ ] | M3-04 | Native webhooks: JSON payloads, `X-Mocksms-Signature` HMAC-SHA256 (architecture §5.6) | M3-01 | REQ-051, REQ-052 |
| [ ] | M3-05 | Docker `localhost` → `host.docker.internal` rewrite; webhook replay endpoint | M3-01 | REQ-054, REQ-055 |
| [ ] | M3-06 | `sim` engine: rule matching, effects (reject, fail_async, delay, hang, rate_limit), built-in magic values, YAML rules | M2 | REQ-060, REQ-061 |
| [ ] | M3-07 | Async failures with provider delivery error codes (Twilio 30003/30005/30007; email `bounced`) | M3-06 | REQ-060 |
| [ ] | M3-08 | Inbound: `POST /api/v1/inbound`, `core.ReceiveInbound`, Twilio `InboundNotifier`, TwiML `ReplyParser` | M3-01 | REQ-070, REQ-071 |
| [ ] | M3-09 | STOP/START keyword handling and `unsubscribed` errors | M3-08 | REQ-072 |
| [ ] | M3-10 | UI: Batches view, reply box, webhook log tab with replay, sim settings (latency, failure %) | M3-01–M3-09 | REQ-062, REQ-081 |
| [ ] | M3-11 | Playwright smoke tests: live arrival, reply → webhook, inspector, batch progress | M3-10 | — |
| [ ] | M3-12 | Docs: webhooks guide, magic-values table, sim rule reference | M3-06 | — |

**M3 milestone gate** (in order, once every task above is `[x]`):
- [ ] Full suite passes on `milestone/m3`: `task test`, `task lint`, `task e2e`, `task build`
- [ ] Final commit (changelog `v0.3.0` heading, progress update) pushed to `milestone/m3`
- [ ] PR `milestone/m3` → `main` merged with a merge commit
- [ ] `v0.3.0` tagged on `main`; next milestone branch created from `main`

## M4: Estimate, MCP, CI kit → v0.4.0

**Branch:** `milestone/m4`. Commit and push every M4 task here.

| | ID | Task | Depends on | REQ |
|---|---|---|---|---|
| [ ] | M4-01 | Pricing tables (`estimate/pricing/twilio.yaml`, `termii.yaml`) with `as_of`; `estimate` package; `GET /projects/{id}/estimate` | M3 | REQ-100 |
| [ ] | M4-02 | Estimate UI with volume inputs | M4-01 | REQ-100 |
| [ ] | M4-03 | `mcpserver` at `/mcp` with the seven tools | M3 | REQ-101 |
| [ ] | M4-04 | CLI subcommands: `otp latest`, `messages wait`, `reset` | M3 | REQ-102 |
| [ ] | M4-05 | `setup-mocksms` GitHub Action (separate repository) | M1-20 | REQ-103 |
| [ ] | M4-06 | Testcontainers modules: Node, Go, Python (separate packages) with `waitForOtp` | M1-20, M2-10 | REQ-104 |
| [ ] | M4-07 | Bulk benchmark (10,000 recipients) tracked in CI | M3 | — |
| [ ] | M4-08 | Docs: MCP setup, CI guide, estimate methodology | M4-03–M4-06 | — |

M4-05 and M4-06 live in separate repositories; use a `milestone/m4` branch and the same test gate there too.

**M4 milestone gate** (in order, once every task above is `[x]`):
- [ ] Full suite passes on `milestone/m4`: `task test`, `task lint`, `task e2e`, `task build`
- [ ] Final commit (changelog `v0.4.0` heading, progress update) pushed to `milestone/m4`
- [ ] PR `milestone/m4` → `main` merged with a merge commit
- [ ] `v0.4.0` tagged on `main`

## Cross-cutting

Commit and push these on the branch of the milestone that is active when the work is done.

| | ID | Task | Blocked by |
|---|---|---|---|
| [!] | X-01 | Verify Termii's undocumented behavior against a real account; update `docs/fidelity.md` | PRD Open Question 2 (Termii account) |
| [ ] | X-02 | Legal check on provider names in docs and marketing | PRD Open Question 5 |
| [ ] | X-03 | Usability session with 5 developers at the end of each milestone | Milestone release |

## Later waves (epics, not broken down yet)

- **Wave 2:** SendGrid, Resend, Vonage (SMS + Verify), AWS SNS adapters; email cost estimates.
- **Wave 3:** Africa's Talking, Hubtel, Arkesel, mNotify adapters.
- **Backlog:** `npx mocksms` wrapper, stdio MCP bridge, WhatsApp and voice channels, hosted product.
