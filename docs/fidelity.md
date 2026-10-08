# mocksms — Provider Fidelity

How closely each adapter matches its real provider, and where behavior is
**unverified** (implemented from best knowledge, pending confirmation
against the provider's docs or a contract test). Per AGENTS.md, undocumented
behavior is never guessed silently: it is implemented most-likely and listed
here.

Contract tests against pinned provider OpenAPI specs live in
`adapters/<provider>/contract_test.go` (M2-05 for Twilio).

## Twilio Messages (M2-03)

Implemented: `POST /2010-04-01/Accounts/{AC}/Messages.json` (create),
`GET …/Messages.json` (list with `To`/`From`/`DateSent` filters and
`PageSize`/`Page` paging), `GET …/Messages/{SM}.json` (fetch).

- Auth: HTTP Basic `AC…:token` or `SK…:secret`; the Account SID in the path
  is the primary credential and auto-provisions a project on first use, as
  in local mode. Missing credentials return 401 code 20003.
- Message SIDs are `SM` + 32 hex chars, stored in the core `provider_ref`.
- Response fields match Twilio's: `sid`, `account_sid`, `to`, `from`,
  `body`, `status`, `num_segments` and `num_media` (as strings),
  `direction: outbound-api`, `price: null`, `price_unit: USD`,
  `error_code`/`error_message`, `uri`, `api_version`, RFC 2822 dates.
- `StatusCallback` is accepted and stored; delivery happens in M3.
- Canonical statuses map one-to-one onto Twilio's (`queued`, `sent`,
  `delivered`, `undelivered`, `failed`).

Decisions (intentional, not Twilio behavior):

- Resource `uri` values omit the local `/twilio` mount prefix, so SDKs see
  the path Twilio itself would return through the rewritten base URL.
- List and fetch show only messages sent through the Twilio adapter in the
  project (they are the only ones with an SM SID to address).
- `MediaUrl` attachments are accepted and counted into `num_media`, but the
  URLs are not stored or served (no media subresources yet).

**Unverified** (correct in M2-05 contract tests if wrong):

- Missing-field codes 21604 (`To`), 21606 (`From`), 21602 (`Body`).
- Spec-table codes 20429 (`rate_limited`), 20503 (`provider_unavailable`),
  20500 (`internal`), 60202 (`max_attempts`).
- Lenient coercion: unparsable `PageSize`/`Page` fall back to defaults
  instead of erroring; unparsable `DateSent` values are ignored.
- `DateSent` range operators (`>`, `<`, `>=`, `<=`) and exact-date matching
  semantics against Twilio's day boundaries.
- `date_sent` is set from the last update for every non-`queued` status.
- List/fetch scan at most the 5000 newest project messages; totals and
  paging are exact below that volume.

## Twilio Verify v2 (M2-04)

Implemented: `POST /v2/Services`, `GET /v2/Services/{VA}`,
`POST …/Verifications`, `GET …/Verifications/{VE}`,
`POST …/Verifications/{VE}` (`Status=canceled|approved`),
`POST …/VerificationCheck` (by `To` or `VerificationSid`).

- Services persist in project settings (`twilio_verify_services`); any `VA`
  SID is accepted and auto-provisioned with defaults (friendly name
  "mocksms", code length 6) when first used.
- Verification message text uses the service friendly name
  (`Your {name} verification code is: {code}`), via a provider-neutral
  `ServiceLabel` on the core verification request.
- Check semantics follow the core: wrong code stays `pending` (`valid:
  false`), correct code approves, exhausted attempts are 429 code 60202,
  expired/canceled/unknown verifications are 404 code 20404.
- Dates are ISO 8601; `channel` accepts `sms` and `email`.

**Unverified** (correct in M2-05 contract tests if wrong):

- Verify parameter codes: 60200 for bad `To`/`Channel`/`Code`/`Status`.
- Expired-verification check code (implemented as 20404 per the spec table).
- 429 (not 403) for exhausted check attempts.
- `To`-based checks targeting the newest pending verification.
- Service-scoped lookups 404ing across services.
- Reduced service/verification field sets (no `lookup_enabled`,
  `skip_sms_to_landlines`, `rate_limits`, `amount` breakdown, etc.).
- `date_updated` mirrors `date_created`; verification updates don't track a
  separate timestamp.
- No per-request `Ttl`/`CustomCode`/`Locale` parameters yet.
