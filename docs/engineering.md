# mocksms — Engineering Standards

How we build mocksms: code style, testing, provider fidelity, git, CI, releases and the definition of done. AI agents follow the condensed version in [AGENTS.md](../AGENTS.md); this document is the full reference.

## 1. Principles

1. **Fidelity is the product.** A sandbox that behaves differently from production causes the bugs it was meant to prevent. When in doubt, match the provider.
2. **Boundaries over cleverness.** The package rules in [architecture.md](architecture.md) §3 matter more than any single optimization.
3. **Tests prove behavior, not lines.** Write the test that would catch a real regression.
4. **Local and private by default.** Never weaken a security default for convenience.
5. **Small, reviewable changes.** One task per PR.

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

- **Default branch:** `main`, always releasable. No direct pushes once CI exists.
- **Branches:** `<type>/<task-id>-<short-name>`, e.g. `feat/M2-03-twilio-messages`, `fix/fidelity-twilio-date-format`.
- **Commits:** [Conventional Commits](https://www.conventionalcommits.org/): `feat(twilio): add Verify v2 checks`, `fix(smtpd): handle empty AUTH username`, `docs:`, `test:`, `refactor:`, `chore:`, `ci:`.
- **PR scope:** one task from [tasks.md](tasks.md). Title uses the same Conventional Commit format.
- **Merge:** squash-merge; the PR title becomes the commit message.

### PR checklist

- [ ] Tests written first and passing (`task test`), lint clean (`task lint`).
- [ ] Golden-file changes reviewed.
- [ ] `openapi.yaml` updated and regenerated (if the native API changed).
- [ ] `docs/tasks.md` and `docs/progress.md` updated.
- [ ] `CHANGELOG.md` `[Unreleased]` updated for user-visible changes.
- [ ] `docs/architecture.md` updated with a decision-log entry (if structure changed).
- [ ] `docs/techstack.md` updated (if dependencies changed).
- [ ] `docs/fidelity.md` updated (if provider behavior was assumed).

## 9. CI gates

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

- **Versioning:** SemVer. Milestones map to `v0.1.0` (M1) through `v0.4.0` (M4); patch releases for fixes in between.
- **Process:**
  1. Move `[Unreleased]` entries in `CHANGELOG.md` to a new version heading with the date.
  2. Tag `vX.Y.Z` on `main`.
  3. The release workflow runs goreleaser: binaries (Windows, macOS, Linux; amd64/arm64), multi-arch Docker image to GHCR, Homebrew tap, Scoop bucket, checksums, cosign signatures.
  4. Verify the release by running the quick start from the README on a clean machine (or container).
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

A task is done when:

1. The behavior described in the task and its PRD requirements works, demonstrated by tests written first.
2. All CI gates pass on the PR.
3. Docs are updated per the PR checklist (tasks, progress, changelog, architecture, techstack, fidelity as applicable).
4. User-facing behavior is documented (README or docs page) if it changes how someone uses mocksms.
5. The PR is reviewed and squash-merged into `main`.

## 14. Windows development notes

Windows is a primary development platform.

- Use `task` instead of `make` (installed via `winget install Task.Task` or `go install github.com/go-task/task/v3/cmd/task@latest`).
- `.gitattributes` enforces LF line endings for source, fixtures and golden files so tests pass identically on every OS.
- No CGO, so no C toolchain is needed.
- Use `filepath` (not string concatenation) for paths; data defaults to `%LOCALAPPDATA%\mocksms`.
- Windows Defender can slow `go test` on large trees; exclude the repo folder if builds are slow.
