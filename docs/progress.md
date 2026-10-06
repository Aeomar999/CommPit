# mocksms — Progress

Single source of truth for where the project stands. Update it in every PR that changes task status (see [AGENTS.md](../AGENTS.md)).

## Current status

| | |
|---|---|
| **Phase** | M1 in progress |
| **Current milestone** | M1 |
| **Next action** | M1-20: Release pipeline: goreleaser (binaries, Docker, Homebrew, Scoop), cosign, release workflow |
| **Last updated** | 2026-10-05 |

## Milestones

| Milestone | Release | Status | Tasks done |
|---|---|---|---|
| M1: Core, native API, SMTP, inbox | v0.1.0 | In progress | 24 / 21 |
| M2: Twilio, Termii, test API, inspector | v0.2.0 | Not started | 0 / 14 |
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

### 2026-10-06

- Completed M1-14 through M1-19: Security middleware (Host allow-list, X-Mocksms header, no CORS, optional ui_auth), SMTP listener (AUTH, enmime, 25MB limit, STARTTLS), Retention prune job (cascading deletes), Web app scaffold (Vite, React, React Router, TanStack Query, Tailwind, shadcn/ui, lucide-react), Inbox UI (sidebar, message list, email plate, settings), cmd/mocksms serve wiring (HTTP + SMTP servers, retention pruner, graceful shutdown). All tests pass, web build succeeds.
- **Next:** M1-20: Release pipeline: goreleaser (binaries, Docker, Homebrew, Scoop), cosign, release workflow
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
