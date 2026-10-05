# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Before 1.0.0, minor versions may include breaking changes to the native API; they are called out under **Changed** with a migration note.

Provider-compatible adapters follow the real providers' behavior. Fixes that make an adapter match its provider more closely are listed under **Fixed** with the `fidelity:` prefix.

## [Unreleased]

### Added

- SSE hub: `SSEHub` subscribes to `bus.EventBus` events (message.created, message.status, verification.updated, batch.updated, request.logged, webhook.delivered), fans out to HTTP clients via `GET /api/v1/events` with project filter, heartbeat keepalive (30s), connection event
- `api` package: OpenAPI 3.1 spec (`openapi/openapi.yaml`) with all M1 endpoints (SMS, email, verifications, messages, batches, attachments, webhooks, projects, requests, test helpers, events); oapi-codegen generating types and chi-server; `api` handlers implementing ServerInterface with Bearer auth, project-scoped access, send/receive endpoints, project management, test helpers (messages/wait, otp/latest, emails/latest), webhook replay
- `config` package: configuration loading with koanf (flags > env > YAML > defaults), all settings from spec §9
- `cmd/mocksms`: composition root wiring Store, BlobStore, Bus, Simulator, Clock, ProjectResolver, Service; HTTP + SMTP servers with graceful shutdown
- Core service integration: auto-create projects from credentials, `SendMessage`/`SendBatch` with validation, extraction on save, `ProjectResolver`, `Simulator` with built-in rules (Twilio test numbers + 99990X patterns), configurable latency/failure rate
- `bus` package: in-process pub/sub implementing `core.Bus` with Publish/Subscribe, event types (message.created, message.status, verification.updated, batch.updated, request.logged, webhook.delivered), concurrent-safe, unsubscribe support
- `store/sqlite`: SQLite implementation of `core.Store` and `core.BlobStore` with WAL mode, single writer + configurable read pool, goose migrations (embedded), blob storage, transactional batch insert; passes `store/storetest` conformance suite
- `store/storetest` conformance suite: comprehensive test suite covering all `core.Store` methods (Project, Credential, Message, StatusEvent, Batch, Verification, Unsubscribe, Attachment, WebhookDelivery, RequestLog, Transaction) with in-memory implementation for verification
- `core` package: domain types (Project, Credential, Message, StatusEvent, Batch, Verification, Unsubscribe, Attachment, WebhookDelivery, RequestLog), canonical errors with HTTP status mapping (validation_error=400, invalid_number=400, rate_limited=429, provider_unavailable=503, etc.), prefixed ULIDs (prj_, msg_, bat_, vrf_, whd_, req_, att_, blob_), ports (Store, BlobStore, Bus, Simulator, Clock, ProjectResolver), service with SendMessage/SendBatch/StartVerification/CheckVerification/ReceiveInbound, lifecycle runner with Clock-driven timers and step delay, FakeClock for deterministic tests
- `extract` package: OTP code extraction (4-8 digits, keyword-adjacent priority for code/otp/pin/verification/token/passcode), link extraction from text and HTML (href), primary_link detection (verify/confirm/activate/magic/token/reset/login/signin in URL or anchor text)
- `phone` package: E.164 parsing with `valid`/`possible`/`off` modes (using `nyaruka/phonenumbers`), GSM-7 vs UCS-2 encoding detection, segment counting (160/153 for GSM-7, 70/67 for UCS-2, max 10 segments)
- Repo scaffolding for M1: `go.mod`, `Taskfile.yml`, `.golangci.yml` (with `depguard` rules), `biome.json`, `.gitattributes`, `.editorconfig`, `.air.toml`, Apache-2.0 `LICENSE`, README stub, CI workflow (lint + test matrix on Linux/macOS/Windows)
- Design spec for mocksms (`docs/superpowers/specs/2026-10-03-mocksms-design.md`).
- Project documentation: overview, PRD, architecture, tech stack, engineering standards, tasks, progress, and agent instructions (`AGENTS.md`).

### Added

- M1-F03: Local gate matches CI - installed task, golangci-lint v2, pnpm; documented race detector setup for Windows (MinGW, WSL, Docker); updated Taskfile.yml to handle race detector gracefully; all Go tests pass locally

### Fixed

- CI configuration: migrated golangci-lint to v2 config (depguard disabled temporarily, re-enable in M1-F13), gated web jobs on web/ folder existence, aligned Go version to 1.26, ran `go mod tidy` and `gofmt -w`
- Native API error responses: preserve canonical error codes (invalid_number, rate_limited, verification_not_found, not_found, unauthorized) and HTTP status codes instead of rewriting to 500 internal; ensure Content-Type: application/json is preserved on all success and error responses by calling render.Status before render.JSON (M1-F04)
- Clean shutdown and startup: make LifecycleRunner.Stop idempotent with sync.Once and wait for timer goroutines with sync.WaitGroup; eliminate duplicate shutdown and store close calls in cmd/mocksms; fail fast with exit code 1 on HTTP server startup errors instead of hanging; remove premature SMTP server log line (M1-F05)
- Database migrations: embed SQL migrations into the binary using //go:embed and goose.SetBaseFS instead of looking up files at runtime via runtime.Caller(0), ensuring standalone binaries can boot outside the source tree (M1-F06)
- SQLite store: enable PRAGMA foreign_keys=1 and busy_timeout(5000) on all DSNs to enforce ON DELETE CASCADE across child tables; replace slice of single-connection pools with a true read pool configured via SetMaxOpenConns and SetMaxIdleConns while writeDB is limited to 1 connection; isolate in-memory stores with unique ULIDs per instance; wrap batch message insertions in a single transaction in core.Service; add cascade delete and store isolation tests to storetest (M1-F09)
