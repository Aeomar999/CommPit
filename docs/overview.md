# mocksms — Overview

> A free sandbox for SMS, OTP and email. Build and test messaging flows against realistic fake providers, and pay for a real provider only when you go live.

**Status:** Design approved (2026-10-03). Implementation has not started. See [progress.md](progress.md).

## What it is

`mocksms` is a sandbox messaging provider that runs on your machine or in CI. Your app sends SMS, one-time passwords and email to `mocksms` instead of Twilio, Termii or SendGrid. Nothing is delivered. Instead, every message:

- appears instantly in a **live web inbox** (phone-style SMS threads, full email previews),
- can be read by your **automated tests** through a test API (`GET /api/v1/otp/latest?to=…`),
- moves through a **realistic delivery lifecycle** (`queued → sent → delivered`, or `failed`) with the same **webhooks and signatures** the real provider would send.

When you're ready for production, you point your app at the real provider: a base-URL or credentials change for Termii, SMTP and the native API, and a documented few-line snippet for Twilio.

## Who it's for

- **Developers** building signup, login, 2FA and notification flows who don't want to pay per test message or reach for their phone every time.
- **QA and automation engineers** who need end-to-end tests of OTP and email-verification flows that don't flake.
- **AI coding agents** that need to read an OTP to finish a flow (through the built-in MCP server).

## How it works

```
your app ── Twilio SDK / Termii HTTP / SMTP / native API ──▶  mocksms (one binary)
                                                               ├─ web inbox (live)
your tests ── GET /api/v1/otp/latest ─────────────────────────▶├─ test API
                                                               └─ webhooks ──▶ your app
```

Two ways to integrate:

1. **Provider-compatible adapters.** Keep using the official provider SDK and only change where it sends requests.
2. **Native API.** One clean, provider-neutral REST API for SMS, email and verifications.

## What's in the first release (Wave 1)

| Milestone | You get |
|---|---|
| **M1** | Single binary, native API, SMTP capture, live web inbox |
| **M2** | Twilio (SMS + Verify) and Termii (SMS, bulk, Token) adapters, test API, request inspector |
| **M3** | Delivery webhooks, failure simulation, inbound replies and STOP, bulk batch view |
| **M4** | Go-live cost estimate, MCP server for AI agents, CI kit (GitHub Action, Testcontainers, CLI) |

Later waves add SendGrid, Resend, Vonage, AWS SNS, Africa's Talking, Hubtel, Arkesel and mNotify.

## Planned usage (available from M1)

```bash
mocksms                      # starts HTTP on 127.0.0.1:4010 and SMTP on 127.0.0.1:1025
```

Point your app's SMTP settings at `localhost:1025`, open `http://localhost:4010`, and send an email.

## Principles

- **Zero setup.** No accounts, no config, any credentials accepted locally.
- **Realism over convenience.** Real validation, real error codes, real response shapes.
- **Local and private.** Listens on loopback only by default; no telemetry.
- **Open-core.** The local tool is Apache-2.0. A hosted team product may follow.

## Documentation map

| Document | Purpose |
|---|---|
| [PRD.md](PRD.md) | Product requirements: problem, personas, prioritized requirements, metrics |
| [architecture.md](architecture.md) | Living system architecture: components, boundaries, flows, data |
| [techstack.md](techstack.md) | Every technology used, and why |
| [engineering.md](engineering.md) | Engineering standards: code, tests, git, releases, definition of done |
| [tasks.md](tasks.md) | Milestone task breakdown (M1–M4) |
| [progress.md](progress.md) | Current status, session log, next steps, blockers |
| [../CHANGELOG.md](../CHANGELOG.md) | Release notes (Keep a Changelog) |
| [../AGENTS.md](../AGENTS.md) | Instructions for AI coding agents working in this repo |
| [superpowers/specs/2026-10-03-mocksms-design.md](superpowers/specs/2026-10-03-mocksms-design.md) | The approved design spec (frozen design record) |
