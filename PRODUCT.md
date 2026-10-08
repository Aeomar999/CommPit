# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Single Go binary serving a React + Vite + TypeScript UI, styled with Tailwind CSS v4 and shadcn/ui, embedded with `go:embed`. Decided in [docs/techstack.md](docs/techstack.md).

## Users

- **Developers** (backend and full-stack) building signup, login, 2FA and notification flows. The inbox sits open beside their editor and terminal on a laptop and gets glanced at many times an hour while they iterate.
- **QA and automation engineers** who need end-to-end tests of OTP and email-verification flows that don't flake. They mostly use the test API; the UI is where they debug a failing run.
- **Freelancers and agency developers** juggling several client projects, often working alongside an AI coding agent. They need projects kept separate and want the agent to read OTPs itself.

In CI, nobody looks at the UI; the test API, CLI and MCP server carry that use.

## Product Purpose

mocksms is a free sandbox that stands in for SMS, OTP and email providers during development. Apps send to it instead of a real provider; nothing is delivered. Every message appears in a live inbox within a second, can be read by automated tests, and moves through a realistic delivery lifecycle with provider-format webhooks.

Success: a developer sees and uses any message instantly, tests the failure paths (undelivered, invalid number, unsubscribed, bad webhook signature) before launch, and switches to the real provider with a configuration change, paying nothing until then.

## Positioning

Faithful fakes of real providers, not a message catcher: provider-compatible APIs (Twilio, Termii and SMTP in the first release) with real validation, real error codes, real delivery lifecycles, and webhooks signed the way the real provider signs them. Focused on SMS and OTP, including African providers where almost no sandbox exists; email is supported but is not the differentiator.

## Operating Context

- Runs locally (`127.0.0.1:4010` for HTTP and the inbox, `:1025` for SMTP) or in CI via Docker, a GitHub Action, or Testcontainers.
- Developers keep the inbox in a browser tab or side window next to their editor, terminal and the app under test, often on a single laptop screen.
- Typical loop: trigger a signup in their app → the OTP appears → copy it → paste it into the app; or inspect a failed request in the inspector; or reply as the phone to test two-way flows.
- Projects are created from whatever credentials an app uses; several credentials can be linked into one project.

## Capabilities and Constraints

- Channels: SMS and email; OTP/verification on top of both. WhatsApp and voice are out of scope for now.
- Inbox: SMS conversations per number, email detail (HTML, text, source, headers, attachments), OTP list, bulk batches, request inspector, webhook log with replay, settings, go-live cost estimate.
- Constraints: no outbound traffic except developer-configured webhooks; no telemetry; email HTML renders sandboxed with remote images off by default; listens on loopback by default.
- Terminology: *project*, *message*, *batch*, *verification* (OTP session), *status* (`queued`, `sent`, `delivered`, `undelivered`, `failed`, `received`), *inspector*, *webhook delivery*, *magic numbers* (built-in failure triggers).
- Undecided: the final product name (working name `mocksms`), the hosted team product, pricing-data sources for the estimate.

## Brand Commitments

- Name: **mocksms** (working name; keep it in the UI for now).
- Voice: **warm and encouraging**. Friendly and supportive, never cute; errors stay clear, specific and actionable.
- Open-core: the local tool is Apache-2.0.
- No logo or visual identity assets exist yet.
- Visual direction (owner-pinned, 2026-10-03): a warm, light, orange-accented dashboard in the spirit of a reference bulk-SMS product the owner supplied. mocksms keeps its own identity: never the reference's name, logo, wordmark, copy or exact brand colour, and never presented so it could be mistaken for a live messaging provider. Details in [docs/design.md](docs/design.md).

## Evidence on Hand

None yet: no users, testimonials, case studies, benchmarks or customer logos. Future work must not fabricate any of them. Demo content in the UI (sample messages, numbers, projects) must be clearly synthetic.

## Product Principles

1. **Fidelity over convenience.** Behave like the real provider; a sandbox that differs from production causes the bugs it exists to prevent.
2. **Zero setup.** Works with any credentials, no accounts, sensible defaults.
3. **Show the raw truth.** Raw requests, responses, MIME and webhook payloads are always one click away; never hide errors.
4. **The code is the point.** OTPs, links and IDs are what people came for: instantly visible, one click to copy.
5. **Local and private.** Everything stays on the developer's machine.

## Accessibility & Inclusion

WCAG 2.2 AA. Fully keyboard-navigable. Status is never conveyed by colour alone (always paired with an icon or label). Light and dark themes. Latest two versions of Chrome, Edge, Firefox and Safari.
