# Go-Live Estimate Methodology

How mocksms turns sandbox traffic into launch-cost numbers — and what
the numbers leave out.

## Inputs

1. **Observed mix:** every outbound SMS in the project, grouped by
   provider, destination country (from the E.164 prefix) and average
   segments per message. Native and SMTP traffic is excluded — it has
   no rate card. Email is excluded (Wave 2).
2. **Verifications:** the project's verification count, for per-OTP pricing.
3. **Volume:** the monthly message count you enter (`?volume=N`).

## Math

- Scale factor = volume ÷ observed messages. Each mix row scales to
  `round(count × factor)` monthly messages; verifications scale by the
  same factor.
- SMS cost per row = monthly messages × average segments × the
  per-segment unit price. Verification cost = monthly verifications ×
  the per-verification price. Totals round to cents.
- Zero volume or no observed traffic yields zeros — the estimate scales
  *your* mix and refuses to invent one.

## Prices

From `estimate/pricing/<provider>.yaml`, embedded in each release:

- Twilio: per-segment outbound SMS per destination country plus $0.05
  per successful verification (channel fees beyond SMS excluded).
- Termii: generic-route SMS units plus per-contact Verify price; the
  DND/OTP premium route costs 2 units.

Each table carries `as_of` and `source`, shown in the UI and API. Tables
refresh each release; between releases, confirm rate cards with the
provider before launch.

## What is excluded

- Carrier passthrough fees, number rental, sender-ID fees.
- Retries and multi-segment overhead beyond the observed average.
- Countries without table data: listed under `no_price_data` with
  message counts, never priced by guess.
- Anything marked **estimate only** is planning input, not a quote.

## API

```bash
curl "http://127.0.0.1:4010/api/v1/projects/<prj_…>/estimate?volume=10000" \
  -H "X-Mocksms: true"
```

The UI Estimate page calls `/projects/default/estimate`, so it prices
the default project. Response: `volume`, observed counts, and per
provider `monthly_messages`, `total_cost`, `by_country[]`,
`monthly_verifications`, `verify_cost`, `no_price_data[]`.
