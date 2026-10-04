# M1 progress review (M1-01 to M1-13)

- **Date:** 2026-10-04
- **Scope:** `main..milestone/m1` (13 commits, c062e62 to cfd6c38), plus repo hygiene, CI, and docs accuracy
- **Method:** read every non-generated Go file in scope; `go build`, `go vet`, `go test -cover` locally; GitHub Actions logs for the branch. No code was changed.

## Verdict

The structure follows the architecture well (ports in `core`, store conformance suite, prefixed ULIDs, per-task commits with `Refs:` footers and no AI attribution). But **M1-01 to M1-13 are not done by the project's own definition of done**: CI has failed on every push to `milestone/m1`, `core`, `api`, `config` and `cmd` have no tests, and several defects would break the product's core promises (canonical errors, clean shutdown, a shippable binary, STOP/START, recipient matching). Recommend fixing the critical items below before starting M1-14.

| Check | Result |
|---|---|
| `go build ./...`, `go vet ./...` | Pass |
| `go test ./...` (local, no race) | Pass; no tests in `api`, `cmd/mocksms`, `config`, `core` |
| Coverage | extract 99%, storetest 69%, phone 63%, store/sqlite 61%, core/api/config 0% |
| `gofmt -l` | 9 files unformatted: `api/handlers.go`, `api/sse.go`, `cmd/mocksms/main.go`, `config/config.go`, `core/lifecycle.go`, `core/ports.go`, `core/resolver.go`, `core/simulator.go`, `store/sqlite/verification_test.go` |
| CI on `milestone/m1` | **Failed on all runs** (lint config, race test, missing `web/`) |

---

## 1. CI and tooling

| # | Finding | Evidence |
|---|---|---|
| CI-1 | **Lint never runs.** `.golangci.yml` uses v1 syntax (`linters-settings`, `gosimple`, `typecheck`) but CI installs golangci-lint `latest` (v2): "can't load config: can't unmarshal config ... 4 error(s)". So `depguard` has never enforced the architecture. | CI run 37170611412, Lint job |
| CI-2 | **The depguard rule is wrong even once it loads.** It has no file scoping, so it denies `cmd/mocksms` its own imports (`store/sqlite`, `config`, `api`, `bus`), and it allows `api` → `bus` nowhere, yet `api` imports `bus`. | `.golangci.yml:24-58` |
| CI-3 | **A real race-test failure is hidden locally.** `TestEventBus_ConcurrentPublishSubscribe` fails on Linux, macOS and Windows runners with `-race`. Locally `-race` cannot run (cgo disabled), so `task test` was never actually green. | CI Test jobs; `go test -race` → "requires cgo" |
| CI-4 | **UI jobs fail because `web/` does not exist yet** (lint's Biome step, `test-web`, `build`). Gate them on the folder existing until M1-17, or add them in M1-17. | CI UI Tests job |
| CI-5 | Go version drift: `go.mod` says `go 1.26.0`, CI pins `1.25`, docs say "≥ 1.25". The local toolchain is also mixed (go1.26.0 binary, go1.25.5 tools: `go test -cover` prints "version does not match"). | `go.mod:3`, `ci.yml:10` |
| CI-6 | Three package managers: docs say pnpm, `Taskfile.yml` uses npm, CI uses bun. Pick one. | techstack.md, Taskfile.yml, ci.yml |
| CI-7 | `golangci-lint` and `task` are not installed locally, so the AGENTS.md "run `task test` and `task lint`" gate cannot be met on this machine. | local shell |
| CI-8 | Every dependency in `go.mod` is marked `// indirect`; `go mod tidy` has not been run. | `go.mod` |
| CI-9 | No `//go:generate` directive exists, so `task generate` (`go generate ./...`) regenerates nothing. | repo-wide grep |

## 2. Critical defects

| # | Where | Defect | Failure |
|---|---|---|---|
| C-1 | `api/handlers.go:136-140` | `h.error` calls `core.IsError(err, "")`, which is only true for an error with an empty code. Every `core.Error` has a code, so **every API error is rewritten to `internal`**. | `POST /sms` to `+15005550001` returns `{"code":"internal", ...}` instead of `invalid_number`; clients and the future adapters can never see canonical codes |
| C-2 | `core/lifecycle.go:217-218` with `cmd/mocksms/main.go:128-131,150` | `service.Shutdown()` is called twice (in the `ctx.Done` goroutine and again after it), and `Stop()` does `close(lr.stopCh)` each time. | Ctrl+C or SIGTERM panics with "close of closed channel" on every shutdown |
| C-3 | `store/sqlite/store.go:94-101` | Migrations are located with `runtime.Caller(0)`, i.e. read from the source tree at runtime, not embedded (the CHANGELOG says "embedded"). | The released binary, the Docker image, or any machine without the source checkout fails at startup ("run migrations: ...") |
| C-4 | `bus/bus.go:37-41,63-65` | `Unsubscribe` writes `sub.closed` without a lock while `Publish` reads it outside the lock; unsubscribed subscriptions are never removed. | Data race (CI fails); every subscribe/unsubscribe leaks a handler for the process lifetime |
| C-5 | `core/lifecycle.go:56-62,104-105` | The lifecycle goroutines mutate the same `*Message` that `SendSMS` is JSON-encoding into the response and the SSE hub is marshalling. | Data race; torn or wrong status in responses and events |
| C-6 | `core/lifecycle.go:76-83,148-173` | With `step_delay: 0`, `advanceImmediately` loops while `advance(sent)` also spawns a 0-delay timer goroutine that advances the same message. | Duplicate `delivered` status events and webhooks; races on the message |

## 3. Correctness

### core

| # | Where | Defect | Failure |
|---|---|---|---|
| K-1 | `core/service.go:366-370` | `ReceiveInbound` rejects any inbound from an unsubscribed number before checking for START keywords. | After STOP, a reply of START is rejected with `unsubscribed`; the number can never resubscribe (spec §8.4) |
| K-2 | `core/service.go:424-430,76,156` | The E.164-normalised number from `phone.Parse` is discarded (`_ = parsed`); the raw input is stored and matched. | `+233 24 123 4567` and `+233241234567` are different recipients: unsubscribe, `otp/latest`, `messages/wait?to=` and sim rules miss |
| K-3 | `core/service.go:503-511` | `generateCode` takes `time.Now().UnixNano()%10` after `time.Sleep(1)`, bypassing the injected `Clock` and sleeping in domain code. | Low-entropy, often repeating codes (e.g. `000000` on coarse Windows timers) |
| K-4 | `core/service.go:156-164,212` | Batch recipients that fail validation or simulation are silently dropped; batch members are always scheduled with an empty `SimResult`. | Batches lose recipients without a trace; async failures never apply to batches; violates "never swallow errors" |
| K-5 | `core/service.go:423-431` vs `156-159` | `validateSendRequest` rejects the whole batch if one number fails parsing, but `sendBatch` later skips bad recipients individually. | Inconsistent behaviour depending on which check fails first |
| K-6 | `core/service.go:103,184` | `CallbackURL: &req.CallbackURL` is always a non-nil pointer, even when empty. | Messages store `""` as a callback URL; the M3 webhook worker will try to POST to it |
| K-7 | `core/lifecycle.go:85-102` | `advance` always goes queued → sent → delivered; `SimResult.AsyncFail` and `Hang` are never applied. | Magic numbers `…999902`, `…999905` and `sim.failure_rate` still deliver |
| K-8 | `core/lifecycle.go:131-145` | Batch counts are read-modify-written from concurrent timer goroutines without a transaction. | Wrong batch counts under load (the 10k-batch case) |
| K-9 | `core/lifecycle.go:114` | Status events use `NewWebhookDeliveryID()`. | Status event IDs carry the `whd_` prefix |
| K-10 | `core/lifecycle.go:187-207`, `store/sqlite/store_crud.go:250` | `ResumeQueuedAndSent` is never called at startup, and it lists with `projectID ""`, which the store matches literally. | Messages mid-lifecycle at restart stay `queued`/`sent` forever (M1-09 claims "resume on startup") |
| K-11 | `core/lifecycle.go:16,225` | `wg` is never incremented; `Stop` cannot wait for in-flight goroutines. Store errors in `advance` are swallowed. | Writes after the store is closed during shutdown; silent lost transitions |
| K-12 | `core/simulator.go:21,89` | A single `math/rand.Rand` is used from concurrent requests outside the lock. | Data race on the random source |
| K-13 | `core/simulator.go:102,124` | Rules match only `req.To[0]`; `sendBatch` passes the whole request for every member. | Every batch member is simulated as the first recipient |
| K-14 | `core/simulator.go:84-96,128-129` | Global latency returns before the failure-rate check; any `…99990X` number (including X = 0, 6-9) skips global settings. | `failure_rate` silently ignored whenever latency is set |
| K-15 | `core/service.go:316-358` | Check-verification is read-modify-write without a transaction; expired and max-attempts return `200`, while spec §7.2 says `verification_not_found` 404 / `max_attempts` 429 for the native API. | Concurrent checks can exceed `max_attempts`; native API diverges from the spec |

### api

| # | Where | Defect | Failure |
|---|---|---|---|
| A-1 | `api/handlers.go:144,193-205` and similar | Handlers pass a fixed status (mostly 400) to `h.error` instead of `core.Error.HTTPStatus()`. | `rate_limited` → 400 not 429; store failures → 400 not 500 |
| A-2 | `api/handlers.go:199-204` | `w.WriteHeader(201)` is called before `render.JSON`, which sets `Content-Type` afterwards. | 201/202 responses go out without `Content-Type: application/json` |
| A-3 | `api/handlers.go:115-117` | `?project=` is accepted on every route (writes included) and is never checked to exist; a failed Bearer resolve error is overwritten. | Any caller can write into any project ID, including non-existent ones; spec allows `?project=` for read/test endpoints only |
| A-4 | `api/handlers.go:572-582`, `store_crud.go:308` | `DELETE /messages` without a key falls back to the `default` project instead of `400`. | An unscoped reset silently wipes the default project (spec §7.1) |
| A-5 | `api/handlers.go:584-636` | `messages/wait`: `since` defaults to nil (spec: request time), `timeout` parses `timeout+"s"` so `10s` fails silently, no 60s cap, returns code `internal` instead of `wait_timeout`, returns the newest match rather than the first after `since`, polls the DB every 100ms instead of waking on the bus. | Tests receive stale OTPs from earlier runs; timeouts behave unexpectedly |
| A-6 | `api/handlers.go:36,91`, `main.go:121` | Global `middleware.Timeout(60s)` and `http.Server.WriteTimeout: 30s` apply to `/events`. | SSE streams are cut after 30s; the 30s heartbeat cannot keep them alive |
| A-7 | `api/sse.go:55-64,94` | The `?project=` filter is registered but never applied when broadcasting. | Every SSE client receives every project's events |
| A-8 | `api/handlers.go:638-656` | `otp/latest` and `emails/latest` return 501. | Contradicts the CHANGELOG ("test helpers (messages/wait, otp/latest, emails/latest)"); they belong to M2-10 anyway |
| A-9 | `api/handlers.go:31-92` vs `api/openapi.go` | Routes are hand-written; the generated `ServerInterface`/router is not used. | The spec-first contract (engineering.md §6) is not enforced; handlers can drift from `openapi.yaml` |
| A-10 | `api/handlers.go:126,133` | A string context key (`"projectID"`) plus an unchecked type assertion. | Key collisions (staticcheck SA1029); a panic if the middleware is ever bypassed |
| A-11 | `api/handlers.go:97-101` | `/healthz` hardcodes `"version": "0.1.0"`. | Wrong version once releases are built with ldflags |

### store/sqlite

| # | Where | Defect | Failure |
|---|---|---|---|
| S-1 | `store/sqlite/store.go:61-91` | `PRAGMA foreign_keys = ON` is never set. | Every `ON DELETE CASCADE` in the schema is inert; deletes leave orphaned status events, verifications, attachments and blobs (also breaks M1-16 retention) |
| S-2 | `store/sqlite/store.go:103-109` | `getReadDB` always returns `readDBs[0]`. | The read pool (default 4) is unused; all reads are serialised on one connection |
| S-3 | `store/sqlite/store.go:63-64` | `:memory:` uses `file::memory:?cache=shared`, one database per process. | Separate `NewStore(":memory:")` calls (tests, or two instances in one process) share state; shared-cache mode raises `SQLITE_LOCKED` under concurrency |
| S-4 | `core/service.go:138-229` | `store.Transaction` exists but `sendBatch` inserts one message per statement. | 10,000 separate write transactions per bulk send; misses the ≤3s budget and the spec's "single transaction" |

### cmd/mocksms

| # | Where | Defect | Failure |
|---|---|---|---|
| M-1 | `main.go:134-139` | A `ListenAndServe` error (e.g. port 4010 in use) is only printed. | The process keeps running with no HTTP server and never exits |
| M-2 | `main.go:141-144` | "Starting SMTP server" is printed but SMTP is a TODO. | Misleading output until M1-15 |
| M-3 | `main.go:66,153` | `store.Close()` deferred and also called explicitly. | Double close (harmless today, fragile) |
| M-4 | `main.go` | No data-directory creation, no startup banner, no lifecycle resume. | First run fails if the data dir does not exist (part of M1-19, noted for tracking) |

### phone

| # | Where | Defect | Failure |
|---|---|---|---|
| P-1 | `phone/encoding.go` `countSegmentsUCS2` | Length counted in runes, not UTF-16 code units. | Emoji under-counted: 70 emoji = 140 units = 3 segments, reported as 1 |
| P-2 | `phone/encoding.go` | Segments silently capped at 10. | A 2,000-character message reports 10 segments instead of a "message too long" validation error |

## 4. Architecture and spec rules

| # | Rule (source) | Finding |
|---|---|---|
| R-1 | "`api` may import only `core`" (architecture.md §3) | `api` imports `bus` (`api/sse.go`, `handlers.go`); it should take `core.Bus` |
| R-2 | "`sim` implements `core.Simulator`" (architecture.md §3) | The simulator lives in `core/simulator.go`; `core` now also imports `math/rand` |
| R-3 | "Time comes from the injected `core.Clock`" (engineering.md §2) | `generateCode` and the simulator seed use `time.Now()`; `generateCode` sleeps |
| R-4 | "Never `time.Sleep` in tests" (AGENTS.md) | 10 `time.Sleep` calls in tests; the failing bus test depends on one |
| R-5 | "Test first" / coverage ≥85% for core (engineering.md §4) | `core` has 0% coverage and no tests; verification behaviour is tested from `store/sqlite/verification_test.go` |
| R-6 | "New `Store` behaviour gets a case in `store/storetest`" | Behaviour was added, but the in-memory reference store lives in `storetest/memstore_test.go` (a test file), so it cannot be reused |
| R-7 | Canonical errors with HTTP mapping (spec §7.2) | Mapping exists in `core.Error.HTTPStatus()` but is never used by the API (see C-1, A-1) |

## 5. Repository hygiene and process

| # | Finding |
|---|---|
| H-1 | **No `.gitignore`.** `mocksms.exe` (~23 MB) was committed in five task commits and pushed; history now carries ~115 MB of binaries. AGENTS.md: "Don't commit generated UI builds or local data". Purging requires a history rewrite on a pushed branch, which is your call. |
| H-2 | `.impeccable/questions/*` (decision-page session state) is tracked and changes in every commit, and `.impeccable/questions/af69b629.state.json` is modified in the working tree now. It should be ignored. |
| H-3 | **Every task was pushed with CI red**, against AGENTS.md "Never commit or push while any test or lint check fails". The local gate could not catch the race or lint failures (CI-3, CI-7). |
| H-4 | M1-11 and M1-12 share one commit (ed908fe); M1-11 was marked done as "already implemented earlier" inside M1-08's commit. |
| H-5 | Good: every commit uses Conventional Commits with a `Refs:` footer, no AI attribution anywhere, and tasks/progress/CHANGELOG are updated in the same commit as the code. |

## 6. Documentation accuracy

| # | Finding |
|---|---|
| D-1 | **Name drift:** the module and repo are `github.com/Aeomar999/CommPit`, while the binary, docs, PRODUCT.md and design.md say `mocksms`. Decide whether CommPit is the product name and update the docs (or record that the module name is internal). |
| D-2 | progress.md still lists "Final name and GitHub organization (Go module path)" as a blocker for M1-01, but it has been resolved. |
| D-3 | CHANGELOG overstates: "goose migrations (embedded)" (C-3), "HTTP + SMTP servers with graceful shutdown" (SMTP is a TODO; shutdown panics), "test helpers (messages/wait, otp/latest, emails/latest)" (two return 501), "Simulator with built-in rules" (async rules have no effect, K-7). |
| D-4 | progress.md marks M1-09 "resume on startup" and M1-06 "passes storetest" complete; resume is never wired (K-10), and the store's cascade behaviour is untested because foreign keys are off (S-1). |
| D-5 | techstack.md lists pnpm; the repo uses npm and bun (CI-6). |

---

## Suggested order of fixes (before M1-14)

1. **Unblock CI:** migrate `.golangci.yml` to v2 syntax with file-scoped depguard rules; gate the web jobs until `web/` exists; align the Go version; add `.gitignore` (binaries, `*.db`, `web/dist`, `.impeccable/questions/`).
2. **Critical defects:** C-1 (error mapping), C-2 (double shutdown panic), C-3 (embed migrations with `embed.FS`), C-4 and C-5 (bus race and shared-message race), C-6 (step-delay 0).
3. **Core promises:** K-1 (START after STOP), K-2 (normalised numbers), S-1 (foreign keys), A-1/A-2 (status codes, content type), A-6 (SSE timeouts).
4. **Tests:** add `core` service and lifecycle tests with `FakeClock`, `api` handler tests with `httptest`, and enable `-race` locally (install a C toolchain or run tests in WSL/Docker) so the gate matches CI.
5. **Docs:** correct the CHANGELOG and progress entries (D-2 to D-4) and settle the name (D-1).
