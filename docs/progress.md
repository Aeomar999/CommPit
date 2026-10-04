# mocksms — Progress

Single source of truth for where the project stands. Update it in every PR that changes task status (see [AGENTS.md](../AGENTS.md)).

## Current status

| | |
|---|---|
| **Phase** | M1 in progress |
| **Current milestone** | M1 |
| **Next action** | M1-11: `config`: koanf with flags > env > YAML > defaults; all settings from spec §9 |
| **Last updated** | 2026-10-04 |

## Milestones

| Milestone | Release | Status | Tasks done |
|---|---|---|---|
| M1: Core, native API, SMTP, inbox | v0.1.0 | In progress | 10 / 21 |
| M2: Twilio, Termii, test API, inspector | v0.2.0 | Not started | 0 / 14 |
| M3: Webhooks, failure simulation, inbound, batches | v0.3.0 | Not started | 0 / 12 |
| M4: Estimate, MCP, CI kit | v0.4.0 | Not started | 0 / 8 |

## Blockers and open items

| Item | Blocks | Owner |
|---|---|---|
| Final name and GitHub organization (Go module path) | M1-01 | Project owner |
| Access to a real Termii account | X-01; Termii marked stable | Project owner |
| Legal check on provider names | Public announcement at M2 | Project owner |

## Session log

Newest first. One entry per working session: what changed, decisions made, what's next.

### 2026-10-04

- Completed M1-10: Verification flow - StartVerification (SMS/Email, code length, TTL, max attempts, serviceRef, fixed code), CheckVerification (correct/wrong code, attempts tracking, max attempts, expiry, already approved), tests in sqlite store; fixed default sender for verification messages
- **Next:** M1-11: `config`: koanf with flags > env > YAML > defaults; all settings from spec §9

### 2026-10-04

- Completed M1-09: Lifecycle runner - uses injected Clock for testability, Schedule/advance with configurable step delay, ResumeQueuedAndSent on startup, Stop() cancels all timers; service exposes LifecycleRunner() and Clock() getters
- **Next:** M1-10: Verification flow: start, check, attempts, expiry, `otp.fixed_code`

### 2026-10-04

- Completed M1-08: Core service integration with auto-create projects from credentials, SendMessage/SendBatch with validation, extraction on save, ProjectResolver, Simulator with built-in rules (Twilio test numbers + 99990X patterns), configurable latency/failure rate, config package with koanf (flags > env > YAML > defaults), cmd/mocksms wiring all components (store, bus, simulator, clock, resolver)
- **Next:** M1-09: Lifecycle runner: Clock-driven timers, configurable step delay, resume on startup

### 2026-10-04

- Completed M1-07: `bus` package - in-process pub/sub implementing `core.Bus` with Publish/Subscribe, event types (message.created, message.status, verification.updated, batch.updated, request.logged, webhook.delivered), concurrent-safe, unsubscribe support
- **Next:** M1-08: Core service integration with store, bus, lifecycle runner, simulator, clock, resolver

### 2026-10-03

- Completed M1-06: `store/sqlite` with WAL mode, single writer + read pool (configurable size), goose migrations (embedded SQL), blob storage, transactional batch insert support; passes `storetest` conformance suite
- **Next:** M1-07: `bus`: in-process pub/sub implementing `core.Bus`

### 2026-10-03

- Completed M1-05: `store/storetest` conformance suite - comprehensive test suite covering all Store interface methods (Project, Credential, Message, StatusEvent, Batch, Verification, Unsubscribe, Attachment, WebhookDelivery, RequestLog, Transaction) with in-memory implementation for verification
- **Next:** M1-06: `store/sqlite` - WAL, single writer + read pool, goose migrations, blobs, transactional batch insert; passes storetest

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
