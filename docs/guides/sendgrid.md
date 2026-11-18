# SendGrid guide

Point the SendGrid SDK at mocksms instead of `api.sendgrid.com`.

## Redirecting the SDK

```js
const sgMail = require('@sendgrid/mail');
sgMail.setApiKey(process.env.SENDGRID_API_KEY);
sgMail.setClient(new sgMail.Client({ baseUrl: 'http://127.0.0.1:4010/sendgrid' }));
```

Or set `SENDGRID_BASE_URL=http://127.0.0.1:4010/sendgrid` and the SDK picks it up.

## Endpoint

`POST /sendgrid/v3/mail/send` accepts the standard SendGrid mail body and returns
`202 Accepted` with an `X-Message-Id` header. The message lands in the inbox as
an email with extracted codes and links.

## Webhooks

SendGrid's event webhook is account-level. Configure the destination in the
project settings:

```bash
curl -X PUT http://127.0.0.1:4010/api/v1/projects/<id>/settings \
  -H "X-Mocksms: true" -H "Content-Type: application/json" \
  -d '{"sendgrid.webhook_url": "http://127.0.0.1:9999/events"}'
```

mocksms posts a JSON array of events (one entry per status transition) to that URL,
signed with `X-SendGrid-Signature` (HMAC-SHA256 over the body, keyed by the
observed API key; SendGrid's real webhook uses ECDSA - see `docs/fidelity.md`).

## Notes

- SendGrid is email-only; SMS magic numbers and sim rules do not apply.
- Cost estimates: see the estimate guide; SendGrid email pricing is included.