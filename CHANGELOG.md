# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Before 1.0.0, minor versions may include breaking changes to the native API; they are called out under **Changed** with a migration note.

Provider-compatible adapters follow the real providers' behavior. Fixes that make an adapter match its provider more closely are listed under **Fixed** with the `fidelity:` prefix.

## [Unreleased]

### Added

- `bus` package: in-process pub/sub implementing `core.Bus` with Publish/Subscribe, event types (message.created, message.status, verification.updated, batch.updated, request.logged, webhook.delivered), concurrent-safe, unsubscribe support
- `store/sqlite`: SQLite implementation of `core.Store` and `core.BlobStore` with WAL mode, single writer + configurable read pool, goose migrations (embedded), blob storage, transactional batch insert; passes `store/storetest` conformance suite
- `store/storetest` conformance suite: comprehensive test suite covering all `core.Store` methods (Project, Credential, Message, StatusEvent, Batch, Verification, Unsubscribe, Attachment, WebhookDelivery, RequestLog, Transaction) with in-memory implementation for verification
- `core` package: domain types (Project, Credential, Message, StatusEvent, Batch, Verification, Unsubscribe, Attachment, WebhookDelivery, RequestLog), canonical errors with HTTP status mapping (validation_error=400, invalid_number=400, rate_limited=429, provider_unavailable=503, etc.), prefixed ULIDs (prj_, msg_, bat_, vrf_, whd_, req_, att_, blob_), ports (Store, BlobStore, Bus, Simulator, Clock, ProjectResolver), service with SendMessage/SendBatch/StartVerification/CheckVerification/ReceiveInbound, lifecycle runner with Clock-driven timers and step delay, FakeClock for deterministic tests
- `extract` package: OTP code extraction (4-8 digits, keyword-adjacent priority for code/otp/pin/verification/token/passcode), link extraction from text and HTML (href), primary_link detection (verify/confirm/activate/magic/token/reset/login/signin in URL or anchor text)
- `phone` package: E.164 parsing with `valid`/`possible`/`off` modes (using `nyaruka/phonenumbers`), GSM-7 vs UCS-2 encoding detection, segment counting (160/153 for GSM-7, 70/67 for UCS-2, max 10 segments)
- Repo scaffolding for M1: `go.mod`, `Taskfile.yml`, `.golangci.yml` (with `depguard` rules), `biome.json`, `.gitattributes`, `.editorconfig`, `.air.toml`, Apache-2.0 `LICENSE`, README stub, CI workflow (lint + test matrix on Linux/macOS/Windows)
- Design spec for mocksms (`docs/superpowers/specs/2026-10-03-mocksms-design.md`).
- Project documentation: overview, PRD, architecture, tech stack, engineering standards, tasks, progress, and agent instructions (`AGENTS.md`).
