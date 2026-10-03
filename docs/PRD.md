# mocksms — Product Requirements Document

## 1. Document Header

| Field | Value |
|---|---|
| Product | mocksms (working name) |
| Author | [Author Name] |
| Date | 2026-10-03 |
| Version | v1.0 |
| Status | In Review |
| Stakeholders | Project owner (product and engineering); [additional stakeholders] |
| Related | [Design spec](superpowers/specs/2026-10-03-mocksms-design.md) · [Architecture](architecture.md) · [Tasks](tasks.md) |

## 2. Executive Summary

mocksms is a free sandbox that stands in for SMS, one-time-password and email providers while software is being built and tested. Developers point their apps at it instead of Twilio, Termii or SendGrid. Every message appears instantly in a web inbox and can be read by automated tests, but nothing is actually delivered and nothing costs money. When the product is ready to launch, the team switches to the real provider with a configuration change and starts paying only then.

## 3. Problem Statement

**The pain.** Almost every product that has user accounts sends verification codes and emails. During development, teams currently pick one of four imperfect options:

| Current option | Why it hurts |
|---|---|
| Use a real provider account | Every test message costs money; sender IDs often need registration first; test messages reach real phones |
| Send to their own phone and read it manually | Slow, impossible to automate, breaks CI |
| Mock the provider inside their code | Mocks drift from real API behavior; webhooks, signatures and error paths go untested |
| Use provider test credentials | Limited: Twilio's test credentials do not cover Verify, and there is no inbox to see what was sent |

**Who experiences it.** Backend and full-stack developers, QA engineers and CI pipelines, on every run of a signup, login, 2FA or notification flow: many times per developer per day.

**Severity.** Medium to high: real money in markets where SMS is expensive relative to developer budgets; flaky or missing end-to-end tests; and launch-day bugs in code paths that were never exercised (delivery failures, unsubscribes, webhook signature checks).

**Cost of inaction.** Teams keep paying for test traffic, skip automated tests of their most critical flow (account creation), and find provider-integration bugs in production.

**Job to be done.** *When I build a flow that sends a code or a verification email, I want to see and use that message instantly, without paying or picking up my phone, so I can build and test the whole flow end to end, including the failure cases.*

## 4. Goals & Success Metrics

### Goals

1. Make developing and testing messaging flows free and instant.
2. Make switching to a real provider a configuration change, not a rewrite.
3. Make automated tests of OTP and email-verification flows reliable in CI.
4. Build a developer user base that a future hosted team product can grow from.

### KPIs

mocksms collects no telemetry, so every KPI uses public or test-based signals.

| KPI | Target | Measured by |
|---|---|---|
| Time from download to first captured message | ≤ 5 minutes `[ASSUMED]` | Moderated usability sessions with 5 developers per milestone |
| Real-SDK compatibility suite | 100% passing on every release | CI (`examples/` against official SDKs) |
| GitHub stars | 1,000 within 6 months of the M2 release `[ASSUMED]` | GitHub |
| Downloads | 10,000 combined release downloads and Docker pulls within 6 months of M2 `[ASSUMED]` | GitHub Releases, GHCR |
| Fidelity bug turnaround | Median ≤ 14 days from report to release `[ASSUMED]` | GitHub issues labelled `fidelity` |

### Anti-goals

- Delivering any real message, or acting as a production gateway.
- Collecting telemetry or usage data.
- Optimizing for hosted, multi-tenant scale in Wave 1.
- Copying every endpoint of each provider. Only the messaging and verification surfaces developers actually use are faked.

## 5. User Personas & Use Cases

### Persona 1: Ama, backend developer at a fintech startup

- **Context:** Builds phone-number signup and transaction OTPs with Termii and sends receipts by SMTP. Works on a laptop with a limited data plan; the company is careful about spend.
- **Motivation:** Stop paying for test SMS and stop waiting for messages to reach a phone.

*Stories:*
- As Ama, I want my app's Termii calls to land in a local inbox so that I can develop without spending SMS credit.
- As Ama, I want to test what happens when an SMS fails to deliver so that my app shows a sensible retry message.
- As Ama, I want to see the exact request my app sent and the response it got so that I can fix integration bugs without guessing.

### Persona 2: Daniel, QA automation engineer

- **Context:** Owns the Playwright end-to-end suite and the CI pipeline. Signup tests are currently skipped because they need a real OTP.
- **Motivation:** Automate the signup and password-reset flows reliably and in parallel.

*Stories:*
- As Daniel, I want to wait for the next OTP sent to a number so that my test can type it in without sleeping.
- As Daniel, I want to extract the verification link from an email so that my test can click it.
- As Daniel, I want to start the sandbox in CI with one step so that every pipeline run has a clean inbox.

### Persona 3: Priya, freelance developer serving several clients

- **Context:** Uses Twilio Verify for one client and SMTP plus Twilio SMS for another. Works with an AI coding agent most of the day.
- **Motivation:** Keep client projects separate, keep production switch-over painless, and let the agent test flows by itself.

*Stories:*
- As Priya, I want each client's messages grouped separately so that I don't confuse their data.
- As Priya, I want my coding agent to read OTPs through MCP so that it can finish a signup flow on its own.
- As Priya, I want an estimate of monthly messaging costs so that I can quote the client before launch.

## 6. Functional Requirements

Priority labels: `[P0 — Must Have]` for the core promise, `[P1 — Should Have]` for important workflow coverage, `[P2 — Nice to Have]` for differentiators that can slip without breaking the core. The milestone column shows when each is built; the owner chose to include all of these in Wave 1.

### 6.1 Capture and inbox

| ID | Requirement | Priority | Milestone |
|---|---|---|---|
| REQ-001 | The system stores every outbound SMS and email it accepts, with full content, and never delivers it to a real recipient. | P0 | M1 |
| REQ-002 | The web inbox shows new messages within 1 second of acceptance without a page reload. | P0 | M1 |
| REQ-003 | SMS messages are grouped into one conversation per recipient number, showing status for each message. | P0 | M1 |
| REQ-004 | Email messages show HTML, plain text, raw source, headers and attachments. | P0 | M1 |
| REQ-005 | Email HTML renders in a sandbox that cannot run scripts; remote images load only after the user allows them for that message. | P0 | M1 |
| REQ-006 | Detected OTP codes and verification links are highlighted with one-click copy. | P1 | M1 |
| REQ-007 | Users can filter the inbox by project, channel, recipient and text. | P1 | M1 |

### 6.2 Native API

| ID | Requirement | Priority | Milestone |
|---|---|---|---|
| REQ-010 | `POST /api/v1/sms` accepts one or many recipients; many recipients create a batch. | P0 | M1 |
| REQ-011 | `POST /api/v1/email` accepts to/cc/bcc, subject, text, HTML and attachments. | P0 | M1 |
| REQ-012 | Messages can be listed with filters and cursor pagination, and fetched individually with their status timeline. | P0 | M1 |
| REQ-013 | Errors use a single canonical error format with a stable code, message and optional field. | P0 | M1 |
| REQ-014 | The API is described by a published OpenAPI document. | P1 | M1 |

### 6.3 Verification (OTP)

| ID | Requirement | Priority | Milestone |
|---|---|---|---|
| REQ-020 | Starting a verification generates a code, sends it as a normal message and returns a verification ID without the code. | P0 | M1 |
| REQ-021 | Checking a verification approves it on the right code, counts attempts, and enforces maximum attempts and expiry. | P0 | M1 |
| REQ-022 | A fixed OTP code can be configured for deterministic tests. | P1 | M1 |
| REQ-023 | A verification can be force-expired through the API for testing expiry paths. | P1 | M2 |

### 6.4 Provider adapters

| ID | Requirement | Priority | Milestone |
|---|---|---|---|
| REQ-030 | The official Twilio SDKs (Node, Python, PHP, Go, C#) can send SMS and list/fetch messages against mocksms using a documented HTTP-client snippet. | P0 | M2 |
| REQ-031 | The official Twilio SDKs can create, check, cancel and approve Verify v2 verifications against mocksms. | P0 | M2 |
| REQ-032 | Twilio responses match Twilio's field names, status values, ID formats, date formats and error codes. | P0 | M2 |
| REQ-033 | Termii SMS, number-SMS and bulk endpoints (up to 10,000 recipients) work with only a base-URL change. | P0 | M2 |
| REQ-034 | Termii Token endpoints (send, verify, in-app generate, email token) work with only a base-URL change. | P0 | M2 |
| REQ-035 | Any SMTP client can send email to mocksms; the SMTP username selects the project. | P0 | M1 |
| REQ-036 | Each adapter can also be served on its own port at `/` for SDKs that accept only a hostname. | P1 | M2 |

### 6.5 Test-assertion API

| ID | Requirement | Priority | Milestone |
|---|---|---|---|
| REQ-040 | `GET /api/v1/messages/wait` long-polls for the first matching message after a given time, with a configurable timeout (max 60 s). | P0 | M2 |
| REQ-041 | `GET /api/v1/otp/latest` returns the newest OTP for a recipient, from either a verification or a code found in a plain message. | P0 | M2 |
| REQ-042 | `GET /api/v1/emails/latest` returns the newest email with extracted codes, links and the most likely verification link. | P0 | M2 |
| REQ-043 | A project's data can be reset through the API for test isolation. | P1 | M2 |

### 6.6 Delivery lifecycle and webhooks

| ID | Requirement | Priority | Milestone |
|---|---|---|---|
| REQ-050 | Every accepted message moves through `queued → sent → delivered` (or a failure state) with a configurable step delay, including zero. | P0 | M1 |
| REQ-051 | Status changes are sent to the developer's callback URL in the originating provider's format. | P1 | M3 |
| REQ-052 | Webhooks are signed the way the real provider signs them, using the developer's credential. | P1 | M3 |
| REQ-053 | Failed webhook deliveries are retried with backoff (5 attempts by default) and every attempt is recorded. | P1 | M3 |
| REQ-054 | Any recorded webhook can be replayed from the UI or API. | P2 | M3 |
| REQ-055 | Webhooks to `localhost` work when mocksms runs in Docker. | P1 | M3 |

### 6.7 Failure simulation

| ID | Requirement | Priority | Milestone |
|---|---|---|---|
| REQ-060 | Documented magic phone numbers and email domains trigger specific provider-accurate errors, immediately or after acceptance. | P1 | M3 |
| REQ-061 | Custom rules can match on recipient, sender, provider, project or channel and apply reject, async failure, delay, hang or rate-limit effects. | P1 | M3 |
| REQ-062 | Global latency and random-failure percentage can be set from the UI. | P2 | M3 |
| REQ-063 | Invalid phone numbers are rejected like a real provider would, with a setting to loosen validation for made-up numbers. | P0 | M1 |

### 6.8 Inbound and two-way SMS

| ID | Requirement | Priority | Milestone |
|---|---|---|---|
| REQ-070 | A user can reply as the phone from the inbox; the reply is sent to the developer's inbound webhook in provider format. | P1 | M3 |
| REQ-071 | TwiML `<Message>` replies from the developer's inbound handler appear as outbound replies in the thread. | P2 | M3 |
| REQ-072 | STOP-family keywords unsubscribe a number and later sends fail with the provider's unsubscribed error; START-family keywords resubscribe. | P1 | M3 |

### 6.9 Bulk

| ID | Requirement | Priority | Milestone |
|---|---|---|---|
| REQ-080 | A 10,000-recipient send is accepted as a single batch with live per-status counts. | P0 | M1 (core), M2 (Termii bulk) |
| REQ-081 | Batches appear as one collapsible row in the inbox, with a progress view. | P1 | M3 |

### 6.10 Inspector, projects and settings

| ID | Requirement | Priority | Milestone |
|---|---|---|---|
| REQ-090 | Every adapter request and response is recorded and viewable raw, with credentials masked. | P1 | M2 |
| REQ-091 | Projects are created automatically from credentials; several credentials can be linked to one project. | P0 | M1 (auto), M2 (linking) |
| REQ-092 | Settings come from flags, environment variables and a YAML file; YAML-defined settings appear read-only in the UI. | P0 | M1 |
| REQ-093 | Old data is pruned automatically (default: 10,000 messages per project or 7 days). | P1 | M1 |

### 6.11 Go-live estimate, MCP and CI kit

| ID | Requirement | Priority | Milestone |
|---|---|---|---|
| REQ-100 | The estimate combines observed traffic mix with developer-entered volume to show monthly cost per provider and country, with the pricing date shown. | P2 | M4 |
| REQ-101 | An MCP server at `/mcp` exposes list, wait, latest-OTP, latest-email, simulate-inbound, list-requests and reset tools. | P2 | M4 |
| REQ-102 | CLI subcommands read the latest OTP, wait for messages and reset projects on a running server. | P2 | M4 |
| REQ-103 | A GitHub Action starts mocksms in CI and outputs its URLs. | P2 | M4 |
| REQ-104 | Testcontainers modules for Node, Go and Python start mocksms and offer a `waitForOtp` helper. | P2 | M4 |

## 7. Non-Functional Requirements

| Area | Requirement |
|---|---|
| Performance | Cold start ≤ 1 s; idle memory ≤ 60 MB; p95 send latency ≤ 20 ms (local, excluding simulated delays); 10,000-recipient batch accepted ≤ 3 s; UI update ≤ 1 s after an event `[ASSUMED targets]` |
| Size | Release binary ≤ 40 MB; Docker image ≤ 30 MB `[ASSUMED]` |
| Reliability | No accepted message is lost across a graceful restart in persistent mode; in-flight lifecycles resume on startup |
| Security | Listens on loopback by default; rejects requests with unexpected `Host` headers (DNS-rebinding protection); no CORS by default; sandboxed email HTML; credentials masked in logs and inspector; optional UI basic auth when exposed |
| Privacy | No telemetry; no outbound traffic except developer-configured webhooks; all data stays in the local data directory |
| Scalability | Single developer or CI job; at least 100,000 stored messages without UI slowdown (paginated) |
| Accessibility | Web UI meets WCAG 2.2 AA; fully keyboard-navigable |
| Platforms | Windows 10+, macOS 13+, Linux; amd64 and arm64; Docker multi-arch |
| Browsers | Latest two versions of Chrome, Edge, Firefox and Safari |
| Licensing | Local tool Apache-2.0 |

## 8. User Experience & Design Direction

### Key flows

**Flow 1: First message (M1)**
1. Developer downloads and runs `mocksms`.
2. Terminal banner shows the inbox URL and SMTP address.
3. Developer sets their app's SMTP host to `localhost:1025` and triggers a signup email.
4. The email appears in the inbox; the verification link is highlighted.

**Flow 2: Twilio Verify in an app (M2)**
1. Developer copies the Twilio HTTP-client snippet for their language from the docs.
2. App calls `verifications.create`; mocksms returns a `VE…` SID.
3. The OTP appears in the SMS thread with a copy button.
4. Developer enters the code in their app; `verificationChecks.create` returns `approved`.

**Flow 3: Automated signup test in CI (M2 + M4)**
1. CI starts mocksms with the GitHub Action.
2. The test records `since`, then submits the signup form.
3. The test calls `GET /api/v1/messages/wait?to=…&since=…` (or `waitForOtp`) and receives the code.
4. The test enters the code and asserts the account was created.

**Flow 4: Testing a delivery failure (M3)**
1. Developer sends to a documented magic number.
2. The API accepts the message; the status moves to `undelivered` with a provider error code.
3. The developer's status webhook receives the failure, correctly signed.
4. The inspector shows the request, the status timeline and the webhook attempt.

**Flow 5: Two-way SMS (M3)**
1. Developer opens a conversation and types `STOP` in the reply box.
2. The developer's inbound webhook receives it in provider format.
3. The next send to that number fails with the provider's unsubscribed error.

### UX principles

- **Zero setup:** works with any credentials, no sign-up, sensible defaults.
- **Show the raw truth:** raw requests, responses, MIME and webhook payloads are always one click away.
- **Copy first:** codes, links, IDs and snippets have copy buttons.
- **Live by default:** nothing requires a refresh.
- **Never hide errors:** warnings (unsigned webhooks, unregistered sender IDs, exposed host) are visible in the UI.

### Design constraints

- Developer-tool aesthetic: dense, readable, light and dark themes.
- Three-pane layout (navigation, list, detail), described in the design spec §8.1.
- Built with Tailwind and shadcn/ui; no custom brand guidelines yet.

### Design handoff notes

- Highest-value screens to design first: SMS thread, email detail, request inspector.
- Status ticks must be distinguishable without color (icon and label).

## 9. Technical Considerations

- **Architecture:** single Go binary with an embedded React UI; modular monolith. See [architecture.md](architecture.md).
- **Integrations:** fake surfaces for Twilio (Messages, Verify v2), Termii (SMS, Token) and SMTP in Wave 1. No calls to real providers.
- **Data model:** projects, credentials, messages, status events, batches, verifications, attachments, webhook deliveries, request logs (spec §6).
- **Compatibility:** Twilio SDKs need a custom HTTP client to redirect; this is the biggest integration-friction risk and is covered by CI tests per language.
- **Backward compatibility:** pre-1.0 releases may change the native API; changes are recorded in the changelog. Provider-compatible surfaces follow the providers, not us.
- **Early investigation items:**
  1. Confirm every Twilio SDK's custom HTTP-client hook supports full URL rewriting (Node, Python, PHP, Go, C#).
  2. Gather Termii's undocumented response bodies (verify failures, errors, delivery reports, webhook signature, inbound support).
  3. Measure SQLite single-writer throughput for 10,000-message batches on Windows.

## 10. Out of Scope (for this version)

- Hosted SaaS: accounts, teams, shared inboxes, billing, Postgres deployment.
- Real delivery of any message.
- WhatsApp, voice and RCS channels.
- SendGrid, Resend, Vonage and AWS SNS adapters (Wave 2).
- Africa's Talking, Hubtel, Arkesel and mNotify adapters (Wave 3).
- MMS / media messages.
- npm wrapper (`npx mocksms`) and stdio MCP bridge.

## 11. Open Questions

1. What is the final product name and GitHub organization (sets the Go module path)? `[Owner: Project owner]` `[Due: before M1-01]`
2. Who provides a real Termii account to verify undocumented behavior, and when? `[Owner: Project owner]` `[Due: before M2 is marked stable]`
3. Should the Twilio redirect snippets also ship as tiny published packages (e.g. one per language)? `[Owner: Project owner]` `[Due: end of M2]`
4. Where does pricing data for the go-live estimate come from, and how often is it refreshed? `[Owner: Project owner]` `[Due: start of M4]`
5. Is using provider names in docs and marketing ("Twilio-compatible") acceptable as nominative use? Get a legal check before public launch. `[Owner: Project owner]` `[Due: before M2 public announcement]`
6. When should work on the hosted product start, and what would trigger it (stars, downloads, waitlist size)? `[Owner: Project owner]` `[Due: TBD]`

## 12. Dependencies & Risks

### Dependencies

- Provider public documentation and OpenAPI specs (Twilio; later SendGrid, Vonage).
- Official provider SDKs, used in CI compatibility tests.
- Open-source Go and TypeScript libraries listed in [techstack.md](techstack.md).
- A Termii account for fidelity verification.

### Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Adapters drift as providers change their APIs | Medium | High | Pinned OpenAPI specs with a scheduled refresh PR; real-SDK tests in CI |
| Termii behavior differs from its docs | High | Medium | Mark unverified items in `docs/fidelity.md`; verify with a real account before "stable" |
| Twilio redirect friction stops adoption | Medium | High | Tested snippets for five languages; consider published helper packages (Open Question 3) |
| Wave 1 scope delays the first release | High | Medium | Four milestones, each a usable release; public announcement at M2 |
| Email-only tools (Mailpit, Mailtrap) make the email side unremarkable | High | Low | Position on SMS/OTP and African providers; email is table stakes |
| Pricing data in the estimate goes stale | Medium | Low | Show `as_of` date and "estimate only" label; update each release |
| Trademark concerns over provider names | Low | Medium | Nominative "compatible with" wording; legal check (Open Question 5) |

## 13. Timeline & Milestones

| Phase | Scope | Dates |
|---|---|---|
| Discovery | Problem, personas, competitive landscape | Done 2026-10-03 |
| Design | Approved design spec | Done 2026-10-03 |
| Engineering M1 → release v0.1.0 | Core, native API, SMTP, inbox | [TBD] |
| Engineering M2 → release v0.2.0 | Twilio, Termii, test API, inspector; public announcement `[ASSUMED]` | [TBD] |
| Engineering M3 → release v0.3.0 | Webhooks, failure simulation, inbound, batches | [TBD] |
| Engineering M4 → release v0.4.0 | Estimate, MCP, CI kit | [TBD] |
| QA | Per milestone: CI suites plus 5-developer usability session | [TBD] |

## 14. Appendix

### Competitive landscape

| Tool | What it covers | Gap mocksms fills |
|---|---|---|
| Mailpit, MailHog, smtp4dev | Local email capture over SMTP | No SMS, OTP or provider APIs |
| Mailtrap | Hosted email sandbox with paid tiers | Email-focused; SMS/OTP sandboxing not its core |
| Twilio test credentials | Magic numbers for Messages | No Verify, no inbox, Twilio only |
| Africa's Talking sandbox | Account-bound simulator | One provider, tied to an account |
| LocalStack | AWS emulation, including SNS | AWS only; not focused on messaging workflows |
| Hand-written mocks | Anything | Drift from real behavior; no webhooks or signatures |

### Related documents

- [Design spec](superpowers/specs/2026-10-03-mocksms-design.md)
- [Overview](overview.md) · [Architecture](architecture.md) · [Tech stack](techstack.md) · [Engineering](engineering.md) · [Tasks](tasks.md)

---
## PRISM's Notes

**Assumptions made:**
- KPI targets (stars, downloads, 5-minute time to first message, 14-day fidelity turnaround) are starting targets, not commitments.
- Performance and size budgets in §7 are initial targets to be confirmed by the M1 benchmark.
- The public announcement happens at M2, when the SMS/OTP differentiator exists.
- Personas are composites drawn from the brainstorming discussion, not interviewed users.

**Gaps filled:**
- Defined KPIs that work without telemetry (the design commits to no telemetry).
- Added accessibility (WCAG 2.2 AA) and browser-support targets, which the design spec did not state.
- Added the trademark/nominative-use question and a competitive landscape.
- Mapped every requirement to a priority and a milestone.

**Top 3 open questions to resolve first:**
1. Final name and GitHub organization — blocks M1-01 repo setup and the Go module path.
2. Access to a real Termii account — Termii is the Wave 1 differentiator, and several of its behaviors are undocumented.
3. Whether Twilio redirect snippets become published packages — the biggest adoption-friction point for the most widely used provider.
