# Failure Simulation Guide

mocksms answers like a provider on a bad day: magic recipients fail on
purpose, and custom rules plus two global knobs shape delivery. Use them
to exercise retries, error banners, opt-out flows and timeout handling
without touching production.

## Magic values

Built-in test recipients, always on:

| Send to | Effect |
|---|---|
| `+15005550001` | Rejected: `invalid_number` (Twilio 21211) |
| `+15005550002` | Rejected: `unroutable` (Twilio 21612) |
| `+15005550004` | Rejected: `unsubscribed` (Twilio 21610); send `START` to resubscribe |
| `+15005550009` | Rejected: `not_sms_capable` (Twilio 21614) |
| Any valid E.164 ending `999901` | Rejected: `invalid_number` |
| Any valid E.164 ending `999902` | Accepted, then `undelivered` with code `30005` (unknown handset) |
| Any valid E.164 ending `999903` | Rejected: `rate_limited` (Twilio 20429) |
| Any valid E.164 ending `999904` | Request hangs 60 s (then completes) |
| Any valid E.164 ending `999905` | Accepted, then `failed` with code `30007` (carrier filtered) |
| Any valid E.164 ending `999906` | Accepted, then `undelivered` with code `30003` (unreachable handset) |
| `*@invalid.mocksms.test` | Email rejected: `invalid_address` (SMTP 550) |
| `*@bounce.mocksms.test` | Email accepted, then `failed` with code `bounced` |

Async failures move the message after acceptance and report through the
status webhook (`ErrorCode` included for Twilio and native callbacks).

STOP-family inbound (`STOP`, `STOPALL`, `UNSUBSCRIBE`, `CANCEL`, `END`,
`QUIT` — trimmed, case-insensitive) unsubscribes the sender; later sends
fail with `unsubscribed`. `START`, `YES` and `UNSTOP` resubscribe.

## Custom rules (`sim.rules`)

Custom rules live in `mocksms.yaml` and run before the built-ins, so
they win on overlap:

```yaml
sim:
  latency: 300ms        # added to every lifecycle step
  failure_rate: 0.0     # share of sends that fail after acceptance
  rules:
    - match: { to: "+15551234567" }
      effect: { reject: { code: invalid_number, message: "blocked" } }
    - match: { provider: twilio, channel: sms }
      effect: { delay_ms: 250, rate_limit_per_min: 60 }
    - match: { to: "*@example.test" }
      effect: { fail_async: { code: bounced, message: "gone", status: failed } }
```

Match fields (all optional, combined with AND): `to`, `from`
(`*`-suffix wildcards supported, e.g. `*@example.test`),
`provider` (`twilio`, `termii`, `smtp`, `native`), `project`
(project ID), `channel` (`sms` or `email`).

Effects:

| Key | Meaning |
|---|---|
| `reject: {code, message}` | Synchronous canonical error. Codes: `invalid_number`, `invalid_sender`, `unroutable`, `not_sms_capable`, `invalid_address`, `unsubscribed`, `rate_limited`, `provider_unavailable`, `validation_error` |
| `fail_async: {code, message, status?}` | Accepted, then terminal with the code. `status` is `failed` or `undelivered` (default `undelivered`) |
| `delay_ms` | Milliseconds added to each lifecycle step |
| `hang_ms` | Milliseconds the request blocks before responding (cancel-safe) |
| `rate_limit_per_min` | Sends per minute per project per provider (sliding window); over-quota sends fail with `rate_limited`. Allowed sends consume quota, rejected ones do not |

`reject` wins when several effects are set. Bad codes, channels or
negative durations fail startup with the rule index. Every rule needs at
least one effect.

## Runtime settings

Latency and random failure are also editable live — the Settings page or
the API (applied immediately, reset to YAML values on restart):

```bash
curl http://127.0.0.1:4010/api/v1/sim/config -H "X-Mocksms: true"
# {"latency_ms":0,"failure_rate":0}

curl -X PUT http://127.0.0.1:4010/api/v1/sim/config \
  -H "X-Mocksms: true" -H "Content-Type: application/json" \
  -d '{"latency_ms":250,"failure_rate":0.05}'
```
