# Twilio Guide

Use the official Twilio SDKs against mocksms instead of Twilio's cloud.
Nothing is delivered; every message lands in the local inbox, runs through
a realistic lifecycle, and is inspectable via the test API.

## Redirecting the SDKs

Twilio SDKs expose no base-URL setting. Rewrite `https://api.twilio.com`
and `https://verify.twilio.com` to `http://127.0.0.1:4010/twilio` with a
custom HTTP client. The two APIs' paths do not overlap, so one prefix serves
both Messages and Verify. Ready-made snippets live in `examples/twilio/`
(Node, Python, PHP, Go, C#) and run in CI against this binary:

```js
// Node: examples/twilio/node/send-sms.js
const client = twilio(accountSid, authToken, { httpClient: new MocksmsClient() });
```

```python
# Python: examples/twilio/python/send_sms.py
client = Client(account_sid, auth_token, http_client=MocksmsClient())
```

```php
// PHP: examples/twilio/php/send_sms.php
$client = new Client($sid, $token, $sid, null, new MocksmsClient($base));
```

```go
// Go: examples/twilio/go/send-sms.go
custom := &mocksmsClient{Client: client.Client{Credentials: client.NewCredentials(sid, token)}}
restClient := twilio.NewRestClientWithParams(twilio.ClientParams{Client: custom})
```

```csharp
// C#: examples/twilio/csharp/Program.cs
var restClient = new TwilioRestClient(sid, token, sid, httpClient: new MocksmsHttpClient(base));
```

Any Account SID and token are accepted in local mode; the first use of a
credential provisions a project for it. Link several credentials to one
project in `mocksms.yaml` or on the Settings page (see below).

SDKs that accept only a hostname can use a dedicated port instead
(`--twilio-port 4020`): the adapter is then served at `/`, so
`http://127.0.0.1:4020` replaces the `.../twilio` prefix.

## Messages

`POST /twilio/2010-04-01/Accounts/{AC}/Messages.json` (form-encoded),
`GET …/Messages.json` (`To`, `From`, `DateSent` filters; `PageSize`, `Page`
paging), `GET …/Messages/{SM}.json`. Responses match Twilio's shape (`sid`,
`status`, `num_segments`, `direction: outbound-api`, RFC 2822 dates).

```bash
curl -u ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:test-token \
  -d "To=%2B15005550006" \
  -d "From=%2B15555550100" \
  -d "Body=Hello from mocksms" \
  http://127.0.0.1:4010/twilio/2010-04-01/Accounts/ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/Messages.json
```

`StatusCallback` is accepted and stored; delivery starts with the M3
webhooks worker.

## Verify v2

`POST /twilio/v2/Services` (`FriendlyName`, `CodeLength`),
`POST …/Services/{VA}/Verifications` (`To`, `Channel=sms|email`),
`POST …/VerificationCheck` (by `To` or `VerificationSid` plus `Code`),
`POST …/Verifications/{VE}` (`Status=canceled|approved`), plus `GET` for
services and verifications. Any `VA…` SID is accepted and becomes a service
with defaults (friendly name `mocksms`, 6-digit codes). Dates are ISO 8601.

```bash
VA=$(curl -s -u ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:test-token \
  -d "FriendlyName=my-app" \
  http://127.0.0.1:4010/twilio/v2/Services | python3 -c "import json,sys; print(json.load(sys.stdin)['sid'])")
curl -s -u ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX:test-token \
  -d "To=%2B15005550006" -d "Channel=sms" \
  http://127.0.0.1:4010/twilio/v2/Services/$VA/Verifications
```

The SMS text uses the service friendly name
(`Your my-app verification code is: 123456`).

## Magic test numbers

| To | Behavior |
|---|---|
| `+15005550001` | Rejected: `invalid_number` (21211) |
| `+15005550002` | Rejected: `unroutable` (21612) |
| `+15005550004` | Rejected as unsubscribed (21610); send `START` to resubscribe |
| `+15005550009` | Rejected: `not_sms_capable` (21614) |
| `…999901` … `…999905` suffix | Pattern rules (invalid, async-fail, rate-limit, hang, carrier-filter) |

See the simulator rules in `mocksms.yaml` (`sim.rules`) for custom behavior.

## Error mapping

Canonical errors map to Twilio codes: `invalid_number` → 21211,
`invalid_sender` → 21212, `unsubscribed` → 21610, `unroutable` → 21612,
`not_sms_capable` → 21614, missing credentials → 20003 (401), unknown
resources → 20404 (404), Verify validation → 60200, exhausted checks →
60202 (429). Every error carries `{code, message, more_info, status}`.

## Credential linking

One app often uses several credentials. Map them onto a single project in
`mocksms.yaml`:

```yaml
projects:
  - id: prj_01JAAAAAAAAAAAAAAAAAAAAAAAAA
    name: combined
    credentials:
      - provider: twilio
        key: ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
      - provider: smtp
        key: app-user
```

…or call `POST /api/v1/projects/{id}/credentials` (`{"provider", "key"}`)
or use the Settings page action. See `docs/fidelity.md` for behaviors that
are still unverified against Twilio's docs.
