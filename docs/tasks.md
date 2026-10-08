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
| [x] | M1-08 | Core service: projects and credentials (auto-create), `SendMessage`, `SendBatch`, validation, extraction on save | M1-02, M1-03, M1-06, M1-07 | REQ-001, REQ-010, REQ-091 |
| [x] | M1-09 | Lifecycle runner: Clock-driven timers, configurable step delay, resume on startup | M1-08 | REQ-050 |
| [x] | M1-10 | Verification flow: start, check, attempts, expiry, `otp.fixed_code` | M1-08 | REQ-020, REQ-021, REQ-022 |
| [x] | M1-11 | `config`: koanf with flags > env > YAML > defaults; all settings from spec §9 | M1-01 | REQ-092 |
| [x] | M1-12 | `openapi/openapi.yaml` for M1 endpoints; oapi-codegen setup; `api` handlers for send, read, verifications, projects, `/healthz` | M1-08, M1-10 | REQ-010–REQ-014 |
| [x] | M1-13 | SSE hub and `GET /api/v1/events` | M1-07, M1-12 | REQ-002 |
| [x] | M1-14 | Security middleware: `Host` allow-list (421), `X-Mocksms` header, no CORS, optional `ui_auth`, non-loopback warning | M1-12 | — |
| [x] | M1-15 | `smtpd`: listener, AUTH username → project, enmime parsing, 25 MB limit, optional STARTTLS | M1-08 | REQ-035 |
| [x] | M1-16 | Retention prune job with cascading deletes | M1-06 | REQ-093 |
| [x] | M1-17 | Web app scaffold: Vite, Tailwind, shadcn/ui, TanStack Router + Query, openapi-typescript/openapi-fetch, `web/embed.go`, Vite proxy for development; design tokens, fonts, Phosphor icons and themed browser surfaces from [design.md](design.md) §5–§9 and §15; record the direction contract (design.md §17) with `impeccable surface-brief write` | M1-12 | — |
| [x] | M1-18 | Inbox UI per [design.md](design.md) §10–§11: sidebar, top bar with Sandbox pill, project switcher, message list, code tiles, delivery track, SMS threads, email plate (sandboxed iframe, remote-image toggle), filters, get-started checklist, settings page, live updates; finish with `impeccable detect`, the impeccable finish review, and the impeccable documenter writing root `DESIGN.md` + `.impeccable/design.json` | M1-13, M1-17 | REQ-002–REQ-007 |
| [x] | M1-19 | `cmd/mocksms serve`: wiring, startup/shutdown order (architecture §7), banner | M1-09–M1-16 | — |
| [x] | M1-20 | Release pipeline: goreleaser (binaries, Docker, Homebrew, Scoop), cosign, release workflow | M1-19 | — |
| [x] | M1-21 | Docs: README quick start; SMTP setup for Laravel, Django, Rails, Nodemailer, Spring | M1-19 | — |

### M1 review fixes (from the 2026-10-04 review)

The [M1 progress review](reviews/2026-10-04-m1-progress-review.md) of M1-01 to M1-13 found that CI has failed on every push to `milestone/m1` and that several defects break core promises. These tasks fix them. **Do M1-F01 to M1-F03 first** (they make the test gate real), then the Critical tasks, **before starting M1-14**. Tasks M1-06, M1-09, M1-12 and M1-13 stay marked done, but their open defects are tracked here. Review finding IDs (C-1, K-2, …) are in brackets.

Same rules as every task: test first, `task test` and `task lint` green, one commit per task with a `Refs: M1-Fxx` footer, pushed to `milestone/m1`.

| | ID | Fix | Severity | Depends on |
|---|---|---|---|---|
| [x] | M1-F01 | Make CI run: golangci-lint v2 config with scoped depguard, gate web jobs, align Go version, tidy modules, gofmt | Critical | — |
| [x] | M1-F02 | Repo hygiene: `.gitignore`, untrack the binary and impeccable session state | High | — |
| [x] | M1-F03 | Make the local gate match CI: `-race` on Windows, install `task` and `golangci-lint`, one package manager | High | — |
| [x] | M1-F04 | API error responses: canonical codes, correct HTTP statuses, JSON content type | Critical | M1-F01 |
| [x] | M1-F05 | Clean shutdown and startup failures (no double-close panic, exit on port conflict) | Critical | M1-F01 |
| [x] | M1-F06 | Embed database migrations in the binary | Critical | M1-F01 |
| [x] | M1-F07 | Event bus race and leak: remove the data race and the subscription leak; deterministic tests | Critical | M1-F03 |
| [x] | M1-F08 | Lifecycle runner: no shared mutable messages, single step-delay-0 path, apply sim results, batch counts, resume on startup | Critical | M1-F07, M1-F09 |
| [x] | M1-F09 | SQLite store: foreign keys on, real read pool, isolated in-memory stores, transactional batch insert | High | M1-F06 |
| [x] | M1-F10 | Recipients: store normalised E.164, START after STOP, no empty callbacks, no silent batch drops | High | M1-F09 |
| [x] | M1-F11 | Simulator: move to `sim`, goroutine-safe randomness, correct rule precedence; secure OTP generation | High | M1-F08 |
| [x] | M1-F12 | Verification checks: atomic attempts, native API statuses per spec §7.2 | Medium | M1-F04 |
| [x] | M1-F13 | API scoping: `?project=` only where allowed, scoped reset, typed context key, `api` depends on `core.Bus` | High | M1-F04 |
| [x] | M1-F14 | Long-lived requests: SSE and `messages/wait` survive timeouts, SSE project filter, wait semantics | High | M1-F13 |
| [x] | M1-F15 | Spec-first API: generate the server from `openapi.yaml` and fail CI on drift | Medium | M1-F13, M1-F14 |
| [x] | M1-F16 | SMS segments: UTF-16 counting for UCS-2, reject messages over the segment limit | Medium | M1-F01 |
| [x] | M1-F17 | Backfill tests for `core`, `api`, `config`; reusable in-memory store | High | M1-F07 |
| [x] | M1-F18 | Correct the docs: CHANGELOG, progress, product name, package manager | Medium | — |

#### M1-F01 — Make CI run

**Why.** Every push to `milestone/m1` has failed CI, so nothing has been checked by lint and `depguard` has never enforced the architecture. Three causes: `.golangci.yml` is written in golangci-lint v1 syntax but CI installs v2 ("can't load config"); the depguard rule has no file scoping, so it would also block `cmd/mocksms` from its own imports; and the UI jobs run in a `web/` folder that does not exist yet. [CI-1, CI-2, CI-4, CI-5, CI-8, gofmt]

**Where.** `.golangci.yml`, `.github/workflows/ci.yml`, `go.mod`, the 9 files `gofmt -l` reports.

**Do.**
1. Run `golangci-lint migrate` (v2) to convert the config, then remove `gosimple` and `typecheck` (merged or removed in v2) and move `gofmt`/`goimports` under `formatters:`.
2. Replace the single depguard rule with one rule per package group from [architecture.md](architecture.md) §3, each scoped with `files:`. Example:
   ```yaml
   linters:
     settings:
       depguard:
         rules:
           core:
             files: ["**/core/*.go"]
             list-mode: lax
             deny:
               - pkg: "github.com/Aeomar999/CommPit/store"
                 desc: "core may import only phone and extract"
               - pkg: "github.com/Aeomar999/CommPit/bus"
                 desc: "core may import only phone and extract"
               - pkg: "github.com/Aeomar999/CommPit/api"
                 desc: "core may import only phone and extract"
   ```
   Add rules for `api` (only `core`), `adapters/*` (no `store/...`), `store/sqlite`, `bus`, `sim` (only `core`, `phone`). Leave `cmd/mocksms` unrestricted.
3. Pin the lint action and version (`golangci/golangci-lint-action@v8` with an explicit `version: v2.x`) instead of `latest`.
4. In CI, replace `go-version: '1.25'` with `go-version-file: go.mod`, and update the "Go ≥ 1.25" line in [techstack.md](techstack.md) to match `go.mod`.
5. Gate every web step with `if: hashFiles('web/package.json') != ''` until M1-17 creates `web/`.
6. Run `go mod tidy` (all requirements are currently `// indirect`) and `gofmt -w` on the 9 listed files.

**Done when.** A push to `milestone/m1` shows the Lint job running depguard, and `golangci-lint run` reports the known `api → bus` violation (fixed in M1-F13). No job fails because `web/` is missing.

#### M1-F02 — Repo hygiene

**Why.** There is no `.gitignore`. `mocksms.exe` (about 23 MB) was committed in five task commits and pushed, and `.impeccable/questions/*` (a design session's state) changes in every commit. AGENTS.md forbids committing builds and local data. [H-1, H-2]

**Do.**
1. Add `.gitignore` with at least: `/bin/`, `*.exe`, `/mocksms`, `*.db`, `*.db-wal`, `*.db-shm`, `web/dist/`, `web/node_modules/`, `.impeccable/questions/`, `.impeccable/review/`, `coverage.out`, `.env`.
2. Untrack without deleting locally: `git rm --cached mocksms.exe` and `git rm --cached -r .impeccable/questions`.
3. Decide separately whether to purge the binary from pushed history (needs `git filter-repo` and a force-push of `milestone/m1`). Record the decision in [progress.md](progress.md). Not required to close this task.

**Done when.** `git status` stays clean after `task build` and a design session, and no tracked file is larger than 1 MB.

#### M1-F03 — Make the local gate match CI

**Why.** AGENTS.md requires `task test` (with `-race`) and `task lint` before every commit, but on this machine `-race` cannot run (it needs cgo and a C compiler) and neither `task` nor `golangci-lint` is installed. That is why a race failure was pushed in every commit. The repo also names three package managers: pnpm in the docs, npm in `Taskfile.yml`, bun in CI. [CI-3, CI-6, CI-7]

**Do.**
1. Install a C toolchain for the race detector, e.g. `choco install mingw`, then `go env -w CGO_ENABLED=1`. Alternatively run `task test` inside WSL or Docker and say so in [engineering.md](engineering.md) §14.
2. Install Task (`winget install Task.Task`) and golangci-lint v2 (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`).
3. Choose one JavaScript package manager (the docs say pnpm) and use it in `Taskfile.yml`, CI and [techstack.md](techstack.md).
4. Fix the local toolchain mismatch (go1.26.0 binary with go1.25.5 tools) by reinstalling Go once.

**Done when.** `task test` runs `go test -race ./...` locally and fails today on the bus race (fixed in M1-F07), and `task lint` runs.

#### M1-F04 — API error responses

**Why.** `h.error` calls `core.IsError(err, "")`, which is only true for an error with an empty code, so **every error is rewritten to `internal`**. Handlers also pass a fixed status (mostly 400) instead of the error's own, so `rate_limited` returns 400 rather than 429. Success responses call `w.WriteHeader(201)` before `render.JSON`, so `Content-Type: application/json` is never sent. [C-1, A-1, A-2]

**Where.** `api/handlers.go:136-152` and every handler that calls `h.error` or `w.WriteHeader`.

**Do.**
1. Change the helper to take only the error:
   ```go
   func (h *Handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
       var ce *core.Error
       if !errors.As(err, &ce) {
           slog.Error("unexpected error", "path", r.URL.Path, "err", err)
           ce = core.NewInternal("internal error")
       }
       w.Header().Set("Content-Type", "application/json")
       w.WriteHeader(ce.HTTPStatus())
       _ = json.NewEncoder(w).Encode(map[string]any{"error": ce})
   }
   ```
   Make `core.IsError` use `errors.As` too.
2. Remove the status argument from every call site.
3. For success responses use `render.Status(r, http.StatusCreated)` followed by `render.JSON(w, r, v)`, never `WriteHeader` first.

**Done when.** Handler tests (httptest) show `+15005550001` → 400 `invalid_number`, a rate-limit rule → 429 `rate_limited`, a store failure → 500 `internal`, and every response carries `Content-Type: application/json`.

#### M1-F05 — Clean shutdown and startup failures

**Why.** `main.go` calls `service.Shutdown()` twice (in a goroutine on `ctx.Done()` and again after the HTTP server stops), and `LifecycleRunner.Stop()` closes `stopCh` each time, so **every Ctrl+C panics**. If port 4010 is taken, `ListenAndServe`'s error is only printed and the process hangs with no server. `store.Close()` also runs twice, and "Starting SMTP server" is printed although SMTP is not implemented. [C-2, K-11, M-1, M-2, M-3]

**Where.** `cmd/mocksms/main.go`, `core/lifecycle.go:213-226`.

**Do.**
1. Make `Stop()` idempotent with `sync.Once`, and track every timer goroutine with `wg.Add(1)` / `defer wg.Done()` so `Stop()` really waits.
2. Keep one shutdown sequence in `run()` (architecture.md §7): stop HTTP, stop lifecycle, close store once. Delete the extra `ctx.Done()` goroutine and the deferred `store.Close()` (or keep only the defer).
3. Send `ListenAndServe` errors to an error channel; on any error other than `http.ErrServerClosed`, cancel the context and return the error so the process exits non-zero.
4. Remove the SMTP log line until M1-15 adds the listener.

**Done when.** A test or manual run shows SIGINT exits with code 0 and no panic, and starting a second instance on the same port exits with code 1 and a clear message.

#### M1-F06 — Embed database migrations

**Why.** `runMigrations` finds the SQL files through `runtime.Caller(0)`, i.e. from the source tree on the build machine. The released binary and the Docker image will fail at startup ("run migrations"). The CHANGELOG says they are embedded. [C-3]

**Where.** `store/sqlite/store.go:94-101`.

**Do.**
```go
//go:embed migrations/*.sql
var migrations embed.FS

func runMigrations(db *sql.DB) error {
    goose.SetBaseFS(migrations)
    if err := goose.SetDialect("sqlite3"); err != nil {
        return err
    }
    return goose.Up(db, "migrations")
}
```

**Done when.** A binary built with `go build -o /tmp/x ./cmd/mocksms`, copied outside the repo and run with `--data-dir` pointing at an empty folder, starts and creates the schema.

#### M1-F07 — Event bus race and leak

**Why.** `Unsubscribe` writes `sub.closed` with no lock while `Publish` reads it outside the lock: a data race, which is why `TestEventBus_ConcurrentPublishSubscribe` fails under `-race` in CI. Unsubscribed handlers are never removed, so every SSE connection leaks handlers for the life of the process. `generateID` returns the same string every time. The failing test also depends on `time.Sleep`, which AGENTS.md bans. [C-4, R-4]

**Where.** `bus/bus.go`, `bus/bus_test.go`.

**Do.**
1. Give each subscription a pointer back to the bus and its event type; `Unsubscribe` takes the bus lock and removes the subscription from the slice (make it idempotent).
2. In `Publish`, copy the subscriber slice under the read lock and call handlers outside it. Document that handlers must not block (the SSE hub already uses non-blocking sends).
3. Generate IDs with ULIDs (or an atomic counter); delete `randomString`.
4. Rewrite the concurrency test without sleeps: subscribe N handlers, publish M events, wait on a `sync.WaitGroup` or channel, assert exactly N×M deliveries; then unsubscribe and assert zero further deliveries; add a test that unsubscribing actually shrinks the subscriber list.

**Done when.** `go test -race ./bus/...` passes 100 times in a row (`-count=100`).

#### M1-F08 — Lifecycle runner correctness

**Why.** The lifecycle goroutines mutate the same `*Message` the HTTP handler is encoding into its response and the SSE hub is marshalling (data race). With `step_delay: 0`, `advanceImmediately` loops while `advance(sent)` also spawns a timer that advances the same message, producing duplicate `delivered` events. Simulation results (`AsyncFail`, `Hang`) are ignored, so failure magic numbers still deliver. Batch counts are read-modify-written from concurrent goroutines. Status events get `whd_` IDs. `ResumeQueuedAndSent` is never called, and it lists with project `""`, which matches nothing, so M1-09's "resume on startup" does not happen. [C-5, C-6, K-7, K-8, K-9, K-10]

**Where.** `core/lifecycle.go`, `core/types.go`, `cmd/mocksms/main.go`, `core/ports.go` (new store method).

**Do.**
1. Schedule by message ID, not pointer. Each step reloads the message from the store, applies the transition, saves it, and publishes a copy. Nothing outside the runner ever sees a pointer the runner mutates.
2. One code path decides the next step: `advance` performs a single transition and returns; `Schedule` either loops synchronously (delay 0) or arms one timer per step. Never both.
3. Apply `SimResult`: `AsyncFail` makes the delivery step end in `undelivered` or `failed` with the given error code and message; `Delay` adds to each step. `Hang` belongs to the HTTP layer (hold the response until the hang duration or client disconnect, then return 504 `provider_unavailable`), not the lifecycle.
4. Batch counts: recompute from `SELECT status, COUNT(*) FROM messages WHERE batch_id = ? GROUP BY status` inside a transaction, and publish `batch.updated` at most every 250ms per batch.
5. Add `StatusEventIDPrefix = "sev_"` and `NewStatusEventID()`; add it to the ID list in [architecture.md](architecture.md) §4.
6. Add a `Store.ListInFlightMessages(ctx)` port method (all projects, status `queued` or `sent`) with a `storetest` case, and call `ResumeQueuedAndSent` in `run()` after the store opens and before HTTP starts.
7. Log store errors with `slog` instead of returning silently.

**Done when.** FakeClock tests cover: delay 0 produces exactly one `sent` and one `delivered` event; delay > 0 advances only when the clock moves; an `AsyncFail` result ends in `undelivered` with its code; a restart resumes a `queued` message; a 100-message batch ends with correct counts. All pass under `-race`.

#### M1-F09 — SQLite store fixes

**Why.** `PRAGMA foreign_keys` is never enabled, so every `ON DELETE CASCADE` in the schema is ignored: deletes leave orphaned status events, verifications, attachments and blobs (and M1-16 retention will leak the same way). `getReadDB` always returns connection 0, so the read pool is unused. `:memory:` uses one shared database per process, so separate stores (and tests) share state. `store.Transaction` exists but bulk sends insert each message separately. [S-1, S-2, S-3, S-4]

**Where.** `store/sqlite/store.go`, `core/service.go` (`sendBatch`).

**Do.**
1. Add `_pragma=foreign_keys(1)` (and `busy_timeout`) to both the file and memory DSNs.
2. Use one write `*sql.DB` (max 1 connection) and one read `*sql.DB` with `SetMaxOpenConns(readPoolSize)`, instead of a slice that is never rotated.
3. Give each in-memory store a unique name: `file:mocksms-<ulid>?mode=memory&cache=shared`.
4. Wrap `CreateBatch` plus all `CreateMessage` calls in `store.Transaction` (spec: one transaction per batch).
5. Add `storetest` cases: deleting a message removes its status events and attachments; deleting a project removes everything under it; two in-memory stores do not see each other's data.

**Done when.** The new storetest cases pass for both the memory store and SQLite, and a 10,000-recipient batch inserts in under 3 seconds (engineering.md §12).

#### M1-F10 — Recipient handling

**Why.** The normalised number from `phone.Parse` is discarded (`_ = parsed`) and the raw input is stored, so `+233 24 123 4567` and `+233241234567` are different recipients and unsubscribe checks, `messages/wait?to=` and sim rules miss. `ReceiveInbound` rejects any inbound from an unsubscribed number before checking for START, so a number that sent STOP can never resubscribe. `CallbackURL` is always a non-nil pointer, so empty callback URLs get stored. Batches silently drop recipients that fail validation or simulation, and simulate every member as the first recipient. [K-1, K-2, K-4, K-5, K-6, K-13]

**Where.** `core/service.go`.

**Do.**
1. Store and match recipients in normalised E.164 (the value `phone.Parse` returns); apply the same normalisation to `to` filters in `ListMessages` callers, unsubscribe records and simulator matching.
2. In `ReceiveInbound`, handle STOP and START keywords first and always accept inbound messages; only outbound sends to an unsubscribed number fail with `unsubscribed`. Replace the hand-written lowercase and trim helpers with `strings.ToLower` and `strings.TrimSpace`.
3. Set `CallbackURL` only when the request provides one.
4. For batches: validate each recipient once, simulate each with a single-recipient request, and return rejected recipients instead of dropping them. Define the response shape in `openapi/openapi.yaml` first (e.g. `rejected: [{to, code, message}]` on the batch response), then implement.

**Done when.** Tests show: formatted and unformatted numbers resolve to one conversation; STOP then START re-enables sending; a message sent without a callback stores none; a batch with one invalid number returns it in `rejected` and delivers the rest.

#### M1-F11 — Simulator and OTP generation

**Why.** The simulator lives in `core` (architecture says `sim` implements `core.Simulator`). It shares one `math/rand.Rand` between concurrent requests outside its lock (data race). Global latency returns before the failure-rate check, so `failure_rate` is ignored whenever latency is set, and any `…99990X` number skips global settings even for X outside 1-5. Separately, `generateCode` builds OTPs from `time.Now().UnixNano()%10` with a 1ns sleep, bypassing the injected clock and producing predictable, often repeated digits. [K-3, K-12, K-14, R-2, R-3]

**Where.** `core/simulator.go` (move), `core/service.go:503-511`, `cmd/mocksms/main.go`.

**Do.**
1. Move the simulator to a new `sim` package implementing `core.Simulator`; wire it in `cmd/mocksms`; add the depguard rule for `sim` from M1-F01.
2. Use goroutine-safe randomness: `math/rand/v2` top-level functions, or an injected source guarded by a mutex so tests stay deterministic.
3. Combine effects: latency and random failure can both apply to one send. Treat only `…999901` to `…999905` as magic numbers.
4. Generate OTPs with `crypto/rand` (e.g. `rand.Int(rand.Reader, big.NewInt(10))` per digit); remove the sleep and `time.Now()`.

**Done when.** Concurrent `Evaluate` calls pass under `-race`; table tests cover every built-in magic number and the latency-plus-failure combination; 10,000 generated codes contain no fixed pattern and all have the requested length.

#### M1-F12 — Verification checks

**Why.** `CheckVerification` reads, increments and writes `attempts` without a transaction, so concurrent checks can exceed `max_attempts`. For the native API it returns 200 for expired and max-attempts cases, while spec §7.2 says `verification_not_found` (404) and `max_attempts` (429). [K-15]

**Where.** `core/service.go:316-358`, `api/handlers.go` (check handler).

**Do.**
1. Wrap the check in `store.Transaction`.
2. Native API: wrong code → 200 `{valid: false, status: "pending"}`; expired, approved or canceled → 404 `verification_not_found`; attempts exhausted → 429 `max_attempts`. Update `openapi.yaml` to match. (The Twilio adapter maps these its own way in M2-04.)

**Done when.** Tests cover each case above plus 20 concurrent wrong-code checks never exceeding `max_attempts`.

#### M1-F13 — API scoping

**Why.** `?project=` is accepted as a project ID on every route, writes included, and is never checked to exist, so any caller can write into any (even non-existent) project; spec §7.1 allows it only on read and test endpoints. `DELETE /messages` with no key silently wipes the `default` project instead of returning 400. The context key is a plain string with an unchecked type assertion. `api` imports `bus` directly, which the architecture forbids. `/healthz` hardcodes version `0.1.0`. [A-3, A-4, A-10, A-11, R-1]

**Where.** `api/handlers.go:104-134,572-582`, `api/sse.go`, `cmd/mocksms/main.go`.

**Do.**
1. In the auth middleware: resolve the Bearer key (return its error, do not overwrite it); accept `?project=` only on GET read and test endpoints, and return 404 if that project does not exist; on send endpoints with no key, use the `default` project.
2. `DELETE /messages` requires a key or `?project=`; otherwise 400 `validation_error`.
3. Use a private key type (`type projectKey struct{}`) and a helper that returns an error instead of panicking.
4. Depend on `core.Bus` in `api` (handlers and SSE hub); remove the `bus` import.
5. Pass the build version into `NewHandlers` and report it from `/healthz`.

**Done when.** Handler tests cover each rule, and `golangci-lint` shows no depguard violation for `api`.

#### M1-F14 — Long-lived requests and SSE

**Why.** `/events` is wrapped by the global `middleware.Timeout(60s)` and the server's 30s `WriteTimeout`, so live updates are cut after 30 seconds (the 30s heartbeat cannot prevent it). The SSE hub ignores the `?project=` filter, so every client gets every project's events. `messages/wait` returns old messages when `since` is omitted (spec: default to request time), parses `timeout` as `value+"s"` so `10s` silently becomes the default, has no 60s cap, returns code `internal` on timeout, returns the newest match rather than the first after `since`, and polls the database every 100ms. [A-5, A-6, A-7]

**Where.** `api/handlers.go:31-92,584-636`, `api/sse.go`, `cmd/mocksms/main.go:118-124`.

**Do.**
1. Apply `middleware.Timeout` only to ordinary routes; mount `/events` and `/messages/wait` outside it.
2. In the SSE handler clear the write deadline per connection: `http.NewResponseController(w).SetWriteDeadline(time.Time{})`. Send the heartbeat every 15s.
3. Register SSE clients with their project and broadcast each event only to that project's clients (plus clients that asked for all projects, in local mode). Emit `event: <type>` lines so `EventSource` listeners can filter by name.
4. `messages/wait`: default `since` to the request's arrival time; accept `timeout` as seconds or a Go duration, capped at 60s; on timeout return 408 with a new canonical code `wait_timeout`; return the oldest match after `since`; wake on the bus (`message.created` for that project and recipient) instead of polling.
5. `otp/latest` and `emails/latest` stay M2-10 work: either remove the 501 stubs or keep them and correct the CHANGELOG (see M1-F18).

**Done when.** An SSE client stays connected for 2 minutes and receives heartbeats; a client filtered to project A never receives project B's events; wait tests cover default `since`, `timeout=5`, `timeout=5s`, the cap, the 408 body and first-match ordering.

#### M1-F15 — Spec-first API

**Why.** Routes and handlers are hand-written; the generated `ServerInterface` in `api/openapi.go` is not used, and there is no `//go:generate` directive, so `task generate` regenerates nothing. Nothing stops the handlers drifting from `openapi.yaml`, which engineering.md §6 makes the published contract. [A-9, CI-9]

**Where.** `api/`, `go.mod`, `.github/workflows/ci.yml`.

**Do.**
1. Add oapi-codegen as a Go tool (`go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen`) and an `api/generate.go` with `//go:generate go tool oapi-codegen --config=oapi-codegen.yaml ../openapi/openapi.yaml`, configured for models, chi server and strict server.
2. Implement the generated strict interface and mount it with the generated chi handler; delete the manual route list. Endpoints missing from the spec are added to `openapi.yaml` first.
3. Add a CI step: `go generate ./... && git diff --exit-code` to fail on drift.

**Done when.** Removing a route from `openapi.yaml` and running `task generate` breaks the build, and CI fails if generated code is stale.

#### M1-F16 — SMS segment counting

**Why.** UCS-2 length is counted in runes rather than UTF-16 code units, so characters outside the Basic Multilingual Plane (emoji) are under-counted: 70 emoji are 140 units and 3 segments, reported as 1. Messages longer than 10 segments are silently capped at 10 instead of rejected, unlike real providers (Twilio error 21617). [P-1, P-2]

**Where.** `phone/encoding.go`, `core/service.go` (validation).

**Do.**
1. Count UCS-2 length as the sum of `utf16.RuneLen(r)` over the body, and never split a surrogate pair across segments.
2. Add form feed (`\f`) to the GSM-7 extended table.
3. Return the uncapped segment count; in send validation, reject bodies over 10 segments with `validation_error` on field `body` ("message too long").

**Done when.** Table tests cover 70 and 71 BMP characters, 35 and 36 emoji, extended-table characters at segment boundaries, and an 11-segment body rejected.

#### M1-F17 — Backfill tests

**Why.** `core`, `api`, `config` and `cmd` have no tests (core coverage 0% against an 85% target), so most defects above shipped unseen. Verification behaviour is tested from `store/sqlite`, and the in-memory reference store lives in `storetest/memstore_test.go`, a test file other packages cannot import. [R-5, R-6]

**Do.**
1. Move the in-memory store to `store/storetest/memstore.go` (non-test) so `core` and `api` tests can use it.
2. Add `core` tests (service, lifecycle, verification, inbound) with `FakeClock`; move the verification cases out of `store/sqlite/verification_test.go`.
3. Add `api` handler tests with `httptest` for every route's success and error paths.
4. Add `config` tests for flags > env > YAML > defaults precedence.
5. Run everything with `-race`; replace the remaining `time.Sleep` calls in tests with clock or channel synchronisation.

**Done when.** `go test -race -cover ./...` shows `core` ≥ 85% and no package without tests except `cmd/mocksms` (covered by M1-F05's shutdown test).

#### M1-F18 — Correct the docs

**Why.** Several records overstate what exists, and the product name is unsettled. [D-1 to D-5]

**Do.**
1. **Name:** the module and repo are `CommPit`, while the binary and every doc say `mocksms`. Decide which is the product name; if CommPit, rename across the docs, `PRODUCT.md` and `design.md`; if mocksms, note in `README.md` that `CommPit` is only the repository name.
2. **CHANGELOG:** correct the entries for embedded migrations (until M1-F06), the SMTP server (not yet), graceful shutdown (until M1-F05), the test helpers (`otp/latest`, `emails/latest` are stubs) and the simulator's built-in rules (async rules have no effect until M1-F08).
3. **progress.md:** remove the resolved "module path" blocker, and note that M1-06, M1-09, M1-12 and M1-13 have open fixes listed here.
4. **techstack.md:** record the chosen package manager (M1-F03) and the Go version (M1-F01).

**Done when.** Every CHANGELOG line describes behaviour that exists in the code at that commit.

**M1 milestone gate** (in order, once every task above is `[x]`):
- [x] Full suite passes on `milestone/m1`: `task test`, `task lint`, `task e2e`, `task build`
- [x] Final commit (changelog `v0.1.0` heading, progress update) pushed to `milestone/m1`
- [x] PR `milestone/m1` → `main` merged with a merge commit
- [x] `v0.1.0` tagged on `main`; next milestone branch created from `main`

## M2: Twilio, Termii, test API, inspector → v0.2.0

**Branch:** `milestone/m2`. Commit and push every M2 task here.

| | ID | Task | Depends on | REQ |
|---|---|---|---|---|
| [x] | M2-01 | `adapterkit`: panic recovery via `WriteError`, credential → project resolution, request logging with masking (64 KB cap) | M1 | REQ-090 |
| [x] | M2-02 | RequestLog storage and `GET /api/v1/requests[/{id}]` | M2-01 | REQ-090 |
| [x] | M2-03 | Twilio Messages API: create, list (filters, paging), fetch; Twilio response shape and error mapping; stores `StatusCallback` | M2-01 | REQ-030, REQ-032 |
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
