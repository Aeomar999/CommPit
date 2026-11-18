# Vonage guide

Point the Vonage SDK at mocksms.

## Redirecting the SDK

```js
const { Vonage } = require('@vonage/server-sdk');
const vonage = new Vonage({ apiKey: 'key', apiSecret: 'secret' }, { apiHost: 'http://127.0.0.1:4010/vonage' });
```

## Endpoints

- `POST /vonage/sms/json` - send SMS. Body: `{api_key, api_secret, from, to, text}`. Returns `{message-count, messages: [...]}`.
- `POST /vonage/v2/verify` - start verification. Body: `{brand, workflow: [{channel, to}]}`. Returns `{request_id, status}`.
- `POST /vonage/v2/verify/{request_id}` - check code. Body: `{code}`. Returns `{request_id, status}`.

## Notes

- Vonage is SMS-only in Wave 2; voice/WhatsApp are backlog.