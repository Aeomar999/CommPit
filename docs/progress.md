# mocksms — Progress

Single source of truth for where the project stands. Update it in every PR that changes task status (see [AGENTS.md](../AGENTS.md)).

## Current status

| | |
|---|---|
| **Phase** | M1 in progress |
| **Current milestone** | M1 |
| **Next action** | M1-02: `phone` package (E.164 parsing, GSM-7/UCS-2, segment counting) |
| **Last updated** | 2026-10-03 |

## Milestones

| Milestone | Release | Status | Tasks done |
|---|---|---|---|
| M1: Core, native API, SMTP, inbox | v0.1.0 | In progress | 1 / 21 |
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
