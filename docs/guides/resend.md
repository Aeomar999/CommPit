# Resend guide

Point the Resend SDK at mocksms instead of `api.resend.com`.

## Redirecting the SDK

```js
const { Resend } = require('resend');
const resend = new Resend(process.env.RESEND_API_KEY, {
  baseUrl: 'http://127.0.0.1:4010/resend',
});
```

## Endpoint

`POST /resend/emails` accepts the standard Resend email body and returns
`200 OK` with `{ "id": "..." }`. The message lands in the inbox as an email.

## Webhooks

Resend uses Svix for webhook delivery. Configure the destination in project
settings:

```bash
curl -X PUT http://127.0.0.1:4010/api/v1/projects/<id>/settings \
  -H "X-Mocksms: true" -H "Content-Type: application/json" \
  -d '{"resend.webhook_url": "http://127.0.0.1:9999/events"}'
```

mocksms posts a JSON event (`email.sent`, `email.delivered`, `email.bounced`)
to that URL with Svix-style headers (`svix-id`, `svix-timestamp`,
`svix-signature`). The signature uses HMAC-SHA256 (sandbox convention; Resend's
real webhook uses the same Svix scheme with a `whsec_` secret).

## Notes

- Resend is email-only.