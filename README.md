# CommPit

A local-first sandbox messaging provider for development. Send SMS, OTP and email to it instead of a real provider; nothing is delivered. Messages appear in a live web inbox, are readable via a test API, and run through a realistic lifecycle with provider-format webhooks.

**Current status:** v0.1.0 (M1) — in development

## Quick start

```bash
# Install (once binaries are published)
# brew install aeomar999/tap/commpit
# scoop install commpit

# Or run from source
go run ./cmd/mocksms serve
```

The server starts on `http://127.0.0.1:4010` with the web inbox at `/` and SMTP on `:1025`.

## Features (Wave 1)

- **Native API** — Clean REST API for SMS, email, and verifications (`/api/v1/*`)
- **SMTP listener** — Receive email from any framework on `:1025`
- **Web inbox** — Live-updating UI with SMS threads, email viewer, OTP codes, filters
- **Test API** — `messages/wait`, `otp/latest`, `emails/latest` for reliable E2E tests
- **Lifecycle** — Messages progress through `queued` → `sent` → `delivered` (or `failed`)
- **Webhooks** — Status callbacks and inbound SMS in provider formats (M3)

## Documentation

- [Architecture](docs/architecture.md)
- [Design spec](docs/superpowers/specs/2026-10-03-mocksms-design.md)
- [Tasks & progress](docs/tasks.md)
- [Engineering standards](docs/engineering.md)

## License

Apache-2.0 © 2026 Aeomar999