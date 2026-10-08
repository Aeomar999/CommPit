// Twilio redirect snippet for mocksms (Node.js).
//
// Points the official twilio SDK at a local mocksms sandbox by rewriting
// api.twilio.com and verify.twilio.com to <MOCKSMS_URL>/twilio, then sends
// an SMS and runs a full Verify check. Serves as documentation and as the
// Node leg of the M2-06 CI e2e run.
//
// Env: MOCKSMS_URL (default http://127.0.0.1:4010),
// TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN,
// TO_NUMBER (default +15005550010), FROM_NUMBER, OTP_CODE (required).
'use strict';

const twilio = require('twilio');
const RequestClient = require('twilio/lib/base/RequestClient');

const baseUrl = (process.env.MOCKSMS_URL || 'http://127.0.0.1:4010').replace(/\/$/, '');
const accountSid = process.env.TWILIO_ACCOUNT_SID || 'ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX';
const authToken = process.env.TWILIO_AUTH_TOKEN || 'testtoken123';
const toNumber = process.env.TO_NUMBER || '+15005550010';
const fromNumber = process.env.FROM_NUMBER || '+15555550100';
const otpCode = process.env.OTP_CODE;

function fail(message) {
  console.error(`FAIL: ${message}`);
  process.exit(1);
}

if (!otpCode) {
  fail('OTP_CODE is required (start mocksms with the same --otp-code value)');
}

// Redirects Twilio API traffic into the local mocksms sandbox.
class MocksmsClient extends RequestClient {
  constructor() {
    super();
  }

  request(opts) {
    const url = new URL(opts.uri);
    if (url.hostname.endsWith('twilio.com')) {
      opts = { ...opts, uri: `${baseUrl}/twilio${url.pathname}${url.search}` };
    }
    return super.request(opts);
  }
}

async function main() {
  const client = twilio(accountSid, authToken, { httpClient: new MocksmsClient() });

  const service = await client.verify.v2.services.create({ friendlyName: 'mocksms-e2e-node' });
  if (!service.sid.startsWith('VA')) {
    fail(`expected VA service sid, got ${service.sid}`);
  }
  console.log(`service: ${service.sid}`);

  const message = await client.messages.create({
    to: toNumber,
    from: fromNumber,
    body: 'Hello from the mocksms Node snippet',
    statusCallback: `${baseUrl}/e2e-hook`,
  });
  if (!message.sid.startsWith('SM')) {
    fail(`expected SM message sid, got ${message.sid}`);
  }
  console.log(`message: ${message.sid} status=${message.status}`);

  const verification = await client.verify.v2.services(service.sid).verifications.create({
    to: toNumber,
    channel: 'sms',
  });
  if (verification.status !== 'pending') {
    fail(`expected pending verification, got ${verification.status}`);
  }
  console.log(`verification: ${verification.sid}`);

  const check = await client.verify.v2.services(service.sid).verificationChecks.create({
    to: toNumber,
    code: otpCode,
  });
  if (check.valid !== true || check.status !== 'approved') {
    fail(`expected approved check, got valid=${check.valid} status=${check.status}`);
  }
  console.log(`check: valid=${check.valid} status=${check.status}`);
  console.log('E2E-OK node');
}

main().catch((err) => fail(err && err.message ? err.message : String(err)));
