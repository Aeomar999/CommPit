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
