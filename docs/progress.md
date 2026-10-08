# mocksms — Progress

Single source of truth for where the project stands. Update it in every PR that changes task status (see [AGENTS.md](../AGENTS.md)).

## Current status

| | |
|---|---|
| **Phase** | Milestone 2 in progress |
| **Current milestone** | M2 |
| **Next action** | M2-09: Termii golden fixtures and `docs/fidelity.md` unverified listing |
| **Last updated** | 2026-10-08 |

## Milestones

| Milestone | Release | Status | Tasks done |
|---|---|---|---|
| M1: Core, native API, SMTP, inbox | v0.1.0 | Done | 21 / 21 |
| M2: Twilio, Termii, test API, inspector | v0.2.0 | In progress | 8 / 14 |
| M3: Webhooks, failure simulation, inbound, batches | v0.3.0 | Not started | 0 / 12 |
| M4: Estimate, MCP, CI kit | v0.4.0 | Not started | 0 / 8 |

## Blockers and open items

| Item | Blocks | Owner |
|---|---|---|
| Access to a real Termii account | X-01; Termii marked stable | Project owner |
| Legal check on provider names | Public announcement at M2 | Project owner |

**Note:** M1-06, M1-09, M1-12 and M1-13 have open fixes tracked in the M1-Fxx review tasks above.

## Session log

Newest first. One entry per working session: what changed, decisions made, what's next.

### 2026-10-08 (Session 14): Termii Token (M2-08)

- Completed M2-08 (`REQ-034`): four Token endpoints on the existing `/termii` mount (no wiring changes).
  - **otp/send**: `pin_length` (default 6), `pin_attempts`, `pin_time_to_live` (minutes→seconds) map onto the verification; the PIN is minted adapter-side so `message_text` + `pin_placeholder` templates render in one write; UUIDv4 `pinId` in `provider_ref`; responds `{pinId, to, smsStatus}`.
  - **otp/verify** (`pin_id` + `pin`): wrong PIN → 200 `verified: false`, correct → `verified: true` with `msisdn`, unknown PIN → 404, exhaustion → 429.
  - **otp/generate** (`NUMERIC` only): returns `{pin}`, sends and stores nothing.
  - **email/otp/send**: validates the address (net/mail), stores the caller-supplied code verbatim on an email-channel verification.
  - **Core extension** (same justification as M2-04's `ServiceLabel`): provider-neutral `CustomCode` (explicit codes win over `--otp-code`, 4–10 chars) and `BodyText` (verbatim template override) on `VerificationRequest`. Covered by `TestService_VerificationCustomCodeAndBody` (mock store gained `CreateVerification`).
  - Tests first, all green: `go test ./...`, `golangci-lint run` (0 issues), `gofmt` clean. Live-binary smoke test of all four endpoints.
  - `docs/fidelity.md` gained a Token section; every response shape is marked **unverified** for X-01/M2-09.
- **Next:** M2-09: Termii golden fixtures.

### 2026-10-08 (Session 13): Termii SMS (M2-07)

- Completed M2-07 (`REQ-033`): new `adapters/termii` package, mounted at `/termii` with store-backed logging and required `api_key` credentials.
  - **Endpoints**: `POST /api/sms/send` (single `to` or array up to 100), `POST /api/sms/send/bulk` (up to 10,000, response adds `code: "ok"`), `POST /api/sms/number/send`. HTTP 200 with `{message_id, message, balance, user}`; numeric IDs in `provider_ref`; multi-recipient sends map to core batches with per-recipient `rejected` reporting.
  - **Sender allow-list** from project settings (`termii.sender_allowlist`, read-only until M2-12): empty accepts all with a warning log, unlisted senders get `invalid_sender`.
  - **Real bug found by TDD**: bodies over 64 KB fail credential extraction (truncated JSON never parses), which would have broken every bulk batch. Added `adapterkit.JSONBodyKeyExtractorWithLimit` (default stays 64 KB) and gave Termii a 1 MB cap; covered by a new adapterkit test with a ~90 KB body.
  - `/termii` exempt from `X-Mocksms` (Host protection kept), covered by a middleware test.
  - Tests first, all green: `go test ./...`, `golangci-lint run` (0 issues), `gofmt` clean. Live-binary smoke test (single, bulk with `code: ok`, number/send, 401 without key). Debugging note: Windows PowerShell corrupts inline curl JSON — byte-exact `@file` bodies (via the Write tool) are the reliable path.
  - `docs/fidelity.md` gained a Termii section with **unverified** items for X-01/M2-09.
- **Next:** M2-08: Termii Token.

### 2026-10-08 (Session 12): Twilio Redirect Snippets (M2-06)

- Completed M2-06 (`REQ-030`): `examples/twilio/` holds official-SDK snippets for Node, Python, PHP, Go and C#, each with a custom HTTP client rewriting both `api.twilio.com` and `verify.twilio.com` to `<MOCKSMS_URL>/twilio`. Uniform flow per snippet: create Verify service → send SMS (with dummy `StatusCallback`) → start verification → check with `OTP_CODE`, asserting `SM`/`VA` SIDs and `approved`. Distinct default recipients per language (`+15005550010`…`014`) so runs never collide on To-based checks.
  - SDK hooks used: Node `httpClient` (`RequestClient` subclass), Python `TwilioHttpClient.request` override, PHP `Twilio\Http\Client` wrapper around `CurlClient` (5th `Client` ctor arg), Go `client.Client.SendRequest` override, C# `Twilio.Http.HttpClient` subclass rebuilding the `Request`.
  - Findings while verifying: twilio-go rejects non-alphanumeric tokens client-side (21224), so all snippets default to `testtoken123`; its `SendRequest` takes a variadic body. Credentials stay push-safe (`ACXXX…` placeholder + env overrides).
  - Harness `examples/twilio_snippets_test.go` (`//go:build e2e`) runs each snippet, asserts exit 0 + `E2E-OK` marker. Runtime probing (not bare `LookPath`) skips Windows Store python stubs and SDK-less dotnet instead of failing.
  - Verified locally: Node/Python/Go pass end-to-end against the binary; PHP/C# skip without runtimes and go through CI. CI e2e job now installs Node 20, Python 3.13, PHP 8.3 and .NET 8 toolchains + snippet deps, starts the binary with `--otp-code 123456`, and runs the harness with `OTP_CODE=123456`.
  - `go test ./...`, `golangci-lint run` (0 issues), `gofmt` clean; `go vet -tags e2e` clean.
- **Next:** M2-07: Termii SMS.

### 2026-10-08 (Session 11): Twilio Golden Fixtures and Contract Tests (M2-05)

- Completed M2-05 (`REQ-032`): the fidelity harness from engineering §5.
  - **Pinned specs** (`adapters/twilio/spec/`, 73 KB + 112 KB): trimmed from upstream twilio-oai (retrieved 2026-10-08) to the implemented endpoints plus their component closure, as canonical JSON. New `adapters/twilio/spectrim` tool re-derives them deterministically; operation examples are dropped because upstream ships invalid ones (a non-RFC3339 `date-time` example fails strict spec validation). Sources and regeneration commands in `spec/README.md`.
  - **Golden fixtures** (`adapters/twilio/testdata/`, 17 files): success and error cases for every Messages and Verify endpoint, with volatile fields (SIDs, dates) normalized and fixed-clock timestamps. `TestTwilio_Golden` compares via go-cmp; regenerate with `-update` and review the diff.
  - **Contract tests** (`adapters/twilio/contract_test.go`): live success responses validated with kin-openapi (now a direct dependency) against the pinned specs — 4 Messages cases, 5 Verify cases. Spec operations are resolved by exact template (the bundled routers can't match suffixed segments like `Messages/{Sid}.json`; route matching itself is covered by handler tests).
  - **Findings fixed by the harness**: the suite's account SID was 32 chars, but the spec requires `AC`+32 (34 total) — corrected across tests and goldens; added a `max_attempts`→`max_attempts_reached` mapper for Verify statuses per the spec enum description.
  - **Refresh workflow** (`.github/workflows/spec-refresh.yml`): weekly cron + manual trigger; re-downloads, re-trims, opens a PR on change, then runs the contract tests.
  - `go test ./...`, `golangci-lint run` (0 issues), `gofmt` clean. `docs/fidelity.md` records what the contract confirms vs. what stays unverified (error codes, lenient coercions, `date_sent` semantics).
- **Next:** M2-06: Twilio redirect snippets.

### 2026-10-08 (Session 10): Twilio Verify v2 (M2-04)

- Completed M2-04 (`REQ-031`): Verify v2 endpoints on the existing `/twilio` mount (no wiring changes needed).
  - **Services** (`POST /v2/Services`, `GET /v2/Services/{VA}`): `VA`+32-hex SIDs, `FriendlyName` (default "mocksms"), `CodeLength` (default 6, enforced 4-8). Records persist in project settings under `twilio_verify_services` — no migration, no core store change. Unknown `VA` SIDs auto-provision with defaults on verification create (spec §7.4).
  - **Verifications**: create by `To` + `Channel=sms|email` (201, `VE` SID, `pending`/`valid:false`); fetch; update with `Status=canceled|approved` (non-pending transitions rejected, event published like the native expire path).
  - **VerificationCheck** by `To` (newest pending) or `VerificationSid`: wrong code stays `pending`, correct code approves, exhaustion is 429 code 60202, unknown/expired/canceled is 404 code 20404.
  - **Core extension** (adapter-checklist rule: core was missing something): provider-neutral `ServiceLabel` on `VerificationRequest`, preferred over `ServiceRef` in the message text, so the SMS reads `Your {friendly name} verification code is: {code}` while `ServiceRef` keeps the `VA` reference. Covered by `TestService_VerificationText`; no provider formats in core.
  - Tests written first (`adapters/twilio/verify_test.go`: services, friendly-name text via the stored message, auto-provision, email channel, check flows incl. 5-attempt exhaustion, cancel/approve/get, error codes). `go test ./...`, `golangci-lint run` (0 issues), `gofmt` clean. Live-binary smoke test: service create → verification create → wrong-code check (`pending/false`) → cancel (`canceled`).
  - `docs/fidelity.md` gained a Verify section with **unverified** items for M2-05 contract tests.
- **Next:** M2-05: Twilio golden fixtures and contract tests.

### 2026-10-08 (Session 9): Twilio Messages API (M2-03)

- Completed M2-03 (`REQ-030`, `REQ-032`): new `adapters/twilio` package, mounted at `/twilio` in `cmd/mocksms` with a store-backed `RequestLog` sink and required credentials.
  - **Create** (`POST /2010-04-01/Accounts/{AC}/Messages.json`): form-encoded `To`/`From`/`Body`/`StatusCallback`, `SM`+32-hex SID stored in `provider_ref`, 201 with Twilio's message shape (string `num_segments`/`num_media`, `direction: outbound-api`, `price: null`, RFC 2822 dates, `date_sent: null` while queued).
  - **List** (`GET …/Messages.json`): `To`/`From`/`DateSent` (exact + range operators) filters, `PageSize` (default 50, max 1000) / `Page` paging with Twilio's envelope (`total`, `num_pages`, `first/next/previous_page_uri`).
  - **Fetch** (`GET …/Messages/{SM}.json`): lookup by `provider_ref`, 404 code 20404.
  - **WriteError**: canonical → Twilio codes (21211, 21212, 21610, 21612, 21614, 20003, 20404, 20429, 20500/20503, 60202) with `{code, message, more_info, status}`.
  - **Middleware**: `/twilio` exempt from the `X-Mocksms` header (SDKs can't send it) with Host allow-list still enforced; covered by new `middleware/security_test.go`.
  - Decisions (documented in new `docs/fidelity.md`): URIs omit the `/twilio` prefix; list/fetch show Twilio-provider messages only; `MediaUrl` counted but not stored; in-memory scan cap 5000. Best-guess codes (21604/21606/21602, 20500/20503, …) marked **unverified** for M2-05 contract tests.
  - Tests written failing first (missing package, then a real pagination-total failure that reshaped the list implementation). `go test ./...`, `golangci-lint run` (0 issues), `gofmt` clean. Smoke-tested the built binary end-to-end: create → 201, lifecycle → `delivered` with `date_sent`, list envelope, fetch, 21211 on bad `To`, 20003 without auth.
- **Next:** M2-04: Twilio Verify v2.

### 2026-10-08 (Session 8): RequestLog Inspector API (M2-02)

- Completed M2-02 (`REQ-090`): `GET /api/v1/requests` and `GET /api/v1/requests/{id}` are now spec-complete.
  - **List** honors `limit` (default 50, clamped to 200) and `cursor` from the generated params instead of a hardcoded 50; responses use the generated `RequestLog` type via a new `convertRequestLog` helper (request/response bodies as strings, `duration_ms` as int).
  - **Get** is now project-scoped: a log from another project returns 404 `not_found` instead of leaking across projects (M1-F13 scoping rule).
  - Store layer (`core.Store`, SQLite, memstore, `storetest` case, migration, retention pruner) already covered RequestLog from M1, so no schema or port changes were needed; adapter sink wiring lands with the first adapter in M2-03 via `adapterkit.RequestLogSinkFunc`.
  - Added `TestHandlers_RequestLogs` (list shape, limit+cursor paging over two pages, get by id, 404 on unknown id, 404 on cross-project access). Written failing first: limit ignored and cross-project 200 confirmed before the fix.
  - `go test ./...`, `golangci-lint run` (0 issues) and `biome check` pass. Note: `pnpm test`/`pnpm lint` script spawning hangs in this sandbox (even for the no-op echo script); Go tests, golangci-lint and biome run directly are green.
- **Next:** M2-03: Twilio Messages API.

### 2026-10-08 (Session 7): Milestone 2 Start — Adapterkit Middleware & Request Logging (M2-01)

- Implemented `adapters/adapterkit` package providing shared adapter middleware and coordinators (`REQ-090`):
  - **Adapter Interface**: Defined `Adapter` (`Name`, `Routes`, `WriteError`) contract in `adapters/adapterkit/adapter.go`.
  - **Panic Recovery (`Recoverer`)**: Catches panics in adapter handlers, logs error and stack trace with `slog.Error`, propagates `http.ErrAbortHandler` using `errors.Is`, and translates panics to canonical 500 errors formatted via `adapter.WriteError`.
  - **Credential Resolution (`ResolveProject`)**: Modular extractors for Basic Auth username, Bearer tokens, headers, path params, query params, and JSON body keys (`FirstOf`), auto-resolving projects via `core.ProjectResolver` and safely injecting project and credential identities into request context.
  - **Credential Masking (`MaskHeaders`, `MaskBody`, `MaskURL`)**: Comprehensive redaction of Basic auth passwords, Bearer tokens, cookies, API keys, and sensitive fields across JSON and form-urlencoded bodies, and URI query parameters.
  - **Bounded Request Logging (`RequestLogger`)**: Audits incoming adapter traffic with a strict 64 KB cap on request and response bodies using `io.LimitReader` and `responseCapture`. Non-destructively preserves full downstream request streaming using `io.MultiReader`. Emits `core.EventRequestLogged` to `core.Bus` and pipes logs to `RequestLogSink`.
  - **Kit Coordinator (`Kit`)**: Bundles project resolution, event bus, sink, and logging configurations into an ergonomic `Wrap(adapter, extractor)` builder.
  - Added unit test suite in `adapters/adapterkit/adapterkit_test.go` achieving 80.3% coverage.
  - Added depguard boundaries for `adapterkit` and `adapters` in `.golangci.yml`.
  - All tests (`task test`) and linters (`task lint`) pass cleanly.

### 2026-10-07

- Completed UI/UX overhaul based on BMS (Bulk Messaging Solutions) design inspiration and `docs/design.md` specifications:
  - Configured design tokens and color palette (`ember-50` through `ember-950`, cool neutral ramp, semantic colors) with Figtree and JetBrains Mono typography.
  - Implemented top header bar with BMS-style live stats pills (`Total`, `Delivered`, `Failed`), global `Ctrl+K` search, project switcher, and persistent `Sandbox: nothing is delivered` pill.
  - Implemented collapsible 248px/72px sidebar navigation with warm speech-bubble mark, duotone icons, active indicator bars, unread badges, local port status indicators (`HTTP :4010`, `SMTP :1025`), and dark mode toggle.
  - Built split-pane master/detail inbox with SMS conversation bubbles, sandboxed email plate (isolated iframe, headers, source tabs, remote-image toggle), 4-step delivery lifecycle progress track, and signature 1-click copyable CodeTiles for OTPs and links.
  - Built BMS-inspired EmptyState card and Onboarding Checklist card with copyable commands (`localhost:1025`, native API curl, Twilio/Termii SDK guides, `/otp/latest`).
  - Built interactive Compose Modal with SMS/Email/OTP tabs, country flag selector (`🇬🇭 +233`, etc.), templates, live GSM-7/UCS-2 segment counter, and failure simulation toggle.
  - Implemented OTPs grid page, Batches tracker page, Inspector logs page, and updated Settings page with accessible forms.
  - Developed full marketing landing page at `/landing` following the BMS visual shot and `docs/design.md` §11.3:
    - Hero section with headline ("Test every SMS, OTP and email. Pay nothing until launch."), 4-tab copyable installer commands (Homebrew, Docker, Linux/macOS, Windows Scoop), and interactive live simulation demo widget featuring live `CodeTile` and 4-step `DeliveryTrack`.
    - Key metric highlight pills ("100% Offline", "Single Binary", "Full Fidelity").
    - Signature ember statement bands ("Every message matters. Test every send before real users ever see it.").
    - 3-column feature grid ("Every tool your team needs") covering instant OTP extraction, sandboxed email preview, and webhook replay.
    - 3-column architecture capabilities ("Smarter testing, built-in") covering GSM-7/UCS-2 segments, magic pattern numbers, and synchronous wait endpoints.
    - Developer API showcase with dark code editor tabs for Twilio Node.js, Twilio Python, Native REST (cURL), and SMTP (Nodemailer).
    - Interactive "Free until production" volume slider calculating real cost savings against Twilio and Termii.
    - Accessible footer with quick links, documentation, and MIT open-source details.
  - Added `useSSE` hook for reactive real-time inbox refresh without polling. All Go tests pass, Biome checks pass with 0 errors, and web bundle builds cleanly.

### Recent Sessions

**2026-10-08 (Session 6): High-Craft $50k Developer-First UI/UX Overhaul & Interactive Storytelling Canvas**
- Elevated `LandingPage.tsx` into a high-tier developer tool design using `framer-motion`:
  - Added fluid scroll reveals and staggered fade-ins for all page sections (`<motion.section>`).
  - Added layered entry animations for the Hero Section, staggering the pill tag, main headline, subtitle, and CTA buttons.
  - Enhanced the Bento Grid feature cards with premium micro-interactions (translate-y elevation, subtle shadow expansion) on hover.
  - Enhanced the Main CTA buttons with responsive active down-scaling and smooth hover growth (`hover:scale-[1.03] active:scale-[0.97]`).
  - **Reimagined Interactive Storytelling Stage (`StoryScroll.tsx`)**: Built a cinematic 4-act pinned interactive studio sandbox (`01. SEND` -> `02. INTERCEPT` -> `03. EXTRACT` -> `04. SIMULATE`). Features smooth scroll-driven morphing between standard SDK code dispatch, 0ms local interceptor daemon logs, real-time SSE inbox with 1-click copy OTP extraction, and chaos/magic pattern failure simulation with interactive chapter scrubbing pills.
  - Perfected strict TypeScript adherence, Biome linting with 0 errors, and passing Go backend test suite.

**2026-10-07 (Session 5): Complete High-Fidelity Landing Page Matching mNotify BMS Reference**
- Rebuilt `web/src/pages/LandingPage.tsx` from the ground up to match the provided high-res screenshot (`media_1791393150451.png`):
  - Top Navigation with search trigger, Login link, and orange pill "Get Started" CTA.
  - Full-width hero banner with top pill tag, bold typography ("Africa's bulk messaging solution"), 6-icon service strip, and centered macOS browser showcase window with live interactive OTP digits (`5 8 9 2`).
  - Partner / ecosystem ticker strip (Paystack, Flutterwave, Kuda, Chipper, Paga, Interswitch).
  - Circular deliverability gauge section featuring a radial SVG 99.9% progress ring alongside real-time metrics (24/7 uptime, 0ms sandbox, 150+ carriers).
  - Full-width curved orange statement quote banner with transparent quotation iconography and white copy.
  - "Every tool your team needs" section with an interactive SMS Campaign Composer UI mockup (Quick SMS, Campaign, Schedule tabs, Sender ID badge, merge tags, character counter).
  - "Smarter messaging, built in" 3-card section with phone cleansing, route fallback histogram, and scheduled queue UI frames.
  - "Rich on features, enterprise ready" 6-card bento grid (Analytics, Sender IDs, Team Roles, Contact Groups, Multi-Channel Fallback, Bank-Grade Security dark card).
  - "Build with the BMS API" section featuring an orange polka-dot pattern background, step list, and macOS dark code window with language tabs.
  - "Pay for what you send" pricing section with service switch pills, 3 pricing tiers (featuring the solid vibrant orange "Growth Tier" card), volume estimation slider, and guarantee badges.
  - "There's more to BMS" 2-card section with JSON webhook payload preview and compliance audit UI cards.
  - Full-width closing orange CTA banner and 5-column comprehensive footer.
- All tests pass (`go test ./...`), Biome linter checks pass with 0 errors, and web bundle builds cleanly.

**2026-10-07 (Session 4): Marketing Landing Page Built**
- Completed M1-09: Marketing Landing Page Built - Implemented `web/src/pages/LandingPage.tsx` with dynamic feature previews, zero-cost volume pricing calculator, dual-column hero, code snippets for Twilio/Termii integration, 3-tier pricing layout, comprehensive footer. Matches aesthetic guidelines from `docs/design.md`.

**2026-10-06**
- Completed M1-14 through M1-19: Security middleware (Host allow-list, X-Mocksms header, no CORS, optional ui_auth), SMTP listener (AUTH, enmime, 25MB limit, STARTTLS), Retention prune job (cascading deletes), Web app scaffold (Vite, React, React Router, TanStack Query, Tailwind, shadcn/ui, lucide-react), Inbox UI (sidebar, message list, email plate, settings), cmd/mocksms serve wiring (HTTP + SMTP servers, retention pruner, graceful shutdown). All tests pass, web build succeeds.
- Completed M1-F18: Correct the docs - updated README.md to clarify product name is `mocksms` (module/repo is `CommPit`); corrected CHANGELOG.md entries (test helpers `otp/latest` and `emails/latest` are stubs, SMTP server deferred to M2, simulator async rules effective from M1-F08, migrations embedded); removed resolved "module path" blocker from progress.md; added note about M1-06/09/12/13 open fixes; updated techstack.md with confirmed Go 1.26 and pnpm choices. All docs now accurately reflect implemented behavior.
- Completed M1-F16: SMS segment counting - fixed UCS-2 length counting to use UTF-16 code units via `utf16.RuneLen(r)`, added form feed (`\f`) to GSM-7 extended table, removed segment count capping (returns uncapped count), added validation in core/service.go to reject messages over 10 segments with `validation_error` on field `body` ("message too long"). Updated tests to reflect correct segment calculations (307 GSM-7 = 2 segments, 135 UCS-2 = 2 segments). All tests pass.
- Completed M1-F15: Spec-first API - added oapi-codegen as Go tool, created api/generate.go with //go:generate directive, generated openapi.gen.go with ServerInterface, types, and chi server. Updated handler method signatures to match generated interface (GetProject, GetVerification, GetMessage, GetMessageRaw, GetAttachment, GetBatch, GetRequestLog, ReplayWebhook, GetLatestOTP, GetLatestEmail, ListProjects, ListMessages, ListVerifications, ListRequestLogs, ListWebhooks, DeleteMessages, WaitForMessage, SSEEvents). All routes now use adapter functions to extract params from request. All tests pass.
- Completed M1-F14: Long-lived requests - mounted `/events` and `/messages/wait` outside the global 60s timeout middleware. SSE handler now clears write deadline with `http.NewResponseController`, uses 15s heartbeat, registers clients by project filter, and emits `event: <type>` lines for EventSource filtering. `messages/wait` defaults `since` to request arrival time, accepts timeout as seconds or Go duration capped at 60s, returns 408 with canonical `wait_timeout` code, returns oldest match after `since`, and wakes on `message.created` bus events instead of polling. Added `ErrCodeWaitTimeout` to core errors with HTTP 408. All tests pass.
- Completed M1-F13: API scoping - split auth middleware into write (Bearer only, default project) and read/test (Bearer or ?project= with project existence validation). Added typed context key `projectKey{}` with error-returning accessor. DELETE /messages now requires explicit project via Bearer or ?project=, returns 400 `validation_error` otherwise. Replaced `bus` import with `core.Bus` in api package (handlers and SSE hub). Passed build version to `NewHandlers` and used in `/healthz`. All handler tests pass.
- Completed M1-F11: Simulator and OTP generation - moved simulator to new `sim` package implementing `core.Simulator`, wired in `cmd/mocksms`, and removed legacy simulator from `core`. Implemented thread-safe randomness with `math/rand/v2` and mutex-guarded test override. Combined latency and random failure rate effects so both apply to a send. Treated only `...999901` through `...999905` as magic pattern numbers, allowing other numbers to use standard latency/failure rates. Generated OTP verification codes using `crypto/rand` (`big.NewInt(10)` per digit), removing sleep and timestamp dependencies. Added `core.NoopSimulator` as default in `core.NewService`. Verified 10,000 generated codes have uniform digit distribution with exact length, table tests cover all magic numbers, and concurrent simulation passes under race detector.
- **Next:** M1-F12: Verification checks: atomic attempts, native API statuses per spec §7.2
- Completed M1-F10: Recipient handling - stored and matched recipients in normalized E.164 via phone.Parse/phone.Normalize, applying normalization to ListMessages/WaitForMessage filters, unsubscribe suppression records, and simulator matching. In ReceiveInbound, evaluated STOP and START keywords first using strings.ToLower and strings.TrimSpace, allowed START to remove unsubscribe suppression, and always accepted inbound messages without rejecting on unsubscribe status. Set CallbackURL to nil unless non-empty. In batches, validated and simulated each recipient individually, returning rejected recipients in batch response (openapi schema and core types updated with BatchRejectedRecipient) while delivering all valid recipients. Added tests in core/service_test.go and api/handlers_test.go.
- **Next:** M1-F11: Simulator: move to sim, goroutine-safe randomness, correct rule precedence; secure OTP generation
- Completed M1-F08: Lifecycle runner correctness - schedule by message ID instead of shared mutable pointer, reloading from store per transition step to eliminate concurrent data races. Unified transition logic so step-delay 0 loops synchronously without spawning background timers or duplicate events. Applied SimResult with AsyncFail ending in undelivered status with error code. Recomputed batch counts from transactional grouping queries and throttled event publishing to at most every 250ms. Added StatusEventIDPrefix ("sev_") and NewStatusEventID(). Added Store.ListInFlightMessages port method, called ResumeQueuedAndSent on server startup, and created deterministic virtual FakeClock with condition synchronization. All FakeClock and lifecycle concurrency tests pass.
- **Next:** M1-F10: Recipients: store normalised E.164, START after STOP, no empty callbacks, no silent batch drops
- Completed M1-F09: SQLite store fixes - enabled foreign_keys(1) and busy_timeout(5000) pragmas on both file and in-memory DSNs so ON DELETE CASCADE triggers across all child tables. Replaced readDBs slice with a single read pool with SetMaxOpenConns(readPoolSize) while writeDB has max 1 connection. Isolated in-memory stores with unique ULIDs (file:mocksms-<ulid>?mode=memory&cache=shared). Wrapped batch insertions in a single transaction in core.Service. Added storetest conformance tests for message cascade deletion, project cascade deletion, and in-memory store isolation. Added 10k batch insert benchmark passing in 0.57s (under 3s requirement).
- **Next:** M1-F08: Lifecycle runner: no shared mutable messages, single step-delay-0 path, apply sim results, batch counts, resume on startup
- Completed M1-F05: Clean shutdown and startup failures - made LifecycleRunner.Stop idempotent with sync.Once and tracked active timers with sync.WaitGroup. In cmd/mocksms, eliminated double shutdown and double store close, used signal.NotifyContext, routed ListenAndServe errors via error channel to exit with code 1 on port conflicts, and removed premature SMTP server log line. Added tests in cmd/mocksms/main_test.go and core/lifecycle_test.go.
- Completed M1-F04: API error responses now use `writeError` taking only the error, resolving canonical `*core.Error` via `errors.As` with proper HTTP status codes (`invalid_number` → 400, `rate_limited` → 429, `verification_not_found`/`not_found` → 404, `unauthorized` → 401). Unexpected errors are logged via `slog.Error` and masked as 500 `internal`. Fixed response header ordering by calling `render.Status` before `render.JSON` to ensure `Content-Type: application/json` is sent on all success and error responses. Added comprehensive unit tests in `api/handlers_test.go`.

- Completed M1-F07: Fixed event bus data race and subscription leak - used atomic.Bool for subscription closed state, Unsubscribe now removes subscription from slice under bus lock, Publish copies subscriber slice under lock and calls handlers outside lock, uses ULID for IDs. TestEventBus_ConcurrentPublishSubscribe passes under -race 100x.
- **Next:** M1-F04: API error responses - canonical codes, correct HTTP statuses, JSON content type

### 2026-10-04

- Completed M1-F03: Local gate matches CI - installed task, golangci-lint v2, pnpm; documented race detector setup for Windows (MinGW, WSL, Docker); updated Taskfile.yml to handle race detector gracefully; all Go tests pass locally
- **Next:** M1-F04: API error responses - canonical codes, correct HTTP statuses, JSON content type

### 2026-10-03

- Completed M1-04: `core` package with domain types (Project, Message, Batch, Verification, etc.), canonical errors with HTTP status codes, prefixed ULIDs (prj_, msg_, bat_, vrf_, whd_, req_, att_, blob_), ports (Store, BlobStore, Bus, Simulator, Clock, ProjectResolver), service with SendMessage/SendBatch/StartVerification/CheckVerification/ReceiveInbound, lifecycle runner with Clock-driven timers, FakeClock for tests
- **Next:** M1-05: `store/storetest` conformance suite

### 2026-10-03

- Completed M1-03: `extract` package with OTP code extraction (4-8 digits, keyword-adjacent priority for code/otp/pin/verification/token/passcode), link extraction from text and HTML (href), primary_link detection (verify/confirm/activate/magic/token/reset/login/signin in URL or anchor text)
- **Next:** M1-04: `core` domain types, canonical errors, prefixed ULIDs, ports (Store, BlobStore, Bus, Simulator, Clock, ProjectResolver), fake clock for tests

### 2026-10-03

- Completed M1-02: `phone` package with E.164 parsing (valid/possible/off modes), GSM-7/UCS-2 detection, segment counting (160/153 for GSM-7, 70/67 for UCS-2, max 10 segments)
- **Next:** M1-03: `extract` package (OTP codes, links from text/HTML, primary_link)

### 2026-10-03

- Completed M1-01: Repo scaffolding (`go.mod`, `Taskfile.yml`, `.golangci.yml` with `depguard`, `biome.json`, `.gitattributes`, `.editorconfig`, `.air.toml`, `LICENSE`, `README.md`, CI workflow)
- **Next:** M1-02: `phone` package (E.164 parsing, GSM-7/UCS-2 detection, segment counting)

### 2026-10-03

- Brainstormed the product: local-first sandbox, native core plus provider adapters, Go + embedded React UI, open-core (Apache-2.0).
- Wave 1 scope fixed: SMTP, Twilio (SMS + Verify), Termii; test API, webhooks, failure simulation, inbound, request inspector, go-live estimate, MCP server, CI kit; delivered as milestones M1–M4.
- Design spec approved and committed: `docs/superpowers/specs/2026-10-03-mocksms-design.md`.
- Created the project documentation set: overview, PRD, architecture, tech stack, tasks, progress, engineering, CHANGELOG, AGENTS.
- Refinements recorded in architecture.md decision log: ports live in `core` (ADR-007); native webhook format and signature defined (ADR-008).
- **Next:** resolve the module path, write the M1 implementation plan, start M1-01.
