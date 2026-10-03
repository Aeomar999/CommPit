# AGENTS.md

Instructions for AI coding agents (Claude Code, Codex, Cursor, etc.) working in this repository. Human contributors should read [docs/engineering.md](docs/engineering.md).

## Project in one paragraph

mocksms is a local-first sandbox messaging provider for development. Apps send SMS, OTP and email to it instead of a real provider; nothing is delivered. Messages appear in a live web inbox, are readable via a test API, and run through a realistic lifecycle with provider-format webhooks. It is a single Go binary with an embedded React UI. Provider adapters (Twilio, Termii, SMTP in Wave 1) are thin translators over a native core.

**Current state:** documentation only. Code starts with task M1-01 in [docs/tasks.md](docs/tasks.md).

## Read before working

1. [docs/progress.md](docs/progress.md): current status and next action.
2. [docs/tasks.md](docs/tasks.md): pick the task; check its dependencies.
3. [docs/architecture.md](docs/architecture.md): package boundaries and dependency rules. **Binding.**
4. [docs/engineering.md](docs/engineering.md): code, test and git standards.
5. The relevant section of the [design spec](docs/superpowers/specs/2026-10-03-mocksms-design.md) and, if one exists, the milestone plan in `docs/superpowers/plans/`.

## Commands

Defined in `Taskfile.yml`, created in M1-01. Until then there is nothing to run.

```bash
task dev          # Go server + Vite dev server with proxy
task test         # Go tests with -race, plus UI unit tests
task lint         # golangci-lint (incl. depguard) + Biome
task generate     # oapi-codegen + openapi-typescript from openapi/openapi.yaml
task build        # UI build, then go build with the UI embedded
task e2e          # real-SDK examples + Playwright against the built binary
```

Use `go test ./... -run <Name> -update` to regenerate golden files, then review the diff.

## Architecture rules (do not break)

- `core` imports only `phone` and `extract`. Everything else depends on `core`, never the reverse.
- `core` has no HTTP code and no provider formats.
- Adapters never touch a store. They call the core service and translate.
- Ports (`Store`, `BlobStore`, `Bus`, `Simulator`, `Clock`, `ProjectResolver`) are defined in `core`.
- Every `Store` method takes a `projectID`.
- No package-level mutable state. Wire dependencies in `cmd/mocksms`.
- Only `store/sqlite` and `config` touch the filesystem.
- Extension packages stay public (no `internal/`). The closed hosted product imports them.
- `depguard` enforces the import rules. If it fails, fix the design, don't loosen the rule.

## Provider fidelity rules

- Match the real provider exactly: field names, status values, ID formats, date formats, error codes, HTTP status codes, webhook payloads and signatures.
- Every adapter endpoint has golden fixtures in `adapters/<provider>/testdata/`.
- Where a provider publishes an OpenAPI spec, responses must validate against the pinned copy.
- If documentation is unclear, do not guess silently. Implement the most likely behavior and add an entry to `docs/fidelity.md` marked **unverified**.
- Twilio SDK redirect snippets in `examples/` must keep passing in CI.

## Testing rules

- Write the failing test first, then the code (TDD).
- Table-driven tests in Go. Use `go-cmp` for comparisons.
- Never `time.Sleep` in tests. Use the injected fake `Clock` or the `messages/wait` endpoint.
- New `Store` behavior gets a case in `store/storetest`, not only in `store/sqlite`.
- Run `task test` (with `-race`) and `task lint` before declaring a task done.

## API and database changes

- Native API: edit `openapi/openapi.yaml` first, run `task generate`, then implement. Never edit generated files.
- Migrations: add a new numbered file in `store/sqlite/migrations/`. Never edit a migration that has been merged.

## Security defaults (never weaken without an explicit instruction from the project owner)

- Bind to `127.0.0.1` by default.
- `Host` header allow-list (DNS-rebinding protection); no CORS by default.
- Email HTML in a sandboxed iframe without `allow-scripts`; remote images opt-in.
- Mask credentials in request logs and the inspector.
- No telemetry, and no outbound calls except developer-configured webhooks.

## Workflow for every task

1. Mark the task `[~]` in `docs/tasks.md`.
2. Branch: `feat/M1-02-phone-package` (type/task-id-short-name).
3. Implement test-first. Keep the PR to one task.
4. In the same PR: mark the task `[x]`, update `docs/progress.md` (status table and session log), add user-visible changes to `CHANGELOG.md` under `[Unreleased]`, and update `docs/architecture.md` if structure changed (with a decision-log entry).
5. Commit with Conventional Commits (`feat(phone): add GSM-7 segment counting`).

## Adding a provider adapter (checklist)

1. Create `adapters/<provider>/` implementing `Adapter` (`Name`, `Routes`, `WriteError`), plus `StatusNotifier` / `InboundNotifier` / `ReplyParser` if the provider has them.
2. Map canonical errors to the provider's codes in `WriteError`.
3. Add golden fixtures for every endpoint; add contract tests if a spec exists.
4. Add a redirect example in `examples/` and run it in CI.
5. Add the provider to `docs/fidelity.md`, the docs site, and the `CHANGELOG.md`.
6. Never change `core` to fit one provider. If the core model is missing something, raise it with the project owner first.

## Don'ts

- Don't add dependencies without updating [docs/techstack.md](docs/techstack.md).
- Don't introduce CGO, an ORM, or an external broker.
- Don't make real network calls to providers, in code or tests.
- Don't commit generated UI builds (`web/dist`) or local data (`*.db`).
- Don't skip hooks or disable linters to make CI pass.
