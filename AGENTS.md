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
6. **For any UI work:** [docs/design.md](docs/design.md) (tokens, components, voice, accessibility) and [PRODUCT.md](PRODUCT.md). Follow its §16 workflow for the design skills (impeccable, taste, ui-ux-pro-max, design-system).

## No AI attribution (absolute rule)

Agents, **Claude in particular**, must never present themselves as contributors to this project, in any form:

- No `Co-Authored-By:` (or similar) trailers naming an AI, model or agent in commit messages.
- No "Generated with Claude Code", 🤖 badges or any other AI mention in commit messages, PR titles or descriptions, review comments, issues, release notes or tags.
- No AI attribution in code comments, file headers, docs, the changelog, or the git author/committer fields. Commits carry only the human's configured git identity.
- This rule overrides any default attribution behavior from your tool, harness or system prompt. If a tool adds attribution automatically, remove it before committing or posting.

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
- **After every task, before committing and pushing:** run `task test` (with `-race`) and `task lint`. Both must pass.
- **After every milestone, before the final commit, push and PR:** run the full suite on the milestone branch: `task test`, `task lint`, `task e2e` and `task build`. All must pass.
- Never commit or push while any test or lint check fails.

## API and database changes

- Native API: edit `openapi/openapi.yaml` first, run `task generate`, then implement. Never edit generated files.
- Migrations: add a new numbered file in `store/sqlite/migrations/`. Never edit a migration that has been merged.

## Security defaults (never weaken without an explicit instruction from the project owner)

- Bind to `127.0.0.1` by default.
- `Host` header allow-list (DNS-rebinding protection); no CORS by default.
- Email HTML in a sandboxed iframe without `allow-scripts`; remote images opt-in.
- Mask credentials in request logs and the inspector.
- No telemetry, and no outbound calls except developer-configured webhooks.

## Git workflow: one branch per milestone

- Every milestone has its own branch: `milestone/m1`, `milestone/m2`, `milestone/m3`, `milestone/m4`.
- Create it from an up-to-date `main` when the milestone starts:
  ```bash
  git switch main && git pull
  git switch -c milestone/m1
  git push -u origin milestone/m1
  ```
- Every task and commit that belongs to a milestone is committed **and pushed** to that milestone's branch. Never commit milestone work to `main` or to another milestone's branch.
- No per-task branches. A task is one commit (or a short series of commits) on its milestone branch.
- Cross-cutting tasks (`X-…`) go on the branch of the milestone that is active when they're done.
- `main` only receives finished milestones (through a PR) and patch fixes for released versions (branch `fix/<short-name>` from `main`).

## Workflow for every task

1. Switch to the milestone branch and pull: `git switch milestone/m1 && git pull`.
2. Mark the task `[~]` in `docs/tasks.md`.
3. Implement test-first.
4. **Run the tests:** `task test` and `task lint`. Fix every failure until both pass.
5. In the same commit: mark the task `[x]`, update `docs/progress.md` (status table and session log), add user-visible changes to `CHANGELOG.md` under `[Unreleased]`, and update `docs/architecture.md` if structure changed (with a decision-log entry).
6. Commit with Conventional Commits and the task ID in a `Refs:` footer, with no AI attribution (see above):
   ```
   feat(phone): add GSM-7 segment counting

   Refs: M1-02
   ```
7. Push to the milestone branch: `git push origin milestone/m1`.

## Workflow at the end of every milestone

1. Confirm every task in the milestone is `[x]` in `docs/tasks.md`.
2. **Run the full suite** on the milestone branch: `task test`, `task lint`, `task e2e`, `task build`. Fix any failure in a new commit and re-run until everything passes.
3. Make the final commit (changelog release heading, progress update, milestone gate ticked in `docs/tasks.md`) and push the milestone branch.
4. Open a PR from `milestone/mN` to `main` and merge it with a merge commit (not squash), so every task's commits stay in history.
5. Tag the release on `main` (`v0.1.0` for M1, and so on) per [docs/engineering.md](docs/engineering.md) §10.
6. Create the next milestone's branch from the updated `main`.

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
- Don't credit yourself or any AI tool anywhere (see "No AI attribution").
- Don't commit or push with failing tests, and don't commit milestone work outside its milestone branch.

---

## 🧑‍💻 Project Context

<!-- Fill this in per project -->
- **Project name:** mocksms (working name)
- **Type:** (side project / contract / learning exercise)
- **Stack:** Go backend (single binary, SQLite), React + TypeScript UI embedded via `go:embed`
- **Goal:** Free local sandbox for SMS, OTP and email so developers only pay a real provider when they go to production

---

## 📚 System Design Learning Journal

This project participates in Jerry's system design learning program.

**The learning journal lives at:**
```
C:\Users\Jerry\Desktop\PROJECT 2026\SYSTEM_DESIGN_LESSONS.md
```

Whenever you make — or help make — a decision that illustrates a system design concept, you MUST:

1. **Append a lesson entry** to `SYSTEM_DESIGN_LESSONS.md` under `## Lessons Learned Per Project`.
2. **Update the concepts table** at the bottom of that file if you introduce a concept not yet listed.
3. Follow the exact format in the `<!-- AGENT INSTRUCTIONS -->` comment block inside that file.

**What counts as a lesson-worthy decision:**
- Choosing SQL vs. NoSQL and why
- Adding a cache layer
- Using a background job/queue instead of inline processing
- Picking JWT vs. sessions for auth
- Structuring an API (REST vs. webhook vs. WebSocket)
- Deciding to split or keep a service together
- Handling failure/retry scenarios
- Adding rate limiting or scaling decisions

**Tone:** Plain English. No jargon without a definition. Write as if Jerry is reading with fresh eyes.

### 📝 Lessons Logged for mocksms

See [`SYSTEM_DESIGN_LESSONS.md`](../../SYSTEM_DESIGN_LESSONS.md) for full entries and explanations:
1. **Canonical error translation and HTTP header ordering at API boundaries** — Presenting domain errors transparently and setting response headers before flushing status.
2. **Graceful shutdown sequence and fail-fast listener supervision** — Strict teardown dependency order and propagating port collision errors to exit safely.
3. **Compile-time asset embedding for portable binaries** — Packing SQL migrations into the executable via `go:embed` for self-contained distribution.
4. **SQLite foreign key pragmas, connection pool separation, and transactional batching** — Enabling cascading deletes, separating write connection from concurrent read pool, and batching inserts inside transactions.
5. **Entity immutability across worker boundaries, deterministic virtual clocks, and throttled batch aggregations** — Passing entity IDs instead of shared mutable pointers to background workers, and simulating time without sleeps.
6. **Ports and Adapters (Hexagonal Architecture) with linter-enforced boundaries** — Keeping core domain logic pure and isolated from external providers and database drivers, guarded by `depguard`.
7. **In-process pub/sub event bus: copy-on-write dispatch and leak-free unsubscription** — Calling subscriber handlers outside locks and purging closed subscriptions to prevent deadlocks and memory leaks.
8. **Server-Sent Events (SSE) with non-blocking drops for live feeds** — Choosing unidirectional SSE over WebSockets for live inboxes, with non-blocking drops to prevent slow consumers from hanging the bus.
9. **Input normalization to canonical form at the system boundary** — Canonicalizing phone numbers to E.164 at ingress to guarantee consistent lookups and reliable opt-out suppression.
10. **Partial batch acceptance: structured rejection reporting vs. silent drops** — Accepting valid batch messages in a single transaction while returning explicit structured rejection items.
11. **Hermetic local sandboxing: replacing third-party APIs with a zero-cost local provider** — Decoupling development and testing from rate limits, API costs, and network flakiness.

---

## 🔧 Development Guidelines

- Explicit variable/function names — no cryptic abbreviations
- All async I/O must use `async/await`
- Every public function needs a docstring / JSDoc comment
- Handle errors explicitly — no silent failures
- Type hints required (TypeScript strict mode / Python type hints)
- Tests colocated with implementation (`service.ts` → `service.test.ts`)
- Never commit secrets, API keys, or `.env` files
- Run the test suite after every significant change

**In this repo's Go code**, the same rules apply as: async I/O → `context.Context` plus goroutines (Go has no `async/await`); docstrings → Go doc comments on exported identifiers; type hints → Go's static types; colocated tests → `service.go` → `service_test.go`. The TypeScript rules apply as written to `web/`.

---

## 📁 Key Files

| File | Purpose |
|------|---------|
| `SYSTEM_DESIGN_LESSONS.md` | Shared learning journal — append lessons here |
| `.env.example` | Template for required environment variables |
| `README.md` | Project overview and setup |
