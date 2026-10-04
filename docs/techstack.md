# mocksms — Tech Stack

Every technology in the project, what it does here, and why it was chosen. Entries marked **(post-spec)** were decided while writing this document rather than in the design discussion; challenge them before M1-01 if you disagree.

**Version policy:** use the latest stable release of each dependency at project start, pinned in `go.mod` / `pnpm-lock.yaml`. Dependabot (or Renovate) opens update PRs weekly; CI must pass before merging.

## Backend (Go)

| Technology | Purpose | Why | Alternatives considered |
|---|---|---|---|
| **Go ≥ 1.26** | Backend language | Single static binary, fast startup, easy cross-compilation, strong stdlib for HTTP and concurrency | TypeScript/Node (bigger runtime, `npx` install), Python (hard single-file distribution) |
| **go-chi/chi v5** | HTTP routing | Lightweight, stdlib-compatible handlers, supports Twilio-style paths (`{sid}.json`), works with `oapi-codegen` | stdlib `ServeMux` (wildcards must be whole path segments), Gin/Echo (non-stdlib handler types) |
| **oapi-codegen v2** | Generates native-API server interfaces from `openapi/openapi.yaml` | Spec-first API; chi "strict server" mode gives typed request/response handlers | Hand-written handlers (drift from the spec) |
| **modernc.org/sqlite** | Embedded database | Pure Go, no CGO, so cross-compiling for Windows/macOS/Linux stays trivial | `mattn/go-sqlite3` (needs CGO) |
| **pressly/goose v3** | Database migrations | Supports embedded SQL files via `embed.FS`; simple forward-only workflow | golang-migrate (heavier) |
| **emersion/go-smtp** | SMTP server | Mature, supports AUTH PLAIN/LOGIN and STARTTLS | Writing an SMTP server by hand |
| **jhillyerd/enmime** | MIME parsing | Handles text/HTML alternatives, attachments and inline `cid:` images; used by Inbucket | `emersion/go-message` (lower level) |
| **nyaruka/phonenumbers** | E.164 parsing and validation | Go port of Google's libphonenumber; same rules real providers use | Regex validation (inaccurate) |
| **oklog/ulid v2** | Sortable unique IDs | Time-ordered IDs make cursor pagination simple | UUIDv7 (also fine; ULIDs read better in URLs) |
| **modelcontextprotocol/go-sdk** | MCP server at `/mcp` | Official Go SDK; streamable HTTP transport | Third-party MCP libraries |
| **knadh/koanf v2** **(post-spec)** | Config loading with flags > env > YAML > defaults | Small, composable providers, clear precedence | spf13/viper (heavier, global state) |
| **spf13/cobra** **(post-spec)** | CLI subcommands (`serve`, `otp`, `messages`, `reset`, `version`) | De facto standard; pflag integrates with koanf | stdlib `flag` (no subcommands) |
| **log/slog** **(post-spec)** | Structured logging | Stdlib, structured, no dependency | zap, zerolog |

## Frontend (web inbox)

| Technology | Purpose | Why | Alternatives considered |
|---|---|---|---|
| **React + TypeScript** | UI | Largest ecosystem; strong typing | Svelte, Vue |
| **Vite** | Dev server and build | Fast; static output is easy to embed with `go:embed` | Next.js (needs a Node server) |
| **Tailwind CSS v4** | Styling | Utility classes, small output | CSS modules |
| **shadcn/ui** | Component primitives | Accessible Radix-based components copied into the repo, so no runtime dependency lock-in | MUI, Chakra |
| **TanStack Query** | Server state and caching | SSE events invalidate queries, giving live updates cheaply | SWR, Redux |
| **TanStack Router** **(post-spec)** | Routing | Type-safe routes and search params (filters live in the URL) | React Router |
| **openapi-typescript + openapi-fetch** **(post-spec for openapi-fetch)** | API types and typed client generated from `openapi.yaml` | UI and API can't drift apart | Hand-written fetch calls |
| **pnpm** | Package manager | Fast, strict dependency resolution; used in Taskfile, CI, and web/ | npm, yarn |
| **Biome** **(post-spec)** | Linting and formatting for TS/JSON/CSS | One fast tool instead of ESLint + Prettier | ESLint + Prettier |
| **Phosphor Icons** (`@phosphor-icons/react`) **(design)** | The only icon library; replaces lucide in copied shadcn components | Regular, duotone and fill weights match the design's soft navigation icons; one consistent family ([design.md](design.md) §8) | lucide-react (shadcn default) |
| **Figtree + JetBrains Mono** via Fontsource (`@fontsource-variable/figtree`, `@fontsource-variable/jetbrains-mono`) **(design)** | UI typeface and code/data typeface, self-hosted | Friendly, legible UI face; unambiguous characters for OTP codes and IDs; no hosted font stylesheets ([design.md](design.md) §6) | Inter + Fira Code |

## Testing

| Technology | Purpose |
|---|---|
| Go `testing` + **google/go-cmp** **(post-spec)** | Unit, golden and conformance tests; readable diffs |
| **getkin/kin-openapi** | Validates adapter responses against pinned provider OpenAPI specs (Twilio; later SendGrid, Vonage) |
| **Official provider SDKs** (twilio-node, twilio-python, …) | Real-SDK compatibility tests in `examples/` |
| **Nodemailer** | Real SMTP client tests in `examples/` |
| **Vitest** **(post-spec)** | UI unit tests (hooks, formatting helpers) |
| **Playwright** | UI end-to-end smoke tests |

## Tooling, CI and distribution

| Technology | Purpose | Why |
|---|---|---|
| **Task (taskfile.dev)** **(post-spec)** | Task runner (`task dev`, `task test`, `task build`) | Cross-platform. `make` isn't available by default on Windows, the primary dev machine |
| **golangci-lint v2** | Go linting, including `depguard` for package dependency rules | One tool, many linters, enforces architecture boundaries |
| **GitHub Actions** | CI (Linux/macOS/Windows matrix), scheduled spec-refresh job, releases | Free for open source; hosts the `setup-mocksms` action |
| **goreleaser** | Cross-platform binaries, Docker images, Homebrew tap, Scoop bucket, checksums | One config for every release artefact |
| **cosign** | Signs release artefacts | Supply-chain trust for a binary people run locally |
| **Distroless static image** (`gcr.io/distroless/static`) | Docker base | Tiny, no shell, minimal attack surface; works because the binary has no CGO |
| **GHCR** | Container registry | Free for public images, next to the code |
| **Testcontainers** (Node, Go, Python) | CI kit modules | The standard way to run service containers in tests |

## Explicitly not used

| Not used | Why |
|---|---|
| CGO | Breaks easy cross-compilation |
| An ORM | A small, hand-written SQL layer behind `core.Store` is clearer and keeps the Postgres port honest |
| Redis, NATS or any external broker (Wave 1) | In-process `core.Bus` is enough locally; the hosted product can swap it in |
| Telemetry or analytics SDKs | Product principle: no telemetry |
