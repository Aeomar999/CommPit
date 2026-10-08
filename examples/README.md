# examples/

Runnable integration examples. Each Twilio snippet points an official
provider SDK at a local mocksms sandbox and exercises SMS + Verify end to
end. They serve as copy-paste redirect documentation and as the M2-06 CI
e2e leg (`go test -tags e2e ./examples/...` against the built binary).

## Layout

- `twilio/node/send-sms.js` (`package.json`) — twilio-node + custom `httpClient`
- `twilio/python/send_sms.py` (`requirements.txt`) — twilio-python + `TwilioHttpClient` subclass
- `twilio/php/send_sms.php` (`composer.json`) — twilio/sdk + `Twilio\Http\Client` wrapper
- `twilio/go/send-sms.go` (`go.mod`, own module) — twilio-go + `SendRequest` override
- `twilio/csharp/Program.cs` (`mocksms.csproj`) — Twilio SDK + `Twilio.Http.HttpClient` subclass
- `twilio_snippets_test.go` (`//go:build e2e`) — runs every snippet against a live binary

Every snippet rewrites `api.twilio.com` and `verify.twilio.com` to
`<MOCKSMS_URL>/twilio`, then creates a Verify service, sends an SMS,
starts a verification and checks it. Each prints `E2E-OK <lang>` and exits
non-zero with a `FAIL:` line on any mismatch.

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `MOCKSMS_URL` | `http://127.0.0.1:4010` | Running mocksms server |
| `TWILIO_ACCOUNT_SID` | `ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX` | Any string; local mode accepts everything |
| `TWILIO_AUTH_TOKEN` | `testtoken123` | Must stay alphanumeric: twilio-go rejects anything else client-side |
| `TO_NUMBER` | per-language `+1500555001x` | Distinct recipients so parallel runs never collide |
| `FROM_NUMBER` | `+15555550100` | Sender |
| `OTP_CODE` | none (required) | Must match the server's `--otp-code` flag |

Never commit real credentials here. The `ACXXX…` placeholder and the
alphanumeric test token are intentionally push-protection safe.

## Run locally

```bash
go build -o bin/mocksms ./cmd/mocksms
bin/mocksms --memory --otp-code 123456 &
cd examples/twilio/node && npm install && cd ../..
cd examples/twilio/python && pip install -r requirements.txt && cd ../..
# PHP / .NET need runtimes the harness skips when absent:
#   composer install (examples/twilio/php), dotnet restore (examples/twilio/csharp)
OTP_CODE=123456 go test -v -tags e2e ./examples/...
```

Snippets for missing runtimes report `SKIP`, never failure.
