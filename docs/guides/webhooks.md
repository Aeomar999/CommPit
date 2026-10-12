# Webhooks Guide

mocksms delivers the same webhooks the real providers send, signed the
same way, so your callback handlers and signature checks run unmodified.
Nothing is delivered; every webhook POSTs to a URL you configure.

## Status callbacks

Every message transition (`queued → sent → delivered`, or `failed` /
`undelivered`) triggers a callback when the message has a destination:

| Provider | Destination | Format |
|---|---|---|
| Twilio | Per-request `StatusCallback` form field | Form-encoded, `X-Twilio-Signature` |
| Termii | Account-level `termii.webhook_url` project setting | JSON delivery report, `X-Termii-Signature` |
| Native | Per-request `callback_url` | JSON event, `X-Mocksms-Signature` |

### Twilio

Pass `StatusCallback` when creating the message. Each transition POSTs
`MessageSid`, `MessageStatus`, `SmsSid`, `SmsStatus`, `AccountSid`,
`From`, `To`, `ApiVersion` (plus `ErrorCode` on failures):

```bash
curl -X POST http://127.0.0.1:4010/twilio/2010-04-01/Accounts/ACxxx/Messages.json \
  -u ACxxx:token \
  -d "To=+15005550006&From=+15555550100&Body=Hi&StatusCallback=http://127.0.0.1:3000/twilio-status"
```

Verify the signature with the official SDK (it recomputes HMAC-SHA1 over
the full URL plus sorted parameters):

```python
# Python (twilio)
from twilio.request_validator import RequestValidator
validator = RequestValidator("token")  # your auth token
valid = validator.validate(url, request.form, request.headers["X-Twilio-Signature"])
```

```javascript
// Node (twilio)
const twilio = require("twilio");
const valid = twilio.validateRequest("token", req.headers["x-twilio-signature"], url, req.body);
```

The signing key is your explicit `twilio.auth_token` project setting,
else the token seen with an `AC…` username, else the API secret (the
server logs a warning for the last two cases). With no known key the
callback goes out unsigned.

### Termii

Set one webhook URL per project (as in Termii's dashboard) — Termii has
no per-request callback. Seed it via project settings:

```bash
# deliveries of every Termii message in the project POST here
# project setting: termii.webhook_url = http://127.0.0.1:3000/termii-dlr
```

Each transition POSTs the JSON delivery report (`type`, `id`,
`message_id`, `receiver`, `sender`, `message`, `sent_at`, `cost`,
`status`, `channel`) with `Content-Type: application/json`. `status` is
`Message Sent`, `Delivered` or `Message Failed`; `cost` is always null
(the sandbox has no billing).

Verify `X-Termii-Signature` (hex HMAC-SHA512 over the exact JSON bytes,
keyed by your `api_key`):

```python
import hmac, hashlib
expected = hmac.new(b"api-key", request.data, hashlib.sha512).hexdigest()
valid = hmac.compare_digest(expected, request.headers["X-Termii-Signature"])
```

### Native

Pass `callback_url` on `/sms`, `/email` or batch sends. Each transition
POSTs `{"event": "message.status", "message": {…}}` (the full stored
message) as JSON:

```bash
curl -X POST http://127.0.0.1:4010/api/v1/sms \
  -H "X-Mocksms: true" -H "Content-Type: application/json" \
  -d '{"from":"+15555550100","to":"+15005550006","body":"Hi","callback_url":"http://127.0.0.1:3000/hook"}'
```

Verify `X-Mocksms-Signature: sha256=<hex HMAC-SHA256 of the raw body,
keyed with the project's API key>`:

```python
import hmac, hashlib
digest = hmac.new(b"api-key", request.data, hashlib.sha256).hexdigest()
valid = hmac.compare_digest(f"sha256={digest}", request.headers["X-Mocksms-Signature"])
```

Projects without an API key send unsigned webhooks.

## Inbound webhooks

Simulate a handset reply in the inbox reply box or via the API:

```bash
curl -X POST http://127.0.0.1:4010/api/v1/inbound \
  -H "X-Mocksms: true" -H "Content-Type: application/json" \
  -d '{"from":"+15005550006","to":"+15555550100","body":"STOP"}'
```

The inbound message POSTs to the project's inbound URL in provider
format — `twilio.inbound_url` (form-encoded, standing in for the
per-number SMS URL) or `native.inbound_url` (`{"event":
"message.inbound", "message": {…}}`, signed like status callbacks).
Termii has no inbound webhook; Termii-provider inbound is not sent.

If your app answers a Twilio inbound webhook with TwiML, each
`<Response><Message>` becomes an outbound reply in the thread:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response>
  <Message to="+15005550006">Thanks, we got it.</Message>
</Response>
```

Messages without a `to` attribute go back to the inbound sender.

## Retries and replay

A failing receiver (connection error or non-2xx) is retried after 1 s,
5 s, 30 s, 2 min and 10 min — six sends max (`webhooks.timeout` /
`webhooks.max_attempts` tune this). Every attempt is recorded with its
response. Re-send any delivery from the Inspector's Webhooks Log tab or
the API — replay re-signs the original payload with current keys and
queues a fresh record:

```bash
curl -X POST http://127.0.0.1:4010/api/v1/webhooks/whd_XXX/replay \
  -H "X-Mocksms: true"
```

## Docker

Inside a container (detected via `/.dockerenv` or
`MOCKSMS_IN_DOCKER=1`), `localhost` / `127.0.0.1` webhook hosts are
rewritten to `host.docker.internal` at send time so callbacks reach your
app on the host; the delivery record carries a note. Off with
`--no-docker-host-rewrite`. Note: signatures still cover the configured
`localhost` URL, so validate SDK signatures against that URL, not the
received host.
