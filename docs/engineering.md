# mocksms — Engineering Standards

How we build mocksms: code style, testing, provider fidelity, git, CI, releases and the definition of done. AI agents follow the condensed version in [AGENTS.md](../AGENTS.md); this document is the full reference.

## 1. Principles

1. **Fidelity is the product.** A sandbox that behaves differently from production causes the bugs it was meant to prevent. When in doubt, match the provider.
2. **Boundaries over cleverness.** The package rules in [architecture.md](architecture.md) §3 matter more than any single optimization.
3. **Tests prove behavior, not lines.** Write the test that would catch a real regression.
4. **Local and private by default.** Never weaken a security default for convenience.
5. **Small, reviewable changes.** One task per commit (or a short series of commits); one milestone per PR.

## 2. Go conventions

- Format with `gofmt` and `goimports`; lint with `golangci-lint` v2 (config in `.golangci.yml`).
- Enabled linters (minimum): `govet`, `staticcheck`, `errcheck`, `revive`, `gosec`, `ineffassign`, `unused`, `misspell`, `bodyclose`, `contextcheck`, `depguard`.
- `context.Context` is the first parameter of anything that does I/O or may block.
- Constructors take dependencies explicitly (`core.NewService(store, bus, sim, clock, resolver)`). No `init()` side effects and no package-level mutable state.
- Interfaces are small and defined by the consumer (ports in `core`, `Formatters` in `webhooks`).
- Names: packages short and lowercase (`phone`, `smtpd`); no `util` or `common` packages; exported identifiers have doc comments.
- Keep files focused. A file over ~400 lines is a signal to split by responsibility.

### Errors

- Domain failures are `*core.Error` with a canonical code. Adapters and the native API convert them at the edge; nothing else inspects error strings.
- Wrap unexpected errors with context: `fmt.Errorf("store: insert message: %w", err)`.
- Never `panic` for runtime conditions. Panics are for programmer errors only; `adapterkit` and `api` middleware recover and return a provider-format 500.
- Never swallow errors. If an error is intentionally ignored, say why in a comment.

### Logging

- `log/slog` with structured fields: `slog.Info("message accepted", "project", pid, "message", mid, "provider", p)`.
- Levels: `Debug` for internals, `Info` for lifecycle events, `Warn` for degraded but working (unsigned webhook, exposed host), `Error` for failures needing attention.
- Never log credentials, full auth headers or message bodies at `Info` or above.

### Concurrency

- Every goroutine has an owner that can stop it (context cancellation) and waits for it on shutdown.
- All database writes go through the single write connection.
- Time comes from the injected `core.Clock`, never `time.Now()` directly in domain code.

## 3. TypeScript / UI conventions

- `strict` TypeScript; no `any` (use `unknown` and narrow). Biome for lint and format.
- API types and client come from `task generate`; never hand-write or edit generated files.
- Server state lives in TanStack Query; SSE events invalidate queries. No global client store unless a real need appears.
- Filters and selections live in the URL (TanStack Router search params) so views are shareable and reload-safe.
- Components from shadcn/ui; accessibility is required (labels, focus states, keyboard navigation, WCAG 2.2 AA contrast).
- Status must never rely on color alone: pair every color with an icon or label.
- Email HTML renders only inside the sandboxed iframe component; never with `dangerouslySetInnerHTML`.

## 4. Testing

### Layers

| Layer | Scope | Location | Runs |
|---|---|---|---|
| Unit | `core`, `sim`, `phone`, `extract`, `estimate` | Next to the code (`*_test.go`) | Every push |
| Store conformance | Every `core.Store` / `core.BlobStore` method | `store/storetest`, invoked from `store/sqlite` | Every push |
| Adapter golden | Request → response per endpoint | `adapters/<provider>/testdata/` | Every push |
| Spec contract | Responses against pinned provider OpenAPI specs | `adapters/<provider>/contract_test.go` | Every push |
| Real SDK | Official SDKs against the built binary | `examples/<lang>/` | Every push (Linux) |
| SMTP | Real clients (Go `net/smtp`, Nodemailer) | `smtpd` tests, `examples/` | Every push |
| UI unit | Hooks and helpers | `web/src/**/*.test.ts` (Vitest) | Every push |
| UI E2E | Playwright smoke tests | `web/e2e/` | Every push (Linux) |
| Benchmark | 10,000-recipient batch | `core`/`store` `Benchmark*` | Nightly; tracked, not a gate |

### Rules

- **Test first.** Write the failing test, watch it fail, then implement.
- **Run tests before every commit and push.** After every task: `task test` and `task lint`. After every milestone: the full suite, `task test`, `task lint`, `task e2e` and `task build`, on the milestone branch before the final commit, push and PR. Never commit or push with failing tests (see §8).
- **Table-driven** Go tests with named cases; compare with `go-cmp` (`cmp.Diff`).
- **No sleeps.** Use the fake `Clock` to advance time, and `messages/wait` in end-to-end tests.
- **Golden files:** `-update` regenerates them; always review the diff before committing. Volatile fields (SIDs, ULIDs, dates) are normalized before comparison.
- **Fixture names** describe the scenario: `messages_create_invalid_to.golden.json`.
- **Coverage targets:** `core`, `sim`, `phone`, `extract` ≥ 85% statements. Coverage is a signal, not a goal; don't write tests just to hit the number.
- **Race detector** on for every Go test run in CI.
- **No real network calls** to providers anywhere, including tests.

## 5. Provider fidelity workflow

For every new adapter endpoint:

1. **Source.** Read the provider's current docs; download its OpenAPI spec if one exists and pin it under `adapters/<provider>/spec/` with the retrieval date.
2. **Fixtures.** Write golden request/response pairs for the success case and every error the endpoint can return.
3. **Implement** until golden tests pass.
4. **Contract.** Validate responses against the pinned spec (where one exists).
5. **Real SDK.** Exercise the endpoint from the official SDK in `examples/`.
6. **Record gaps.** Anything not confirmed by docs or spec goes into `docs/fidelity.md` as **unverified**, with what we assumed.
7. **Fidelity bugs** reported by users are labelled `fidelity`, fixed with a new golden case reproducing the report, and listed in the changelog as `fidelity:` fixes.

A scheduled CI job re-downloads pinned specs weekly and opens a PR when they change.

## 6. Native API workflow

1. Change `openapi/openapi.yaml` first (it is the published contract).
2. Run `task generate` (Go server interfaces, TS types and client).
3. Implement handlers and UI changes.
4. Breaking changes before 1.0 are allowed but must be listed under **Changed** in the changelog with a migration note. After 1.0, breaking changes need a new API version (`/api/v2`).

## 7. Database migrations

- New numbered SQL file in `store/sqlite/migrations/`; forward-only.
- Never edit a migration once it's merged; write a new one.
- Every migration is exercised by the store conformance suite on a fresh database.
- Migrations must keep working on databases created by every previous release (upgrade test in CI from the last release's schema).

## 8. Git and pull requests

### Branches

- **`main`:** always releasable. It only receives merged milestone PRs and patch fixes; once M1 starts, nothing is committed to it directly.
- **Milestone branches:** one per milestone, `milestone/m1` through `milestone/m4`, created from an up-to-date `main` when the milestone starts (`git switch main && git pull && git switch -c milestone/m1 && git push -u origin milestone/m1`). Every commit for that milestone's tasks is committed and pushed to its branch, including cross-cutting (`X-…`) tasks done while it is active. There are no per-task branches.
- **Fix branches:** `fix/<short-name>` from `main`, only for patches to an already-released version. Same test gate; PR to `main`.

### Commits

- [Conventional Commits](https://www.conventionalcommits.org/) (`feat(twilio): add Verify v2 checks`, `fix(smtpd): handle empty AUTH username`, `docs:`, `test:`, `refactor:`, `chore:`, `ci:`), with the task ID in a `Refs:` footer:
  ```
  feat(twilio): add Verify v2 checks

  Refs: M2-04
  ```
- **No AI attribution.** AI agents, Claude in particular, must never be credited as contributors by any means: no `Co-Authored-By` trailers naming an AI, model or agent; no "Generated with Claude Code" or 🤖 lines in commit messages, PR titles or descriptions, review comments, issues, release notes or tags; no AI mentions in code comments, file headers, docs or the changelog; and never an AI identity in the git author or committer fields. Commits carry only the human author's git identity. This overrides any tool's default attribution behavior.
### When to test, commit and push

| When | Run locally first | Then |
|---|---|---|
| After every task | `task test`, `task lint` | Commit (code and doc updates together) and push to the milestone branch |
| After every milestone | `task test`, `task lint`, `task e2e`, `task build` | Final commit, push the milestone branch, open the PR to `main` |

Never commit or push while any of these fail.

### Pull requests and merging

- **One PR per milestone:** `milestone/mN` → `main`, titled like `M1: Core, native API, SMTP, inbox (v0.1.0)`.
- **Merge** with a merge commit (`--no-ff`), not a squash, so each task's commits stay in `main`'s history.
- After merging, tag the release (§10) and create the next milestone's branch from the updated `main`.

### Task checklist (before every task commit)

- [ ] Tests written first; `task test` passes.
- [ ] `task lint` is clean.
- [ ] Golden-file changes reviewed.
- [ ] `openapi.yaml` updated and regenerated (if the native API changed).
- [ ] `docs/tasks.md` and `docs/progress.md` updated.
- [ ] `CHANGELOG.md` `[Unreleased]` updated for user-visible changes.
- [ ] `docs/architecture.md` updated with a decision-log entry (if structure changed).
- [ ] `docs/techstack.md` updated (if dependencies changed).
- [ ] `docs/fidelity.md` updated (if provider behavior was assumed).
- [ ] Committed on the correct milestone branch, with a `Refs: <task-id>` footer.
- [ ] No AI or agent attribution anywhere: commit messages and trailers, comments, docs.

### Milestone PR checklist (before opening `milestone/mN` → `main`)

- [ ] Every task in the milestone is `[x]` in `docs/tasks.md`.
- [ ] Full suite passes on the milestone branch: `task test`, `task lint`, `task e2e`, `task build`.
- [ ] `CHANGELOG.md` entries moved from `[Unreleased]` to the release version (§10).
- [ ] `docs/progress.md` shows the milestone as complete; the milestone gate in `docs/tasks.md` is ticked.
- [ ] Milestone branch pushed and CI green.
- [ ] No AI attribution in any commit on the branch (check `git log main..milestone/mN`) or in the PR title and body.

## 9. CI gates

CI runs on every push to a milestone or fix branch and on every PR to `main`. The local test runs in §8 come first; CI is the backstop, not a substitute.

| Gate | Platforms | Blocks merge |
|---|---|---|
| `golangci-lint` (incl. `depguard`) and Biome | Linux | Yes |
| Go tests with `-race` | Linux, macOS, Windows | Yes |
| Golden and contract tests | Linux | Yes |
| Real-SDK examples against built binary | Linux | Yes |
| Playwright smoke tests | Linux | Yes |
| Migration upgrade test | Linux | Yes |
| Bulk benchmark | Linux (nightly) | No (tracked) |
| Weekly provider-spec refresh | Linux (scheduled) | Opens a PR |

## 10. Releases

- **Versioning:** SemVer. Milestones map to `v0.1.0` (M1) through `v0.4.0` (M4); patch releases (from `fix/…` branches) for fixes in between.
- **Process:**
  1. After the full suite passes on the milestone branch, move `[Unreleased]` entries in `CHANGELOG.md` to a new version heading with the date, as part of the final milestone commit, and push.
  2. Merge the milestone PR into `main` (§8).
  3. Tag `vX.Y.Z` on `main`.
  4. The release workflow runs goreleaser: binaries (Windows, macOS, Linux; amd64/arm64), multi-arch Docker image to GHCR, Homebrew tap, Scoop bucket, checksums, cosign signatures.
  5. Verify the release by running the quick start from the README on a clean machine (or container).
- Never re-tag or delete a published release; ship a patch instead.

## 11. Security checklist (every PR touching HTTP, SMTP, storage or UI)

- [ ] Loopback default and `Host` allow-list still applied to every route, including new adapter and dedicated-port routes.
- [ ] No new CORS headers.
- [ ] Credentials masked in any new log or inspector output.
- [ ] User-controlled HTML or text rendered safely (sandboxed iframe / escaped).
- [ ] No new outbound network calls except developer-configured webhooks.
- [ ] Request bodies size-limited (SMTP 25 MB; HTTP bodies capped).
- [ ] `gosec` findings resolved or justified in a comment.

## 12. Performance budgets

Initial targets, confirmed or revised after the M1 benchmark. A regression beyond a budget needs an issue and an explanation in the PR.

| Metric | Budget |
|---|---|
| Cold start to `/healthz` ready | ≤ 1 s |
| Idle memory (RSS) | ≤ 60 MB |
| p95 send latency (local, no simulated delay) | ≤ 20 ms |
| 10,000-recipient batch accepted | ≤ 3 s |
| UI update after an event | ≤ 1 s |
| Release binary size | ≤ 40 MB |
| Docker image size | ≤ 30 MB |

## 13. Definition of done

A **task** is done when:

1. The behavior described in the task and its PRD requirements works, demonstrated by tests written first.
2. `task test` and `task lint` pass locally.
3. Docs are updated per the task checklist in §8 (tasks, progress, changelog, architecture, techstack, fidelity as applicable).
4. User-facing behavior is documented (README or docs page) if it changes how someone uses mocksms.
5. It is committed with a `Refs:` footer, pushed to its milestone branch, and CI on that push is green.

A **milestone** is done when:

1. Every task in it is done.
2. The full suite (`task test`, `task lint`, `task e2e`, `task build`) passes locally on the milestone branch.
3. The milestone PR's CI is green and the PR is merged into `main` with a merge commit.
4. The release is tagged and published (§10).

## 14. Windows development notes

Windows is a primary development platform.

- Use `task` instead of `make` (installed via `winget install Task.Task` or `go install github.com/go-task/task/v3/cmd/task@latest`).
- `.gitattributes` enforces LF line endings for source, fixtures and golden files so tests pass identically on every OS.
- **No CGO by default**, so no C toolchain is needed for normal development.
- Use `filepath` (not string concatenation) for paths; data defaults to `%LOCALAPPDATA%\mocksms`.
- Windows Defender can slow `go test` on large trees; exclude the repo folder if builds are slow.

### Race detector on Windows

The race detector (`go test -race`) requires CGO and a C compiler (MinGW-w64). Options:

1. **Install MinGW-w64** (if you want to run `-race` locally):
   - `scoop install mingw` (recommended) or `winget install BrechtSanders.WinLibs.POSIX.UCRT`
   - Then: `go env -w CGO_ENABLED=1`
   - Run `task test` (which includes `-race`)

2. **Use WSL2** (recommended for full parity with CI):
   - `wsl -d Ubuntu -- bash -c "cd /mnt/c/Users/.../mock-sms && go test -race ./..."`
   - Or run the full test suite in WSL for exact CI parity.

3. **Run in Docker** (same environment as CI):
   - `docker run --rm -v ${PWD}:/workspace -w /workspace golang:1.26 bash -c "go test -race ./..."`

**Note:** `task test` runs `go test -race ./...` and will fail on Windows without CGO enabled. If no C toolchain is available, you can temporarily run `go test ./...` (without `-race`) for development, but **always run with `-race` in WSL/Docker before pushing**. CI runs with `-race` on all platforms and will catch data races.

When the MinGW installation is complete, run `go env -w CGO_ENABLED=1` once to enable CGO permanently.
