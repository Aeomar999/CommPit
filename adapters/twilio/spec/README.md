# Twilio pinned specs

Trimmed copies of the upstream [twilio-oai](https://github.com/twilio/twilio-oai)
OpenAPI specs, covering only the endpoints mocksms implements. Retrieved
2026-10-08 from:

- `https://raw.githubusercontent.com/twilio/twilio-oai/main/spec/json/twilio_api_v2010.json`
- `https://raw.githubusercontent.com/twilio/twilio-oai/main/spec/json/twilio_verify_v2.json`

Regenerate with `adapters/twilio/spectrim` (kept endpoint paths are listed in
`.github/workflows/spec-refresh.yml`, which re-downloads upstream weekly and
opens a PR when the trimmed output changes):

```bash
go run ./adapters/twilio/spectrim \
  -in /tmp/twilio_api_v2010.json \
  -out adapters/twilio/spec/twilio_api_v2010.json \
  -paths /2010-04-01/Accounts/{AccountSid}/Messages.json,/2010-04-01/Accounts/{AccountSid}/Messages/{Sid}.json
go run ./adapters/twilio/spectrim \
  -in /tmp/twilio_verify_v2.json \
  -out adapters/twilio/spec/twilio_verify_v2.json \
  -paths /v2/Services,/v2/Services/{Sid},/v2/Services/{ServiceSid}/Verifications,/v2/Services/{ServiceSid}/Verifications/{Sid},/v2/Services/{ServiceSid}/VerificationCheck
```

Trimming keeps the listed paths plus their transitive `components` closure,
as canonical JSON. Operation examples are dropped: upstream ships invalid
ones (e.g. a non-RFC3339 `date-time` example on the Messages list
operation) that strict spec validation rejects. Do not edit these files by
hand.
