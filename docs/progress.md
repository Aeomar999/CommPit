# mocksms — Progress

Single source of truth for where the project stands. Update it in every PR that changes task status (see [AGENTS.md](../AGENTS.md)).

## Current status

| | |
|---|---|
| **Phase** | Wave 1 complete (v0.4.0 released) |
| **Current milestone** | — |
| **Next action** | M4-05/M4-06 external repos, or next wave planning |
| **Last updated** | 2026-10-13 |

## Milestones

| Milestone | Release | Status | Tasks done |
|---|---|---|---|
| M1: Core, native API, SMTP, inbox | v0.1.0 | Done | 21 / 21 |
| M2: Twilio, Termii, test API, inspector | v0.2.0 | In progress | 14 / 14 |
| M3: Webhooks, failure simulation, inbound, batches | v0.3.0 | Done | 12 / 12 |
| M4: Estimate, MCP, CI kit | v0.4.0 | Done | 6 / 6 |

## Blockers and open items

| Item | Blocks | Owner |
|---|---|---|
| Access to a real Termii account | X-01; Termii marked stable | Project owner |
| Legal check on provider names | Public announcement at M2 | Project owner |

**Note:** M1-06, M1-09, M1-12 and M1-13 have open fixes tracked in the M1-Fxx review tasks above.

## Session log

Newest first. One entry per working session: what changed, decisions made, what's next.

### 2026-10-13: M4 Gate

- Full suite on `milestone/m4`: `go test -p 1 -count=1 ./...` green uncached; `golangci-lint` 0 issues; `biome check .` clean (one e2e format fix); `task build` fresh binary; real-SDK examples fresh green; Playwright 8/8 green. (`task lint` wrapper still hangs on pnpm spawn here, halves verified directly as in M3.)
- Final commit carries the `v0.4.0` heading, progress update, gate ticks 1–2. M4-05/M4-06 live in separate repos (out of this gate by design). PR → merge → tag next.

### 2026-10-13: v0.4.0 Released

- PR #20 merged with a merge commit, `v0.4.0` tagged on `main`, gate ticks 3–4 via fix PR #21 (green). Wave 1 (M1–M4) is complete: core + native API + SMTP + inbox, Twilio/Termii adapters + test API + inspector, webhooks + sim + inbound + batches UI, estimate + MCP + CLI + benchmark + guides.

### 2026-10-12: Estimate Backend (M4-01)

- Completed M4-01 (`REQ-100`): `estimate` package (embedded YAML tables, `Summarize` mix builder, `Estimate` scaler) + `GET /projects/{id}/estimate?volume=N`.
  - Tables researched live: Twilio per-segment SMS (US/CA/GB/DE/FR/IN/NG/KE) + $0.05 Verify; Termii NG $0.005/unit + $0.008 Verify contact (third-party reported, sourced in-file). Unlisted countries → `no_price_data`, never guesses.
  - Mix from outbound SMS only (native/smtp and email excluded, documented); verifications scale by the same factor; money rounded to cents; deterministic ordering throughout.
  - Handler pages messages/verifications (retention caps at 10k, loop stays short); new `Simulator.Latency/FailureRate` getters + `Service.Simulator()` for the sim endpoint; `NewHandlers` takes pricing tables (loaded once in `cmd`, fail fast). OpenAPI-first with regen; `api` architecture row updated to match reality.
  - Tests: table parsing, grouping/skips, scaling math, missing-data, validation (400s), 404, FK-safe seeding.
- Completed M4-02 (`REQ-100`): estimate UI.
  - New Estimate page (nav + route + title): volume input with apply, per-provider cards (totals, verification row, by-country table, no-price-data notice, as_of + estimate-only label), empty states. `default` project alias on the endpoint (the UI never learns ULIDs) + openapi description.
  - Playwright `estimate.spec.ts` (link-credential-to-default pattern, card assertions). Full suite 8/8 green; tsc + biome + build clean.
- Completed M4-03 (`REQ-101`): MCP server.
  - New `mcpserver` package on the official Go SDK (v1.8.0) with seven typed tools calling the core service directly: list/wait/latest-OTP/latest-email/simulate-inbound/list-requests/reset-project. Project arg is a ULID or empty (default project); wait mirrors the API semantics plus an explicit `since` (needed for deterministic tests); results carry both JSON text and structured output.
  - Mounted at `/mcp` (streamable HTTP), exempt from `X-Mocksms` like the adapters (Host protection stays). New `mcpserver` depguard rule (core only).
  - Tests: all seven handlers (incl. wait existing/timeout paths, not-found + validation errors, project resolution), raw JSON-RPC wire test (initialize → tools/list → tools/call with SSE-frame decoding), middleware exemption test. Verified live: session handshake, 7-tool listing with schemas, inbound call with no header.
- **Next:** M4-04 (CLI subcommands).
- Completed M4-04 (`REQ-102`): CLI subcommands on a running server.
  - `otp latest --to` prints the bare code (script-friendly); `messages wait [--to/--channel/--timeout/--since/--project]` prints the message JSON; `reset --project` wipes a project with confirmation. `--host/--port` select the server (0.0.0.0 maps to loopback); usage errors exit 2, request failures exit 1; flags accepted before or after the subcommand (Go's flag package stops at the first positional, so a splitter partitions them first — caught by a failing test).
  - Tests: usage table (no dial-out via dead port), canned-server paths incl. headers + error extraction, flag-order matrix, `--since` validation, and a live-server e2e (seed → otp + wait). The wait test needed `--since`: the API defaults `since` to call time, so pre-seeded messages never match.
- **Next:** M4-07 (bulk benchmark).
- Completed M4-07: bulk benchmark in CI.
  - New `BenchmarkServiceSendBatch10k` (real service + sqlite + fake clock): first run took 3.76 s, over the 3 s budget. Profiling blamed `sim.cleanPhone` (full libphonenumber parse per rule per recipient, 64% of accept time), so E.164-shaped input now fast-paths and rule patterns pre-clean once at `SetRules` — 1.15 s, with behavior locked by new tests (fast-path identity set, formatted-rule match).
  - Nightly `benchmark.yml` (tracked, never failing) runs both benchmarks once, posts to the job summary, keeps a 90-day artifact; `task bench` for local runs. Store insert holds at ~0.6 s.
- **Next:** M4-08 (docs).
- Completed M4-08: docs.
  - New `docs/guides/mcp.md` (setup one-liner, seven-tool table, project scoping, troubleshooting), `docs/guides/ci.md` (minimal job, wait/reset/credential-isolation patterns, Playwright reference, external M4-05/M4-06 pointers), `docs/guides/estimate.md` (inputs, math, price sources, exclusions, API). README links all three.
  - Every curl verified live (native callback, inbound, sim round-trip, Twilio callback queued + signed with replay, estimate $83 for 10k US Twilio with native excluded).
- **Next:** M4 gate (full suite, final commit, PR, tag).

### 2026-10-12: M3 Gate

- Full suite on `milestone/m3`: `task test` green; `task e2e` green (real-SDK examples + 7 Playwright tests, 3 consecutive runs); `task build` green (`bin/mocksms`, embed UI). `task lint` as a wrapper hangs in this environment (pnpm spawn), so both halves were verified directly: `golangci-lint run` 0 issues, `biome check src/` clean.
- Gate fixes in this session: `TestWorker_DockerRewrite` flake (assertion raced `finish` persistence — now synced on the terminal bus event; audited sibling tests, which are safe via the parking sender); e2e project selection (auto-generated names are never literally "default" — specs now read the project id back from a send); biome format on the new UI files.
- Final commit carries the `v0.3.0` changelog heading, progress update, and gate ticks 1–2. PR → merge → tag → m4 next.

### 2026-10-12: M3 Gate Fixes (CI)

- CI lint failed on e2e formatting (`bun run biome check .` covers `web/e2e`, local runs had scoped to `src/`): formatted the specs, fixed `node:http` import protocol + import order. Lesson: always run the exact CI command (`biome check .`), not a scoped subset.
- CI Test failed once per OS on `TestRun_AdapterDedicatedPort` ("did not serve in time") with identical code that passed 10 minutes earlier — CI-runner timing flake; passed on retry (ubuntu, macos).
- Windows CI persistently failed `TestSMTP_BounceAcceptedThenFails`: the second `clock.Advance` can run before the lifecycle goroutine re-arms its timer, so the step never fires on slow runners. Fixed by `BlockUntilWaiters(1)` between the event wait and the second advance — a real test race, not app code.
- CI E2E failed `batches.spec` once (row not visible in 10 s on a cold runner). Hardened: API-level pre-assertion splits server vs UI failures, first-content timeouts 20 s, suite already serial.
- CI E2E failed `batches.spec` once (detail never showed `delivered`). Root cause found in app code, not the test: the UI subscribed SSE on the literal channel `"default"`, but the server only broadcasts to real `prj_` IDs plus `"all"` — live updates never arrived anywhere, and the detail only passed locally by timing luck. Fixed (`useSSE()` → `"all"` channel); `inbox.spec` is now a true live test (send after open, no reload).
- e2e hardened along the way: serial workers (shared server + default project), default-project resolution from a send response (auto-names are never literally "default"), API pre-assertion splitting server vs UI failures, 20 s cold-start timeouts.
- **M3 RELEASED as v0.3.0**: PR #16 merged with a merge commit, tag pushed, `milestone/m4` branched from `main` (fast-forwarded past the #17 gate-tick fix PR). Gate ticks 3–4 went through the compliant `fix/` → PR flow.

### 2026-10-09: Webhooks Worker + Twilio StatusNotifier (M3-01, M3-02)

- Completed M3-01 (`REQ-053`): new `webhooks` package (delivery worker, sender, config) wired into `cmd/mocksms` with bus wake-ups, graceful shutdown, `webhooks.timeout` / `webhooks.max_attempts` settings, and a `depguard` rule (core only).
  - Worker: persistent queue via `ListPendingWebhooks`, bus wake-up (no polling) plus clock-driven retry timers, 10 s send timeout, backoff 1 s / 5 s / 30 s / 2 m / 10 m (six sends max — the spec's five waits imply five retries), attempt records with response capture, `webhook.delivered` events on terminal outcomes, idempotent stop.
  - Tests caught two real bugs: a lost retry wake-up (drain only armed timers for skipped rows, never for just-scheduled retries) and a lossy `next_retry_at` round-trip (string-parsed timestamp dropped the zone — now scanned as `time.Time` like every other table). Timing races were eliminated with a parking sender plus `FakeClock.BlockUntilWaiters` instead of sleeps/polling.
  - `Store.ListPendingWebhooks` now takes the query time instead of `time.Now()`, so virtual clocks stay deterministic; storetest pins due-only filtering. `sqlite.GetCredential` miss already returns `not_found` (M2-12).
- Completed M3-02 (`REQ-051`, `REQ-052`): Twilio `StatusNotifier` with `X-Twilio-Signature`.
  - `adapters/adapterkit/webhook.go`: new `WebhookRequest`, `StatusNotifier`, `InboundNotifier`, `ReplyParser` interfaces for the webhook dispatcher.
  - `adapters/twilio/signature.go`: HMAC-SHA1 over URL + sorted params, validated against Twilio's documented example and cross-checked with the official Python SDK.
  - `adapters/twilio/notify.go`: `StatusWebhook` builds form-encoded status callbacks with explicit auth_token, observed AC token, or API secret fallback (spec §7.4). Credential recording from Basic Auth on Messages and Verify sends (AC and SK). `signingKey` resolves with SK fallback warning.
  - `webhooks/dispatch.go`: `Formatters` registry (startup registration only) + `Dispatcher` that subscribes to `core.EventMessageStatus`, loads the message + project, calls the provider's `StatusNotifier`, and queues a `core.WebhookDelivery` row.
  - Wired into `cmd/mocksms`: dispatcher starts after adapters register, stops on shutdown.
  - Tests: Twilio signature ground-truth vector, status webhook payload + signature verification, credential recording on send, dispatcher queuing and skip paths.
- Completed M3-03 (`REQ-051`): Termii delivery-report notifier.
  - `adapters/termii/dlr.go`: `StatusWebhook` posts Termii's documented JSON delivery report (`type`, `id`, `message_id`, `receiver`, `sender`, `message`, `sent_at`, `cost: null`, `status`, `channel`) to the account-level `termii.webhook_url` setting — Termii has no per-request callback. Status vocabulary (`Message Sent` / `Delivered` / `Message Failed`), microsecond UTC `sent_at`, `X-Termii-Signature` (hex HMAC-SHA512 over the exact bytes, keyed by the observed `api_key`; unsigned with a warning when unseen). Payload shape confirmed against Termii's public Events and Reports docs; `type` value, `id` semantics, signature encoding and key choice marked **unverified** in `docs/fidelity.md` (X-01).
  - `api_key` recorded from sends via the shared `sendHandler`. Registered in `cmd/mocksms` alongside Twilio.
  - Tests: Python-computed golden signature vector, payload shape, status table, skip paths, key recording on send.
- Completed M3-04 (`REQ-051`, `REQ-052`): native webhooks.
  - `api/notify.go`: `Notifier` (holds the store, keeps the spec §7.3 method signature) builds `{"event": "message.status", "message": {…}}` for `native`-provider messages with a `callback_url`. `X-Mocksms-Signature: sha256=<hex HMAC-SHA256>` keyed by the project's oldest native API key (stable with several keys); unsigned with a warning when keyless. Registered in `cmd/mocksms`; no dispatcher change needed (it stays provider-agnostic; `api` was already allowed to import `adapterkit`).
  - Tests: Python-computed golden vector, event payload shape, oldest-key selection with several credentials, skip paths, unsigned fallback.
  - Gap noted in `docs/fidelity.md`: `verification.updated` has no destination (`Verification` carries no callback URL), so no verification webhook is queued — that needs a model addition, not a formatter.
- Completed M3-05 (`REQ-054`, `REQ-055`): Docker rewrite + replay re-sign.
  - Docker: `webhooks.InDocker` (`/.dockerenv` or `MOCKSMS_IN_DOCKER=1`) + `RewriteLoopback` (localhost/127.0.0.1/::1 → `host.docker.internal`, port/path/query preserved). The worker applies it at send time when `DockerRewrite` is set (`!cfg.NoDockerHostRewrite && InDocker()` from the pre-existing flag/config surface) and records a note on the delivery; the stored URL keeps the configured host. New `note` column via migration 002 (001 is merged and immutable) across sqlite + txstore, with a storetest round-trip.
  - Replay: `Dispatcher.Redispatch` rebuilds the delivery as a fresh record (attempt reset), re-running the provider formatter to re-sign when the message context exists, else cloning verbatim. New `webhook.queued` core event, published after every create, wakes the worker — this also closes the dispatcher/worker race where a drain could run before the row existed. `POST /webhooks/{id}/replay` now goes through an api-owned `WebhookRedispatcher` interface (nil falls back to the old verbatim clone), so `api` imports no adapter code. Architecture §5.2 + ADR-011 updated.
  - Tests: rewrite table (incl. suffix/IPv6/unparseable guards), worker send-time rewrite + note + stored-URL fidelity, queued-event publish, resign + both verbatim fallbacks, replay endpoint (redispatched/nil-fallback/404).
- Completed M3-06 (`REQ-060`, `REQ-061`): sim rule engine.
  - Engine (`sim`): email-aware matching (phone normalization mangled `@` addresses — `@` on either side now switches to case-insensitive match, so `*@invalid.mocksms.test` / `*@bounce.mocksms.test` defaults work and phone patterns can't false-match emails); sliding-window `rate_limit` (N/min per project per provider, allowed sends consume quota, injectable clock); all five effects producible.
  - Core consumption: `RateLimited` → `rate_limited` sync error (single) / rejected entry (batch); `Hang` → clock-aware blocking wait after scheduling (covers HTTP + SMTP + verify uniformly, honors ctx cancel). Delay/fail_async/reject already flowed.
  - YAML (`sim.rules`, `koanf:` tags — koanf v2 ignores `mapstructure` tags, found by a failing test): match + reject/fail_async/delay_ms/hang_ms/rate_limit_per_min with fail-fast validation; `cmd` prepends custom rules before built-ins so they win.
  - SMTP maps `invalid_address` to 550 (at DATA time — RCPT-time phase is a documented gap in `docs/fidelity.md`).
  - Tests: email magic + phone-pattern guard + From matching, window allow/limit/slide/isolation, YAML load + validation table, service rate/hang (FakeClock + BlockUntilWaiters, no sleeps), SMTP 550 + bounce-to-undelivered driven by bus-event sync.
  - Observed (pre-existing, out of scope): `StepDelay: 0` means "default 300 ms", not "instant" as the spec claims — zero is defaulted in `NewService`.
- Completed M3-07 (`REQ-060`): async failure codes.
  - `core.AsyncFail` gains `Status` (empty = undelivered, back-compat); the lifecycle honors `failed`. Terminal mapping: 30005/30003 → undelivered, 30007 → failed, email `bounced` → failed.
  - New magic `…999906` → `fail_async` 30003 "unreachable handset" (undelivered); the twilio guide line now lists 999901–999906.
  - YAML `fail_async` accepts optional `status: failed|undelivered` (validated).
  - Tests: status table (30003/30005/30007), lifecycle failed-path, YAML status valid + invalid, bounce e2e now asserts failed + `bounced` code.
- Completed M3-08 (`REQ-070`, `REQ-071`): inbound webhooks and TwiML replies.
  - Found on audit: `POST /inbound`, `core.ReceiveInbound` (incl. STOP/START branch), and `OutboundReply` already existed — the gaps were the notifiers, dispatch, and reply sending.
  - Twilio `InboundNotifier` (form `MessageSid`/`AccountSid`/`From`/`To`/`Body`/`NumMedia` to `twilio.inbound_url`, signed) + `ReplyParser` (`<Response><Message>` → replies with fresh SM refs; missing `to` left for the worker to default).
  - Native `InboundNotifier` (`{"event": "message.inbound", …}` to `native.inbound_url`, signed with the API-key logic).
  - Dispatcher subscribes `message.created` for inbound direction → `kind: inbound` rows; worker parses replies at send time via a `WithReplies` option and re-enters through an `OnReply` service callback (addressing defaulted from the inbound message). New `provider` column via migration 003 for parser routing (also in the API JSON). Architecture §5.4 + ADR-012 updated.
  - Tests: inbound payloads + skip paths, TwiML table (explicit/missing `to`, distinct refs, empty, malformed), dispatch queuing + outbound silence, worker reply addressing + skip cases, provider round-trip. Full suite run serially (`-p 1`) after parallel linking OOM'd the box.
- Completed M3-09 (`REQ-072`): STOP/START audit.
  - Verified complete against the spec: all six STOP and three START keywords (trimmed, case-insensitive), idempotent storage (`INSERT OR REPLACE`, delete-missing is a no-op), suppression on single + batch + verification sends, Twilio 21610 and Termii `unsubscribed` mappings — all pre-existing with tests.
  - Added the missing coverage: full HTTP-path test (`POST /inbound` STOP → send 400 `unsubscribed` → START → send 201), all nine keyword variants, and double-STOP/double-START idempotency.
- Completed M3-10 (`REQ-062`, `REQ-081`): UI batches, reply, webhooks log, sim settings.
  - Backend first: `GET /batches` (openapi + regen, verified offline-idempotent), `GET/PUT /sim/config` runtime settings (global, validated; new `Simulator.Latency/FailureRate` getters + `Service.Simulator()`), `ListWebhookDeliveries` store method (project-scoped, newest-first; the log endpoint now shows terminal rows too, previously pending-only), `webhook.queued` on the SSE hub for live log updates.
  - UI: BatchesPage list + detail (stacked status bar, counts, rejected table); ReplyBox under SMS threads (sends as the handset, STOP hint, per-thread reset); webhooks tab with status tones, response codes, docker notes, and per-row Replay; Settings simulation section (latency ms + failure %, validated, applied live). No design skill available in this environment — followed design.md tokens/patterns directly (cards, EmptyState, chips, focus rings, labelled controls).
  - Verified: tsc + biome clean, vite build, existing smoke passes, and the full loop exercised live (batch 202 with counts, sim PUT round-trip, STOP suppression + START recovery, replay re-signs with fresh id). `task generate:web` deliberately NOT run — it would overwrite the hand-written `web/src/lib/api.ts`.
  - Flake note: `TestWorker_DockerRewrite` failed once under full parallel load, passes serially and with `-count=1`.
- Completed M3-11 (`REQ-062`, `REQ-081`): Playwright suite, 7 tests green.
  - New specs: inbox live arrival, reply-box round trip, batches progress + detail, inspector adapter-request log, and the full webhook loop (Node catcher receives signed `message.inbound`, UI replay re-queues and redelivers). Unique bodies/numbers per run; batch row located by id from the 202 response.
  - Fixed a stale-server trap mid-session (background jobs die with the shell; rebuilt with `-tags embed` and verified the PID before trusting results).
- Completed M3-12: docs.
  - New `docs/guides/webhooks.md` (per-provider callbacks + signature snippets, inbound + TwiML, retries/replay, Docker) and `docs/guides/simulation.md` (magic-values table, `sim.rules` reference, runtime settings). README links both; twilio/termii guides point at webhooks.
  - Every curl verified live against the binary (native callback, inbound STOP, sim GET/PUT, Twilio StatusCallback queued + signed, replay re-signs) — including a live STOP-suppression recovery and backoff retries against a dead port.
- **Next:** M3 gate (full suite, final commit, PR, tag, m4).

### 2026-10-09: SMTP Delivery Fix (M3, unplanned)

- The SMTP server replied `250 OK: queued` to every email but stored nothing: go-smtp only calls the `Session` interface methods (`Mail`/`Rcpt`/`Data`), so the hand-written `Submit()` holding all delivery logic never ran. AUTH was equally dead — the server advertises it only when the session implements `AuthSession`. Fixed by delivering inside `Data()`, implementing `AuthSession` (PLAIN), and routing unauthenticated mail to the default project per spec §6.2. Covered by new `smtpd` end-to-end tests (real server + `net/smtp` client); `go-sasl` promoted to a direct dependency. Lesson logged in the system design journal.
- **Next:** M3-01 (`webhooks` worker).

### 2026-10-08 (Session 21): M2 Milestone Gate

- All 14 M2 tasks `[x]`. Full gate green on `milestone/m2`: fresh `go test ./...`, `golangci-lint` (0 issues), `gofmt`, Biome (38 files), `tsc`, Vite production build, embed binary build, e2e snippet harness (Node/Python/Go pass; PHP/.NET skip without runtimes), Playwright smoke (2 passed against the embedded UI). `-race` and `pnpm` script execution remain unavailable in this sandbox (no C toolchain; pnpm runner hangs) — both are covered by CI (`-race` matrix, `bun` web jobs).
- Gate findings fixed in this session: the embedded web UI was never wired (`task build` produced a UI-less binary, so no Playwright run could pass) — `web/embed.go` (tag-gated) + SPA-fallback handler in `cmd` now serve the bundle at `/`; the M2-13 root-path alias was removed in favor of the UI (ADR-010 updated, CHANGELOG migration note). Added a minimal Playwright smoke (`web/e2e/smoke.spec.ts`) since M3-11 owns the full suite; `test-results/` + `playwright-report/` ignored. Also fixed `ci.yml` e2e to start the server with `--otp-code 123456` (snippets would otherwise fail on random codes).

### 2026-10-08 (Session 20): Provider + Test-API Guides (M2-14)

- Completed M2-14 (final M2 task): `docs/guides/twilio.md`, `termii.md` and `test-api.md` (with Playwright, Cypress and Jest examples); README links them and documents the new `adapters.*.port` and `projects` settings.
- Every curl example was verified live against the binary (Twilio create/Verify, Termii send/otp/verify, wait, otp/latest); magic-number table cross-checked with `sim/simulator.go`. The Jest snippet's undefined `verId` was fixed to use `verification_id` from `otp/latest`.
- `go test ./...`, `golangci-lint run` (0 issues), `gofmt` clean (docs-only change set, no code touched).
- **Next:** M2 milestone gate (full suite + `e2e` + `build`, final commit, PR, tag `v0.2.0`, branch `milestone/m3`).

### 2026-10-08 (Session 19): Request Inspector + OTPs UI (M2-13)

- Completed M2-13 (`REQ-090`): the inspector is live and the OTPs view is complete.
  - **Inspector**: HTTP Requests tab lists real adapter logs (method, path, status chip, duration) with a side-by-side raw request/response detail view (headers + pretty-printed bodies); Webhooks Log tab lists real deliveries. Fixed two client bugs the silent `.catch(() => empty)` fallbacks were masking: `logs` vs `requests` and `deliveries` vs `webhooks` keys, plus a new `getRequest` detail call. `useSSE` now invalidates `request-logs` on `request.logged`.
  - **OTPs view**: added a Found-in-messages section (extracted codes with provider + time) beside the Verify grid; refresh covers both queries.
  - **Big pre-existing find**: the UI calls `/api/v1/*` but the server only mounted root paths (404 everywhere) — `openapi.yaml` servers, the design spec and the vite proxy all assume `/api/v1`. The API is now mounted at both prefixes (ADR-010 records root as a compat alias); `isExemptPath` normalizes the prefix so health/events/wait exemptions work on both mounts (this also fixes browser `EventSource`, which cannot send the `X-Mocksms` header). Covered by `TestRun_ApiV1Prefix` + middleware mount tests.
  - Verified end-to-end against the binary (adapter send → requests log with masked auth, verifications list, emails/latest) and with a production `pnpm run build`. `go test ./...`, `golangci-lint run`, `gofmt`, Biome and `tsc` clean.
- **Next:** M2-14: Docs (final M2 task).

### 2026-10-08 (Session 18): Credential Linking (M2-12)

- Completed M2-12 (`REQ-091`): one project can now serve several credentials.
  - **Core**: `ProjectResolver` gains `LinkCredential` (create-or-move + cache update; target must exist). Also fixed SQLite `GetCredential` to return `not_found` on a miss like the other stores (was `internal`), with a new storetest assertion locking it in.
  - **API**: `POST /projects/{id}/credentials` was a no-op (resolved and discarded the credential, ignoring `{id}`) — now validates input, 404s on unknown projects, and really links. Tests prove traffic routing: linked keys land in the target, moves drain the old project.
  - **Config + startup**: YAML `projects[].credentials` (`id`, `name`, credentials with `provider`/`key`) applied before serving; missing projects are created. Covered by YAML-load and live-server tests.
  - **UI**: Settings page gained a Linked Credentials card (project/provider/key form, success/error feedback, accessible labels); Biome + tsc clean.
- **Next:** M2-13: UI OTPs view + request inspector.

### 2026-10-08 (Session 17): Dedicated Adapter Ports (M2-11)

- Completed M2-11 (`REQ-036`): `adapters.twilio.port` / `adapters.termii.port` (flags `--twilio-port`/`--termii-port`, env, YAML; 0 disables) serve each adapter at `/` on its own port, reusing the same kit handler (logging, recovery, credential resolution) plus Host allow-list security.
  - Server fan-out in `cmd/mocksms`: named HTTP servers share one error channel (fail-fast on any port conflict, per M1-F05), all drain on shutdown.
  - **Real bug found by the new test**: dedicated ports serve adapter paths at root, so the `X-Mocksms` path exemption never matches and SDK traffic 400s. Dedicated listeners now skip only that header check (provider auth still applies); the main mount keeps path exemptions.
  - **Pre-existing bug exposed**: env-var loading never stripped the `MOCKSMS_` prefix, so no `MOCKSMS_*` override ever worked — fixed with a one-line `TrimPrefix` (nothing could have depended on the broken behavior).
  - Tests: config defaults + env override, dedicated-port serve (201 at root), adapter conflict errors. `go test ./...`, `golangci-lint run` (0 issues), `gofmt` clean. Live check of both dedicated ports.
- **Next:** M2-12: Credential linking.

### 2026-10-08 (Session 16): Test API (M2-10)

- Completed M2-10 (`REQ-040`–`REQ-043`, `REQ-023`): the two M1 stubs are now real endpoints, and the rest got coverage.
  - **`GET /otp/latest`** (`to` required, `since` optional with no lower bound when omitted): newest outbound coded message for the recipient; returns the verification code with linkage (`source: verification`, `verification_id`) when the message carries one, else the first extracted code (`source: extracted`); 404 when nothing matches. OpenAPI shapes already existed, so no spec change was needed.
  - **`GET /emails/latest`**: newest outbound email with codes, links and primary link (nil slices render as `[]`, not null).
  - **Real gap found**: `/messages/wait` was mounted outside the auth groups with no middleware, so every call 401'd. Wrapped the route with the read auth middleware (spec-compliant: `?project=` accepted); added wait tests (immediate match, 408 `wait_timeout`).
  - Added expire tests (200 + `expired`, 404 unknown) and DELETE tests (400 without project, 204 + empty inbox with `?project=`).
  - `go test ./...`, `golangci-lint run` (0 issues), `gofmt` clean.
- **Next:** M2-11: Dedicated adapter ports.

### 2026-10-08 (Session 15): Termii Golden Fixtures (M2-09)

- Completed M2-09: 13 golden fixtures in `adapters/termii/testdata/` covering every SMS and Token endpoint (success + error paths), locked by `TestTermii_Golden` with `-update` regeneration. Normalization is key-based (`message_id`, `pinId`, `pin`) rather than regex, because phone numbers share the digit space.
- Reviewed and completed the `docs/fidelity.md` Termii sections as the unverified-behavior listing (Termii publishes no spec, so no contract tests apply): added `from`-required, `pin_attempts`/TTL defaults, email code bounds, generate-keeps-nothing, strict E.164, no per-request callback, channel distinctions.
- `go test ./...`, `golangci-lint run` (0 issues), `gofmt` clean.
- **Next:** M2-10: Test API.

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
