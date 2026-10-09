# mocksms — Architecture

This is the **living** architecture document. Keep it in step with the code. The [design spec](superpowers/specs/2026-10-03-mocksms-design.md) is the frozen record of the original design; where they differ, this file describes the current system and records why in the decision log (§10).

## 1. System context

```
   Developer's app ──────── Twilio SDK / Termii HTTP / SMTP / native REST ─────┐
   Developer's tests ─────── test API (wait, otp/latest, emails/latest) ───────┤
   AI coding agent ───────── MCP (/mcp) ───────────────────────────────────────┤
   Developer's browser ───── web inbox (/) + SSE (/api/v1/events) ─────────────┤
                                                                                ▼
                                                                   ┌────────────────────┐
                                                                   │      mocksms       │
                                                                   └─────────┬──────────┘
                                                                             │
   Developer's app ◀──── status callbacks, inbound SMS (provider formats) ───┘
```

mocksms never contacts real providers. Its only outbound traffic is webhooks to URLs the developer configured.

## 2. Process view

One OS process. Long-running goroutines:

| Goroutine | Started by | Job |
|---|---|---|
| HTTP server(s) | `cmd/mocksms` | Main port (`:4010`) plus any dedicated adapter ports |
| SMTP server | `smtpd` | `:1025` listener |
| Lifecycle runner | `core` | Timer queue that advances message statuses |
| Webhook worker | `webhooks` | Delivers pending `WebhookDelivery` rows with retries |
| SSE hub | `api` | Fans bus events out to connected browsers |
| Prune job | `cmd/mocksms` | Enforces retention every minute |

```
                     ┌──────────────────── mocksms (single Go binary) ─────────────────────┐
 your app ──HTTP───▶ │ :4010 router                                                         │
                     │   /api/v1/*     native API ──────┐                                   │
                     │   /twilio/*     adapter ─────────┤                                   │
                     │   /termii/*     adapter ─────────┼──▶ core service ───▶ Store        │
                     │   /mcp          MCP server ──────┤      │   ▲  (Simulator) (SQLite)  │
 your app ──SMTP───▶ │ :1025 SMTP listener ─────────────┘      │   └──                      │
                     │                                         ▼                            │
                     │                                       Bus                            │
                     │                                   ┌─────┴──────┐                     │
                     │                          webhook worker     SSE hub ──▶ web inbox (embedded)
                     └────────────────────────────────┼─────────────────────────────────────┘
                                                      ▼
                          status callbacks / inbound SMS ──▶ your app's webhook URL
```

## 3. Packages and dependency rules

Ports-and-adapters layout: `core` owns the domain types **and the interfaces it consumes** (its "ports"). Everything else depends on `core`; `core` depends only on two leaf packages. Extension packages are public (not `internal/`) so the closed hosted product can compose them.

| Package | Responsibility | May import (project packages) |
|---|---|---|
| `phone` | E.164 parsing/validation modes, GSM-7/UCS-2 detection, segment counting | none |
| `extract` | OTP-code and link extraction | none |
| `core` | Domain types, canonical errors, service (`SendMessage`, `SendBatch`, `StartVerification`, `CheckVerification`, `ReceiveInbound`), lifecycle runner, ports: `Store`, `BlobStore`, `Bus`, `Simulator`, `Clock`, `ProjectResolver` | `phone`, `extract` |
| `store/sqlite` | Implements `core.Store` and `core.BlobStore`; embedded migrations | `core` |
| `store/storetest` | Conformance suite any `core.Store` implementation must pass | `core` |
| `bus` | In-process implementation of `core.Bus` | `core` |
| `sim` | Implements `core.Simulator` (rule engine, built-in magic values) | `core`, `phone` |
| `api` | Native REST API (generated from `openapi/openapi.yaml`), test helpers, SSE hub, security middleware | `core` |
| `adapters/adapterkit` | Shared adapter middleware and helpers | `core` |
| `adapters/twilio`, `adapters/termii` | Provider translation | `core`, `adapters/adapterkit`, `phone` |
| `smtpd` | SMTP listener and MIME parsing | `core` |
| `webhooks` | Delivery worker; defines its own `Formatters` interface for looking up adapter webhook formatters | `core` |
| `estimate` | Go-live estimate and pricing tables | `core`, `phone` |
| `mcpserver` | MCP tools over the core service | `core` |
| `config` | Flags, env, YAML loading and precedence | none |
| `web` | React app; `web/embed.go` exposes `web/dist` | none |
| `cmd/mocksms` | Composition root and CLI subcommands | everything |

**Enforced in CI** with `golangci-lint`'s `depguard` rules mirroring this table. Adapters importing `store/...`, or `core` importing anything except `phone` and `extract`, fails the build.

**Other hard rules:**
- `core` contains no HTTP code and no provider-specific formats.
- Adapters never touch a store. They call the core service and translate results.
- Every `Store` method takes a `projectID`.
- No package-level mutable state; dependencies are passed in by `cmd/mocksms`.
- Only `store/sqlite` and `config` touch the filesystem.

## 4. Domain model

Full field lists are in the spec §6.1. Relationships:

```
Project 1───* Credential
Project 1───* Message *───0..1 Batch
Message 1───* StatusEvent
Message 1───* Attachment ───1 Blob
Message 0..1─1 Verification          (the message that carried the code)
Message 1───* WebhookDelivery
Project 1───* RequestLog
Project 1───* Unsubscribe
```

- IDs: prefixed ULIDs (`prj_`, `msg_`, `bat_`, `vrf_`, `whd_`, `req_`, `att_`, `blob_`, `sev_`). Provider-format IDs (`SM…`, `VE…`, Termii `pinId`) live in `provider_ref`.
- Canonical statuses: `queued`, `sent`, `delivered`, `undelivered`, `failed`, `received`.
- Canonical errors: `validation_error`, `invalid_number`, `invalid_sender`, `unroutable`, `not_sms_capable`, `invalid_address`, `unsubscribed`, `rate_limited`, `provider_unavailable`, `verification_not_found`, `max_attempts`, `internal`.

## 5. Key runtime flows

### 5.1 Send SMS through the Twilio adapter

```
App (Twilio SDK + redirect snippet)
  │ POST /twilio/2010-04-01/Accounts/AC…/Messages.json  (form, Basic auth)
  ▼
adapterkit: recover → resolve project from AC… → start RequestLog
  ▼
adapters/twilio: parse form → core.SendMessage(ctx, projectID, SendRequest{…, Provider: "twilio", CallbackURL})
  ▼
core: phone.Validate → unsubscribed? → Simulator.Evaluate
      ├─ reject → *core.Error → twilio.WriteError → 400 {code: 21211, …}
      └─ accept → Store.CreateMessage(status=queued) → Bus.Publish(message.created) → lifecycle.Schedule
  ▼
adapters/twilio: render Twilio JSON (sid SM…, status "queued", num_segments, RFC 2822 dates) → 201
  ▼
adapterkit: finish RequestLog → Bus.Publish(request.logged)
```

### 5.2 Lifecycle and status webhooks

```
lifecycle runner (Clock timer fires)
  → Store.AppendStatusEvent(sent) → Bus.Publish(message.status)
      ├─ SSE hub → browser updates the tick
      └─ webhooks worker: message has callback_url?
           → Formatters.For("twilio").StatusWebhook(msg, event, project)
           → Store.CreateWebhookDelivery(pending) → POST with X-Twilio-Signature
           → record attempt; on failure set next_retry_at (1s, 5s, 30s, 2m, 10m)
  → later: delivered (or failed/undelivered per Simulator decision)
```

### 5.3 Verification

```
StartVerification → generate code (length, or otp.fixed_code)
                 → SendMessage(body with code) → Verification{status: pending, message_id}
CheckVerification → expired? → status expired → verification_not_found
                 → attempts++ → code matches? → approved
                 → attempts == max_attempts? → status max_attempts
```

### 5.4 Inbound with TwiML reply

```
UI reply box → POST /api/v1/inbound → core.ReceiveInbound
  → STOP/START keyword? → Store.Unsubscribe / Resubscribe
  → Message{direction: inbound, status: received}
  → webhooks: Formatters.For(provider).InboundWebhook → POST to project inbound URL
  → response body → ReplyParser (TwiML) → []OutboundReply → core.SendMessage for each
```

### 5.5 SMTP

```
SMTP client → AUTH (username → project) → MAIL/RCPT (Simulator may 550 at RCPT)
  → DATA → enmime parse → BlobStore.Put(raw) → core.SendMessage(channel: email, …)
```

### 5.6 Native webhooks

Native-API messages send JSON webhooks: `{"event": "message.status", "message": {…}}` to `callback_url`, and `{"event": "message.inbound", "message": {…}}` to `native.inbound_url`. When the project has an API key, they are signed with `X-Mocksms-Signature: sha256=<hex HMAC-SHA256 of the raw body, keyed with that API key>`; projects without a key send unsigned webhooks.

## 6. Storage

- SQLite (`modernc.org/sqlite`, no CGO) in WAL mode.
- **One write connection**, a pool of read connections. All writes go through the write connection, which removes `SQLITE_BUSY` under bulk load.
- `SendBatch` inserts the batch and all N messages in one transaction.
- Raw MIME and attachments are stored through `BlobStore` (a SQLite table locally).
- Migrations: embedded SQL files run by `goose` at startup; forward-only.
- `--memory` uses an in-memory SQLite database (for CI).
- Retention: a prune job deletes messages beyond 10,000 per project or older than 7 days, cascading to events, attachments, blobs, webhook deliveries and request logs.

## 7. Startup and shutdown

**Startup order:**
1. Load config (flags > env > YAML > defaults).
2. Open the store and run migrations.
3. Build the core service with its ports (store, bus, simulator, clock, resolver).
4. Resume lifecycles for messages in `queued` or `sent`.
5. Start the webhook worker and the prune job.
6. Start the SMTP listener.
7. Start the HTTP server(s).
8. Print the banner (URLs, data directory, warnings).

**Shutdown** (SIGINT/SIGTERM):
1. Stop accepting new HTTP and SMTP connections; drain in-flight requests (up to 5 s).
2. Stop the lifecycle runner. State is persisted and resumes on next start.
3. Stop the webhook worker. The current attempt completes or times out; pending rows stay in the store.
4. Close the store.

## 8. Security architecture

| Control | Where |
|---|---|
| Loopback bind by default | `config`, `cmd/mocksms` |
| `Host` header allow-list (DNS-rebinding protection), 421 on mismatch | `api` middleware, applied to every HTTP handler |
| No CORS headers by default | `api` middleware |
| Optional UI basic auth (`ui_auth`) | `api` middleware |
| `X-Mocksms: 1` on every response | `api` middleware |
| Sandboxed email HTML iframe; remote images opt-in | `web` |
| Credential masking in request logs | `adapters/adapterkit` |

## 9. Seams for the hosted product

| Port | Local implementation | Hosted implementation (future, closed source) |
|---|---|---|
| `core.Store`, `core.BlobStore` | `store/sqlite` | Postgres, S3-compatible storage |
| `core.Bus` | `bus` (in-process) | Postgres `LISTEN/NOTIFY` or NATS |
| Lifecycle runner, webhook queue | Timers over SQLite rows | Same logic over Postgres with `SKIP LOCKED`, multiple workers |
| `core.ProjectResolver` | Creates projects from credentials | Real API keys, accounts, teams |
| Settings | YAML + database | Per-tenant database settings |

The hosted product is a separate private Go module with its own `cmd/`. It reuses `store/storetest` to prove its Postgres store behaves like SQLite.

## 10. Decision log

| # | Date | Decision | Why |
|---|---|---|---|
| ADR-001 | 2026-10-03 | Local-first single binary, hosted later | Fastest path to users; no auth/billing in Wave 1 |
| ADR-002 | 2026-10-03 | Native core + provider adapters as translators | Real SDKs keep working; adding a provider never touches core |
| ADR-003 | 2026-10-03 | Go backend, React/TS UI embedded via `go:embed` | Single static binary, like Mailpit |
| ADR-004 | 2026-10-03 | Modular monolith over separate services | Local tool; separate services add install pain for no gain |
| ADR-005 | 2026-10-03 | Provider OpenAPI specs for contract tests only, not generated mocks | Generated mocks are stateless; tests still get accuracy |
| ADR-006 | 2026-10-03 | Public top-level packages, no `internal/` for extension points | Open-core: the private hosted module must import them |
| ADR-007 | 2026-10-03 | Ports (`Store`, `Bus`, `Simulator`, …) defined in `core`; `store/storetest` holds the conformance suite | Avoids an import cycle between `core` and `store`; idiomatic consumer-defined interfaces. Refines spec §5.1 |
| ADR-008 | 2026-10-03 | Native webhooks are JSON signed with HMAC-SHA256 using the project's API key | The spec left the native webhook format undefined |
| ADR-009 | 2026-10-08 | Adapterkit middleware layer with streaming body duplication and ingress credential redaction | Protects inspector memory with strict 64 KB caps while preserving complete downstream request bodies, redacting secrets before storage/bus publication |
| ADR-010 | 2026-10-08 | Native API served only under spec-canonical `/api/v1`; `/` serves the embedded web UI (SPA fallback) | The UI, `openapi.yaml` servers and the vite proxy all use `/api/v1`, while M1 had mounted the handlers at root (every UI call 404'd). A both-prefixes alias was staged first, then replaced by the UI mount once the embed wiring landed; pre-1.0 root-path clients migrate to `/api/v1` |
