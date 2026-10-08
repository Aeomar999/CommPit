# Termii Guide

Use Termii integrations against mocksms instead of Termii's cloud by
overriding the base URL. Nothing is delivered; every message lands in the
local inbox and is inspectable via the test API.

## Redirecting clients

Termii gives each account its own base URL and most integrations read it
from configuration, so switching is a one-line change:

```bash
TERMII_BASE_URL=http://127.0.0.1:4010/termii
```

```python
# Python (requests)
import os, requests
BASE = os.getenv("TERMII_BASE_URL", "http://127.0.0.1:4010/termii")
response = requests.post(f"{BASE}/api/sms/send", json={
    "api_key": os.getenv("TERMII_API_KEY", "test-key"),
    "to": "+2348031234567",
    "from": "MyApp",
    "sms": "Hello from mocksms",
})
print(response.json())
```

The `api_key` in the JSON body selects the project (auto-provisioned on
first use). Any key is accepted locally.

SDKs that accept only a hostname can use a dedicated port instead
(`--termii-port 4021`): the adapter is then served at `/`, so
`http://127.0.0.1:4021/api/sms/send` is the endpoint.

## SMS

- `POST /termii/api/sms/send` — `to` takes one number or an array (up to
  100); arrays map to a batch with per-recipient rejection reporting.
- `POST /termii/api/sms/send/bulk` — up to 10,000 recipients; the response
  adds `code: "ok"`.
- `POST /termii/api/sms/number/send` — same shape, for number senders.

```bash
curl -X POST http://127.0.0.1:4010/termii/api/sms/send \
  -H "Content-Type: application/json" \
  -d '{"api_key":"test-key","to":"+2348031234567","from":"MyApp","sms":"Hello from mocksms"}'
```

Success is HTTP 200 with `{message_id, message, balance, user}`. `type`
and `channel` are accepted and ignored; message IDs are numeric and stored
in `provider_ref` for single sends.

### Sender allow-list

Restrict sender IDs per project (YAML has no sender setting yet — configure
via project settings in the UI, or seed `termii.sender_allowlist`
directly). An empty list accepts every sender with a server warning;
unlisted senders are rejected with `invalid_sender`.

## Token (OTP)

- `POST /termii/api/sms/otp/send` — `to`, `from`, `message_text` with a
  `pin_placeholder` (default `< 1234 >`), `pin_length` (default 6),
  `pin_attempts`, `pin_time_to_live` (minutes). Responds with a UUID `pinId`.
- `POST /termii/api/sms/otp/verify` — `pin_id` + `pin`; wrong PINs return
  200 with `verified: false`, correct ones `verified: true`, exhausted
  attempts are 429.
- `POST /termii/api/sms/otp/generate` — `pin_type: NUMERIC`, `pin_length`;
  returns the PIN for in-app verification. Nothing is sent or stored.
- `POST /termii/api/email/otp/send` — `email_address` + caller-supplied
  `code`, stored on an email-channel verification.

```bash
PIN_ID=$(curl -s -X POST http://127.0.0.1:4010/termii/api/sms/otp/send \
  -H "Content-Type: application/json" \
  -d '{"api_key":"test-key","to":"+2348031234567","from":"MyApp","message_text":"Your pin is < 1234 >"}' \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['pinId'])")
curl -s -X POST http://127.0.0.1:4010/termii/api/sms/otp/verify \
  -H "Content-Type: application/json" \
  -d "{\"api_key\":\"test-key\",\"pin_id\":\"$PIN_ID\",\"pin\":\"000000\"}"
```

## Errors

Failures are HTTP-sized with `{message, code}` where `code` is the
canonical code (`invalid_number`, `unsubscribed`, `validation_error`, …).
See `docs/fidelity.md` for behaviors still unverified against a real Termii
account (response wordings, caps, channel semantics) — they will be
confirmed in X-01.
