# mocksms — Design Spec

- **Date:** 2026-10-03
- **Status:** Approved in brainstorming, pending written-spec review
- **Working name:** `mocksms` (renaming is a find-and-replace; the Go module path is set when the GitHub repository is created)

## 1. Summary

`mocksms` is a sandbox messaging provider for development and testing. Applications send SMS, OTP/verification and email to `mocksms` instead of a real provider. Nothing is delivered. Every message is captured, shown in a live web inbox, readable through a test API, and run through a realistic delivery lifecycle with webhooks. When a team is ready for production, it changes a base URL or credentials and points at the real provider. Developers pay nothing until they go live.

Two integration styles are supported:

1. **Native API.** One clean REST API (`/api/v1/sms`, `/email`, `/verifications`) for teams that want a provider-neutral interface.
2. **Provider-compatible adapters.** Fake versions of real provider APIs (Twilio, Termii, SMTP, and in later waves SendGrid, Resend, Vonage, AWS SNS and more African providers). Teams keep using the official provider SDKs and only change where those SDKs send requests.

Adapters are thin translators over the native core. Adding a provider never changes the core.

## 2. Goals and non-goals

### Goals

- Run locally as a **single binary** (or Docker image) with zero accounts and zero configuration.
- Make the switch to production a configuration change for SMTP, Termii and the native API, and a documented few-line HTTP-client snippet for Twilio.
- Behave like a real provider: real validation, real error codes, real response shapes, real webhook formats and signatures.
- Make automated testing of OTP and email-verification flows reliable (no sleeps, no races, parallel-safe).
- Keep the architecture ready for a hosted, multi-tenant product without building that product now.

### Non-goals (for this spec)

- The hosted SaaS itself (accounts, teams, billing, Postgres deployment). Only the seams are built now (§12).
- Real message delivery of any kind.
- WhatsApp and voice channels. The `channel` enum stays open for them.
- Providers beyond Wave 1 are designed at overview level only (§7.5); each gets its own detailed fidelity pass when built.

## 3. Key decisions

| Decision | Choice |
|---|---|
| Hosting model | Local-first tool; structured so a hosted version can be added later |
| Integration model | Native unified API as core + provider-compatible adapters as translators |
| Backend | Go (≥ 1.25), single static binary, no CGO |
| Frontend | React + Vite + TypeScript, Tailwind, shadcn/ui, TanStack Query; embedded via `go:embed` |
| Architecture | Modular monolith in one process (Approach A) |
| API accuracy | Provider OpenAPI specs used for contract tests where available; real-SDK tests in CI |
| Storage | SQLite via `modernc.org/sqlite` (pure Go); in-memory mode for CI |
| License | Open-core: local tool Apache-2.0; hosted product closed source |

## 4. Scope and delivery plan

### Wave 1 (this spec), delivered as four milestones

Each milestone is a usable release.

| Milestone | Ships | Unlocks |
|---|---|---|
| **M1** | Core (including batches), store, native API, SMTP listener, web inbox, live updates | Capturing email from any framework; SMS via the native API, including multi-recipient sends |
| **M2** | Twilio adapter (Messages + Verify v2), Termii adapter (SMS, bulk, Token), test-assertion API, request inspector. Callback URLs are accepted and stored | Using real provider SDKs; automated E2E tests |
| **M3** | Webhook worker (delivers the callbacks stored since M2), failure simulation, inbound SMS and replies, STOP handling, Batches view in the UI | Delivery-receipt, error-path and two-way flows |
| **M4** | Go-live estimate, MCP server, CI kit (GitHub Action, Testcontainers modules, CLI subcommands) | AI coding agents, CI pipelines, launch cost planning |

### Later waves (roadmap, not in this spec)

- **Wave 2:** SendGrid, Resend, Vonage (SMS + Verify), AWS SNS.
- **Wave 3:** Africa's Talking, Hubtel, Arkesel, mNotify.
- **Backlog:** npm wrapper (`npx mocksms`), stdio MCP bridge, WhatsApp and voice channels, the hosted product.

Implementation plans are written per milestone, starting with M1.

## 5. Architecture

```
                     ┌──────────────────── mocksms (single Go binary) ─────────────────────┐
 your app ──HTTP───▶ │ :4010 router                                                         │
                     │   /api/v1/*     native API ──────┐                                   │
                     │   /twilio/*     adapter ─────────┤                                   │
                     │   /termii/*     adapter ─────────┼──▶ core service ───▶ store        │
                     │   /mcp          MCP server ──────┤      │   ▲  (sim rules)  (SQLite) │
 your app ──SMTP───▶ │ :1025 SMTP listener ─────────────┘      │   └──                      │
                     │                                         ▼                            │
                     │                                     event bus                        │
                     │                                   ┌─────┴──────┐                     │
                     │                          webhook worker     SSE hub ──▶ web inbox (embedded)
                     └────────────────────────────────┼─────────────────────────────────────┘
                                                      ▼
                          status callbacks / inbound SMS ──▶ your app's webhook URL
```

### 5.1 Package layout

Extension-point packages are **public** (top level, not `internal/`) so the closed hosted product, a separate Go module, can compose them. The Go package API carries no compatibility promise before 1.0.

| Package | Responsibility |
|---|---|
| `cmd/mocksms` | Local composition root: config, wiring, subcommands (`serve`, `otp`, `messages`, `reset`, `version`) |
| `core` | Domain types and service: `SendMessage`, `SendBatch`, `StartVerification`, `CheckVerification`, `ReceiveInbound`; lifecycle runner; canonical errors; and the ports it consumes: `Store`, `BlobStore`, `Bus`, `Simulator`, `Clock`, `ProjectResolver`. No HTTP, no provider formats |
| `store/storetest` | Shared conformance suite for any `core.Store` / `core.BlobStore` implementation |
| `store/sqlite` | SQLite implementation of `core.Store` and `core.BlobStore`, plus embedded migrations |
| `bus` | In-process implementation of `core.Bus` |
| `sim` | Failure-simulation rule engine implementing `core.Simulator`, checked by core before accepting a send |
| `extract` | OTP-code and link extraction from SMS and email bodies |
| `phone` | E.164 parsing (`nyaruka/phonenumbers`), GSM-7/UCS-2 encoding detection, segment counting |
| `api` | Native REST API (generated from `openapi/openapi.yaml` with `oapi-codegen`), test helpers, SSE hub |
| `adapters/adapterkit` | Shared middleware (panic recovery, credential → project resolution, request logging), form/JSON helpers |
| `adapters/twilio`, `adapters/termii` | One package per provider |
| `smtpd` | SMTP listener (`emersion/go-smtp`) and MIME parsing (`jhillyerd/enmime`) |
| `webhooks` | Persistent webhook delivery worker |
| `estimate` | Go-live cost estimate and embedded pricing tables |
| `mcpserver` | MCP server over the core service (official Go MCP SDK) |
| `config` | Flags, env and YAML loading with precedence rules |
| `web` | React app source; `web/dist` embedded by `web/embed.go` |
| `examples/` | Real-SDK integration examples per language, run in CI |

HTTP routing uses `go-chi/chi`.

### 5.2 Boundaries

- **Adapters never touch the store.** They translate provider requests into `core` calls, and `core` results or errors into provider responses.
- **Core never knows provider formats.** Each message records its `provider`. The webhook worker asks that provider's adapter (via the adapter registry) to format status and inbound webhooks.
- **Only `store` and `config` touch the filesystem.** There is no package-level mutable state. Every store method takes a `projectID`.

## 6. Domain model

### 6.1 Entities

| Entity | Fields |
|---|---|
| **Project** | `id`, `name`, `settings` (per-adapter webhook URLs, Twilio auth token, sender-ID allow-list, sim-rule overrides), `created_at` |
| **Credential** | `provider`, `key`, `project_id`. Maps a credential to a project |
| **Message** | `id`, `project_id`, `batch_id?`, `channel` (`sms`/`email`), `direction` (`outbound`/`inbound`), `provider` (`native`/`twilio`/`termii`/`smtp`/…), `provider_ref`, `from`, `to`, `cc[]`, `bcc[]`, `subject`, `body_text`, `body_html`, `raw_blob_id?`, `encoding` (`gsm7`/`ucs2`), `segments`, `status`, `error_code?`, `error_message?`, `callback_url?`, `extracted_codes[]`, `extracted_links[]`, `primary_link?`, `created_at`, `updated_at` |
| **StatusEvent** | `message_id`, `status`, `error_code?`, `at` |
| **Batch** | `id`, `project_id`, `provider`, `channel`, `total`, per-status counts, `created_at` |
| **Verification** | `id`, `project_id`, `provider`, `provider_ref`, `service_ref?`, `to`, `channel`, `code`, `status` (`pending`/`approved`/`canceled`/`expired`/`max_attempts`), `attempts`, `max_attempts`, `expires_at`, `message_id`, `created_at` |
| **Unsubscribe** | `project_id`, `number`, `at` (set by STOP, cleared by START) |
| **Attachment** | `id`, `message_id`, `filename`, `content_type`, `size`, `blob_id`, `inline_cid?` |
| **WebhookDelivery** | `id`, `project_id`, `message_id?`, `verification_id?`, `kind` (`status`/`inbound`), `url`, `payload`, `headers`, `attempt`, `status` (`pending`/`succeeded`/`failed`), `response_status?`, `response_body?`, `next_retry_at?` |
| **RequestLog** | `id`, `project_id`, `adapter`, `method`, `path`, `request_headers`, `request_body`, `response_status`, `response_body`, `duration_ms`, `created_at` |

IDs are prefixed ULIDs (`prj_`, `msg_`, `bat_`, `vrf_`, `whd_`, `req_`). Adapters also generate IDs in their provider's format (Twilio `SM`/`VE` + 32 hex characters; Termii numeric message IDs and UUID `pinId`s) and store them in `provider_ref`.

Canonical message statuses: `queued`, `sent`, `delivered`, `undelivered`, `failed`, `received` (inbound). Adapters map these to provider vocabularies.

### 6.2 Projects and credentials

- In local mode any credential is accepted. The first time a credential is seen, a project is created for it, named after the provider and a masked key (for example `twilio:AC12…9f`). The UI can rename it.
- One app often uses several credentials (for example Twilio for SMS and SMTP for email). These can be **linked to one project** either in `mocksms.yaml` (`projects[].credentials`) or with the UI's "link credential to project" action.
- Credential sources: Twilio Account SID in the path; Termii `api_key` in the body; SMTP AUTH username (no AUTH → `default` project); native API `Bearer` key.

### 6.3 Outbound lifecycle

```
SendMessage ─▶ validate (E.164 per validation.phone, sender ID, encoding/segments, unsubscribed?)
           ─▶ sim rules ──reject──▶ synchronous canonical error
           ─▶ persist status=queued ─▶ lifecycle runner advances on timers:
                 queued ──▶ sent ──▶ delivered
                                └──▶ failed / undelivered   (when a sim rule says so)
              each transition: persist StatusEvent → publish on bus → status webhook (if a URL applies) + SSE
```

- The lifecycle runner is a timer queue owned by `core` and driven by the injected `Clock`. The step delay defaults to 300 ms and is configurable (`0` = instant).
- On startup, the runner resumes every message left in `queued` or `sent`.
- `SendBatch` writes one Batch plus N Messages in a single transaction. Batch counts update as messages advance.

### 6.4 Verification (OTP) flow

- `StartVerification(to, channel, opts)` generates a code. Its length comes from the request or provider (default 6), or `--otp-code` overrides it with a fixed value. The code is sent through the normal `SendMessage` path, so OTP messages appear in the inbox, follow the lifecycle and trigger webhooks.
- Default message text: native `Your verification code is {code}`; Twilio `Your {service friendly name} verification code is: {code}`; Termii uses the request's `message_text` with `pin_placeholder` replaced.
- `CheckVerification` increments `attempts`, compares the code, and moves to `approved`. After `max_attempts` (default 5, or the provider's/request's value) the status becomes `max_attempts`. Past `expires_at` (default 600 s, or the provider's/request's value) the status becomes `expired`.
- Codes are **stored in plaintext by design**. This is a sandbox, and the UI and test API must show them.

### 6.5 Extraction

Runs on every message, outbound and inbound.

- **Codes:** standalone runs of 4–8 digits (also 4–8 uppercase alphanumerics when they sit next to a keyword). Candidates next to `code|otp|pin|verification|token|passcode` (case-insensitive) come first.
- **Links:** every `http(s)` URL in text bodies and every `href` in HTML, deduplicated, in order of appearance. `primary_link` is the first link whose URL or anchor text matches `verify|confirm|activate|magic|token|reset|login|signin`.

## 7. Integration surfaces

### 7.1 Native API (`/api/v1`)

`openapi/openapi.yaml` is the source of truth. The Go server interfaces are generated with `oapi-codegen` (chi, strict server), and the UI's TypeScript types with `openapi-typescript`. The same file is published as the API docs.

**Auth and scoping:** `Authorization: Bearer <any key>` selects the project, creating it if needed. Send endpoints called without a key use the `default` project. Read and test endpoints also accept `?project=<id>` instead of a key; with neither, they search across all projects (local mode only). `DELETE /messages` requires a key or `?project=` and returns `400 validation_error` without one.

**Pagination:** `limit` (default 50, max 200) and `cursor` (the last ULID returned), newest first.

| Group | Endpoint | Behavior |
|---|---|---|
| Send | `POST /sms` | `{from, to: string \| string[], body, callback_url?}`. One recipient returns `201` and the message; several return `202` and the batch |
| | `POST /email` | `{from, to[], cc[], bcc[], subject, text?, html?, attachments[{filename, content_type, content_base64}], callback_url?}` |
| | `POST /verifications` | `{to, channel, code_length?, ttl_seconds?, max_attempts?}` → verification **without** the code |
| | `POST /verifications/{id}/check` | `{code}` → `200 {valid, status}`. A wrong code is not an error |
| Read | `GET /messages` | Filters: `channel`, `to`, `from`, `status`, `batch_id`, `direction`, `since` |
| | `GET /messages/{id}` | Includes the StatusEvent timeline |
| | `GET /messages/{id}/raw` | `.eml` download (email only) |
| | `GET /attachments/{id}`, `GET /batches/{id}`, `GET /verifications`, `GET /verifications/{id}` | |
| Test helpers | `GET /messages/wait?to=&channel=&since=&timeout=` | Long-poll. Returns the first message matching the filters created after `since`, or `408 wait_timeout`. `timeout` default 10 s, max 60 s. `since` defaults to the time the request arrives; tests should record `since` before triggering the flow |
| | `GET /otp/latest?to=&since=` | `{code, source: "verification" \| "extracted", message_id, verification_id?}`. Finds the newest outbound message to `to` that carries a verification or an extracted code. If it belongs to a verification, returns that code (`source: verification`); otherwise its first extracted code (`source: extracted`). `404` if none |
| | `GET /emails/latest?to=&since=` | `{message, codes[], links[], primary_link}`. `404` if none |
| | | For `otp/latest` and `emails/latest`, an omitted `since` means no lower bound |
| | `POST /verifications/{id}/expire` | Forces expiry |
| | `POST /inbound` | `{from, to, body}` simulates an inbound SMS |
| | `DELETE /messages` | Deletes all messages, verifications, logs and webhooks in the project |
| Observe | `GET /requests`, `GET /requests/{id}` | Request inspector |
| | `GET /webhooks`, `POST /webhooks/{id}/replay` | Webhook log and replay |
| | `GET /projects`, `PATCH /projects/{id}`, `POST /projects/{id}/credentials` | Project management and credential linking |
| | `GET /projects/{id}/estimate?volume=…` | Go-live estimate (§8.6) |
| | `GET /events` | SSE: `message.created`, `message.status`, `verification.updated`, `batch.updated`, `request.logged`, `webhook.delivered` |
| Ops | `GET /healthz` (unprefixed) | Readiness check |

**Native webhooks:** status callbacks POST `{"event": "message.status", "message": {…}}` to `callback_url`; inbound SMS POST `{"event": "message.inbound", "message": {…}}` to `native.inbound_url`. When the project has an API key, the request carries `X-Mocksms-Signature: sha256=<hex HMAC-SHA256 of the raw body, keyed with that API key>`; projects without a key send unsigned webhooks.

### 7.2 Error model

Core defines canonical errors (`core.Error{Code, Message, Field}`). The native API returns `{"error": {"code", "message", "field?"}}`. Each adapter's `WriteError` converts them to its provider's format and status.

| Canonical code | HTTP | Twilio code | Typical trigger |
|---|---|---|---|
| `validation_error` | 400 | 21604 etc. (field-specific) | Missing or malformed field |
| `invalid_number` | 400 | 21211 | Not valid E.164, or a sim rule |
| `invalid_sender` | 400 | 21212 | Sender-ID rules |
| `unroutable` | 400 | 21612 | Sim rule (cannot route to number) |
| `not_sms_capable` | 400 | 21614 | Sim rule (number cannot receive SMS) |
| `invalid_address` | 400 | — (email only) | Malformed email address, or a sim rule; SMTP `550` at `RCPT TO` |
| `unsubscribed` | 400 | 21610 | Recipient sent STOP |
| `rate_limited` | 429 | 20429 | Sim rule |
| `provider_unavailable` | 503 | 20503 | Sim rule |
| `verification_not_found` | 404 | 20404 | Unknown, expired or approved verification |
| `max_attempts` | 429 | 60202 | Too many check attempts |
| `internal` | 500 | 20500 | Recovered panic |

**Synchronous failures** reject the request immediately. **Asynchronous failures** accept the request (`201`), then move the message to `failed` or `undelivered` with a delivery error code (Twilio 30003 unreachable handset, 30005 unknown handset, 30007 carrier filtered), reported by status webhook. Sim rules choose which kind applies.

### 7.3 Adapter interface

```go
type Adapter interface {
    Name() string                                       // "twilio"
    Routes(r chi.Router)                                // mounted at /{name}; optionally on a dedicated port at /
    WriteError(w http.ResponseWriter, err *core.Error)  // canonical error → provider error format
}

// Optional capabilities, discovered by type assertion.
type StatusNotifier  interface { StatusWebhook(core.Message, core.StatusEvent, core.Project) (*WebhookRequest, bool) }
type InboundNotifier interface { InboundWebhook(core.Message, core.Project) (*WebhookRequest, bool) }
type ReplyParser     interface { ParseReply(*http.Response) ([]core.OutboundReply, error) }
```

- `adapterkit` wraps every adapter route with panic recovery (rendered by `WriteError`), credential → project resolution, and request logging (bodies capped at 64 KB, credentials masked).
- **Dedicated ports:** `adapters.<name>.port` in config also serves that adapter's handler at `/` on its own port. This is for SDKs that accept only a hostname. Off by default.
- **Per-project adapter settings** cover webhook URLs that real providers set in a dashboard rather than per request. They are set in `mocksms.yaml` or the project settings page.

### 7.4 Wave 1 adapters

#### Twilio (M2)

| Surface | Endpoints | Fidelity notes |
|---|---|---|
| Messages | `POST /2010-04-01/Accounts/{AC}/Messages.json`, `GET …/Messages.json` (filters `To`, `From`, `DateSent`; `page`, `page_size`, `next_page_uri`), `GET …/Messages/{SM}.json` | Form-encoded input; Basic auth `AC…:token` or `SK…:secret`. Response fields match Twilio's: `sid`, `account_sid`, `to`, `from`, `body`, `status`, `num_segments`, `num_media`, `direction: outbound-api`, `price: null`, `price_unit`, `error_code`, `error_message`, `uri`, `api_version`, RFC 2822 dates |
| Verify v2 | `POST /v2/Services`, `POST /v2/Services/{VA}/Verifications`, `POST /v2/Services/{VA}/VerificationCheck` (by `To` or `VerificationSid`), `POST /v2/Services/{VA}/Verifications/{VE}` (`Status=canceled\|approved`), `GET` for each | Any `VA…` SID is accepted and becomes a service with default settings (code length 6, friendly name "mocksms"). ISO 8601 dates. `channel` accepts `sms` and `email` |
| Status webhooks | Per-request `StatusCallback` | Form-encoded `MessageSid`, `MessageStatus`, `SmsSid`, `SmsStatus`, `AccountSid`, `From`, `To`, `ApiVersion`, `ErrorCode?` |
| Inbound webhooks | Project setting `twilio.inbound_url` (stands in for the per-number SMS URL) | Form-encoded `MessageSid`, `AccountSid`, `From`, `To`, `Body`, `NumMedia` |
| TwiML replies | `ReplyParser` | `<Response><Message>` elements become outbound replies in the thread |
| Signatures | `X-Twilio-Signature` | HMAC-SHA1 over URL + sorted params. Key: project setting `twilio.auth_token`, defaulting to the token seen with an `AC…` username. If only an `SK…` key has been seen and no auth token is configured, the API secret is used and the inspector shows a warning |

**Redirecting the SDKs:** Twilio SDKs expose no base-URL setting. We provide a custom-HTTP-client snippet for Node, Python, PHP, Go and C# that rewrites `https://api.twilio.com` and `https://verify.twilio.com` to `http://localhost:4010/twilio`. The two APIs' paths do not overlap, so one prefix serves both. Each snippet lives in `examples/` and runs in CI against the official SDK.

#### Termii (M2)

| Surface | Endpoints | Fidelity notes |
|---|---|---|
| SMS | `POST /api/sms/send` (`to` string or array, up to 100), `POST /api/sms/send/bulk` (up to 10,000), `POST /api/sms/number/send` | `api_key` in the JSON body. Multi-recipient sends map to a core Batch. Response `{message_id, message, balance, user}` (bulk adds `code`) |
| Token | `POST /api/sms/otp/send`, `POST /api/sms/otp/verify`, `POST /api/sms/otp/generate` (in-app; returns the OTP in the response), `POST /api/email/otp/send` | `pin_length`, `pin_attempts`, `pin_time_to_live` (minutes), `pin_placeholder`, `message_text`, `message_type` map onto Verification |
| Delivery webhooks | Project setting `termii.webhook_url` (account-level, as in Termii's dashboard) | |
| Sender ID | Project setting `termii.sender_allowlist` | Empty allow-list (default) accepts every sender, with a warning in the inspector. A non-empty list rejects unlisted senders with `invalid_sender` |

**Redirecting:** set `TERMII_BASE_URL=http://localhost:4010/termii`. Termii already gives each account its own base URL, so most integrations read it from configuration.

**Unverified fidelity:** Termii publishes no OpenAPI spec. Fixtures are built from Termii's current public docs. Anything not confirmed there is listed in `docs/fidelity.md` as unverified until checked against a real Termii account (§13).

#### SMTP (M1)

- Listens on `:1025`. Accepts AUTH PLAIN/LOGIN with any password; the username selects the project. No AUTH goes to the `default` project.
- STARTTLS only with `--smtp-tls`, using a self-signed certificate generated at startup.
- MIME parsed with `enmime`: text, HTML, attachments, inline images (`cid:`). Size limit 25 MB, returned as SMTP 552 if exceeded.
- Synchronous rejections (`invalid_address`, e.g. `*@invalid.mocksms.test`) are SMTP `550` at `RCPT TO`.

### 7.5 Wave 2 adapters (overview only)

| Provider | Faked surface | How developers redirect |
|---|---|---|
| SendGrid | `POST /v3/mail/send` → `202` + `X-Message-Id`; account-level event webhook (JSON array) | `setDefaultRequest('baseUrl', …)` |
| Resend | `POST /emails` → `{id}`; Svix-signed webhooks | SDK base-URL option or env var |
| Vonage | `POST /sms/json`; Verify v2 `POST /v2/verify` | `restHost`/`apiHost`, likely via a dedicated port |
| AWS SNS | Query protocol `Action=Publish` with `PhoneNumber`; XML response; SigV4 accepted without checking | SDK `endpoint` on a dedicated port |

## 8. Developer features

### 8.1 Web inbox

```
┌─────────────┬──────────────────────────┬───────────────────────────────────────────┐
│ ▾ my-app    │ 🔍 filter: to, text…     │ +233 24 123 4567              [Twilio]    │
│   (project) │──────────────────────────│───────────────────────────────────────────│
│ Inbox       │ +233 24 123 4567  · 2s   │  ┌──────────────────────────────┐         │
│  · SMS      │   Your code is 482913    │  │ Your code is 482913  [copy]  │ ✓✓ 14:02│
│  · Email    │ jane@test.dev     · 1m   │  └──────────────────────────────┘         │
│  · OTPs     │   Verify your email      │         ┌───────────────────┐             │
│ Batches     │ Batch bat_01H… 9,812/10k │  14:03  │ STOP              │ (inbound)   │
│ Inspector   │                          │         └───────────────────┘             │
│ Estimate    │                          │  [ Reply as this phone…          ] [Send] │
│ Settings    │                          │  Timeline: queued → sent → delivered      │
└─────────────┴──────────────────────────┴───────────────────────────────────────────┘
```

- **SMS:** one phone-style conversation per number, with status ticks, highlighted OTP and copy button, and a reply box (simulated inbound).
- **Email:** HTML, Text, Source, Headers and Attachments tabs; extracted codes and links as chips. HTML renders in a sandboxed iframe (`sandbox` without `allow-scripts`); remote images are blocked until the user turns them on per message.
- **OTPs:** verifications with status, attempts, code and an expiry countdown.
- **Batches:** progress bars with per-status counts. A batch is one collapsible row in the inbox.
- **Settings:** project name, linked credentials, adapter webhook URLs, sender allow-list, sim-rule overrides, latency and random-failure settings. YAML-defined values are shown read-only.
- **Live updates:** SSE events refresh TanStack Query caches.
- The UI grows by milestone: M1 ships Inbox (SMS, Email) and Settings; M2 adds OTPs and Inspector; M3 adds Batches, reply box and sim settings; M4 adds Estimate.

### 8.2 Webhook worker (M3)

- Persistent queue: `WebhookDelivery` rows in the store with `next_retry_at`. Bus events wake the worker; there is no fixed polling interval.
- Request timeout 10 s. Retries after 1 s, 5 s, 30 s, 2 min and 10 min: 5 attempts in total, configurable. Any 2xx counts as success.
- When running inside a container (detected via `/.dockerenv` or `MOCKSMS_IN_DOCKER=1`), `localhost`/`127.0.0.1` webhook hosts are rewritten to `host.docker.internal`, with a note on the delivery record. Turned off with `--no-docker-host-rewrite`.
- **Replay** re-signs the original payload and sends it as a new delivery record.

### 8.3 Failure simulation (M3)

Rules are evaluated in order; the first match wins.

- **Match:** `to` (exact or glob), `from`, `provider`, `project`, `channel`.
- **Effect:** `reject` (canonical code, synchronous) · `fail_async` (delivery error code; message accepted then fails) · `delay` (ms added to each lifecycle step) · `hang` (hold the HTTP connection up to `hang_ms`, default 60 s, then close) · `rate_limit` (N per minute per project per adapter, sliding window).

Built-in test values:

| Value | Effect |
|---|---|
| `+15005550001` | `reject invalid_number` (matches Twilio's test number) |
| `+15005550002` | `reject unroutable` (21612, as Twilio's test number) |
| `+15005550004` | `reject unsubscribed` (blocked) |
| `+15005550009` | `reject not_sms_capable` (21614, as Twilio's test number) |
| Any valid E.164 number whose last six digits are `99990X` | X=1 `reject invalid_number`; 2 `fail_async undelivered` (30005); 3 `reject rate_limited`; 4 `hang`; 5 `fail_async failed` carrier filtered (30007) |
| `*@invalid.mocksms.test` | `reject invalid_address` (SMTP 550) |
| `*@bounce.mocksms.test` | `fail_async` with `error_code: bounced` |

Custom rules live in `mocksms.yaml` (`sim.rules`). Two global settings, **latency** (ms) and **random failure %**, are also editable in the UI.

### 8.4 Inbound and STOP (M3)

UI reply or `POST /inbound` → `core.ReceiveInbound` → an inbound Message → the adapter's `InboundWebhook` to the project's inbound URL → if the adapter is a `ReplyParser`, the replies it parses are sent with `SendMessage`. If the project has no adapter-specific inbound URL, inbound for native projects goes to `native.inbound_url` in native JSON format.

`STOP`, `STOPALL`, `UNSUBSCRIBE`, `CANCEL`, `END` and `QUIT` (trimmed, case-insensitive) create an Unsubscribe record. `START`, `YES` and `UNSTOP` remove it. Sends to an unsubscribed number fail with `unsubscribed`.

### 8.5 Request inspector (M2)

Every adapter request produces a RequestLog via `adapterkit`. The inspector lists requests (filter by adapter, status, path) and shows the raw request and response side by side. The webhook delivery log, with replay, is a second tab in the same screen.

### 8.6 Go-live estimate (M4)

- Input: the project's observed traffic mix (provider × channel × destination country × average segments × OTP-per-verification ratio) and a **volume the developer enters** (e.g. 10,000 signups per month at 1.3 OTPs each).
- Output: estimated monthly cost per provider, broken down by country.
- Prices come from `estimate/pricing/<provider>.yaml`, embedded in each release, with an `as_of` date shown prominently and an "estimate only" label. Countries without price data show "no price data" rather than a guess.
- Wave 1 coverage: Twilio SMS + Verify and Termii SMS + Token. Email cost estimates arrive with SendGrid/Resend in Wave 2.

### 8.7 MCP server (M4)

- Served at `/mcp` (streamable HTTP) on the main port using the official Go MCP SDK. Setup: `claude mcp add --transport http mocksms http://localhost:4010/mcp`.
- Tools: `list_messages`, `wait_for_message`, `get_latest_otp`, `get_latest_email`, `simulate_inbound`, `list_requests`, `reset_project`. Each calls the core service directly.

### 8.8 CI kit (M4)

- **CLI subcommands** on the same binary: `mocksms otp latest <to>`, `mocksms messages wait --to <to> [--timeout 10s]`, `mocksms reset [--project <id>]`. They talk to a running server (`--url`, default `http://127.0.0.1:4010`).
- **Docker image:** distroless, multi-arch, on GHCR, data at `/data`.
- **GitHub Action** `setup-mocksms`: downloads the release binary, starts it in the background, waits for `/healthz`, and outputs `api-url`, `smtp-host`, `smtp-port` and `twilio-base-url`.
- **Testcontainers modules** for Node, Go and Python: start the image and expose `apiUrl`, `smtpHost`, `smtpPort`, `twilioBaseUrl` and a `waitForOtp(to, {since, timeout})` helper.

## 9. Configuration

Precedence: flags > `MOCKSMS_*` environment variables > `mocksms.yaml` (looked for in the current directory, then `~/.config/mocksms/`) > defaults.

YAML values are read-only in the UI. Settings changed in the UI are stored in the database and apply only to keys the YAML does not define.

| Setting | Default |
|---|---|
| `http.port` / `smtp.port` | `4010` / `1025` |
| `host` | `127.0.0.1` |
| `data_dir` | `%LOCALAPPDATA%\mocksms` (Windows), `~/.local/share/mocksms` (Linux/macOS), `/data` (Docker) |
| `memory` | `false` (`--memory` uses SQLite in memory) |
| `retention.max_messages_per_project` / `retention.max_age` | `10000` / `7d` (whichever is hit first) |
| `lifecycle.step_delay` | `300ms` |
| `otp.fixed_code` | unset |
| `validation.phone` | `valid` (libphonenumber `IsValidNumber`); `possible` (`IsPossibleNumber`, accepts made-up numbers such as `555` ranges); `off` |
| `adapters.<name>.port` | unset |
| `projects[]` | none (`name`, `credentials[]`, `settings`) |
| `sim.rules[]`, `sim.latency_ms`, `sim.failure_rate` | built-ins, `0`, `0` |
| `ui_auth` | unset (`user:pass`) |

## 10. Storage

- SQLite in WAL mode with **one write connection and a read pool**, so bulk sends don't hit `SQLITE_BUSY`. Bulk inserts run in one transaction.
- Migrations are embedded SQL run by `pressly/goose` at startup.
- Raw MIME and attachment bytes go through `BlobStore` (SQLite table locally).
- A prune job runs every minute and enforces retention. Deleting a message also deletes its events, attachments, blobs, webhook deliveries and request logs.

## 11. Security (local mode)

- Binds to loopback by default. Starting with a non-loopback `host` and no `ui_auth` prints a prominent warning at startup and in the UI header.
- **DNS-rebinding protection:** requests whose `Host` header is not on the allow-list (`localhost`, `127.0.0.1`, `[::1]`, the configured host, `host.docker.internal`) are rejected with 421. No CORS headers are sent by default.
- Email HTML is sandboxed (§8.1).
- No outbound network traffic except webhooks to developer-configured URLs. No telemetry.
- Every HTTP response includes `X-Mocksms: 1`, and a startup banner is printed, so a production deployment pointed at the sandbox by mistake is easy to spot.

## 12. Distribution and hosted seams

### 12.1 Distribution

- `goreleaser`: binaries for Windows, macOS and Linux (amd64/arm64), multi-arch Docker image on GHCR, Homebrew tap, Scoop bucket, install script, checksums, cosign signatures.
- The CI build runs the Vite build first, then `go build` embeds `web/dist`. In development, Vite's dev server proxies `/api`, `/mcp` and `/twilio` etc. to the Go server.
- Repository license: Apache-2.0.

### 12.2 Seams for the hosted product (interfaces now, implementations later)

| Interface | Local implementation | Hosted implementation (future, closed source) |
|---|---|---|
| `core.Store` / `core.BlobStore` | SQLite | Postgres / S3-compatible object storage |
| `core.Bus` | In-process | Postgres `LISTEN/NOTIFY` or NATS |
| Lifecycle runner, webhook queue | Timers over SQLite rows | Same logic over Postgres with `SKIP LOCKED`, multiple workers |
| `core.ProjectResolver` (credential → project) | Creates projects automatically | Real API keys, accounts, teams |
| Settings source | YAML + database | Per-tenant database settings |

The hosted product is a separate private Go module with its own `cmd/`, composing these public packages with its own implementations.

## 13. Risks and unverified items

| Risk | Mitigation |
|---|---|
| Provider APIs change and adapters drift | Pinned OpenAPI specs plus a scheduled refresh job that opens a PR when they change; real-SDK tests in CI |
| Termii behaviour not fully documented (verify-failure bodies, error bodies, delivery-report payload, webhook signature header, inbound SMS support) | Listed in `docs/fidelity.md` as unverified; checked against a real Termii account before Termii is marked stable. If Termii has no inbound webhook, the Termii adapter does not implement `InboundNotifier` and inbound uses the native format |
| Twilio redirect requires an SDK-specific snippet | Snippets for five languages, each tested in CI against the official SDK |
| Pricing data goes stale | `as_of` date shown; "estimate only"; tables updated with each release |
| SQLite contention under bulk load | Single writer plus read pool; transactional batch inserts; bulk benchmark tracked in CI |
| Scope: Wave 1 is large | Four milestones, each a usable release |

## 14. Testing strategy

| Layer | What | How |
|---|---|---|
| Unit | `core`, `sim`, `phone`, `extract`, `estimate` | Table-driven Go tests. `core` uses the injected `Clock`, so lifecycle and expiry tests are instant and repeatable |
| Store conformance | Every `Store`/`BlobStore` method | One shared suite in `store/storetest`, run against `store/sqlite` now and against Postgres in the hosted product |
| Adapter golden tests | Request → response for every adapter endpoint | `testdata/<provider>/` fixtures; changing fields (SIDs, dates) normalized; `-update` flag regenerates them |
| Spec contract tests | Twilio responses (Wave 2: SendGrid, Vonage) | Validated with `kin-openapi` against pinned copies of the providers' OpenAPI specs; a scheduled CI job refreshes the specs and opens a PR when they change |
| Real-SDK tests | The drop-in promise | `examples/` run in CI against the built binary: official Twilio Node and Python SDKs send SMS and start/check a verification; the receiving server checks the status webhook with the SDK's own `validateRequest`. Termii via plain HTTP |
| SMTP | Text, HTML, attachments, inline images, AUTH → project, 550 rejects | Go `net/smtp` plus Nodemailer in `examples/` |
| UI E2E | Live arrival, reply → webhook, inspector, batch progress | Playwright smoke tests against the embedded UI |
| Bulk benchmark | 10,000-recipient batch end to end | `go test -bench`; tracked, not a release gate |

CI: GitHub Actions. Go tests with `-race` on Linux, macOS and Windows; real-SDK, SMTP and Playwright tests on Linux.
