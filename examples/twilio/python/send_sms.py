# Twilio redirect snippet for mocksms (Python).
#
# Points the official twilio SDK at a local mocksms sandbox by rewriting
# api.twilio.com and verify.twilio.com to <MOCKSMS_URL>/twilio, then sends
# an SMS and runs a full Verify check. Serves as documentation and as the
# Python leg of the M2-06 CI e2e run.
#
# Env: MOCKSMS_URL (default http://127.0.0.1:4010),
# TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN,
# TO_NUMBER (default +15005550011), FROM_NUMBER, OTP_CODE (required).
import os
import sys
from urllib.parse import urlsplit, urlunsplit

from twilio.http.http_client import TwilioHttpClient
from twilio.rest import Client

BASE_URL = os.getenv("MOCKSMS_URL", "http://127.0.0.1:4010").rstrip("/")
ACCOUNT_SID = os.getenv("TWILIO_ACCOUNT_SID", "ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
AUTH_TOKEN = os.getenv("TWILIO_AUTH_TOKEN", "testtoken123")
TO_NUMBER = os.getenv("TO_NUMBER", "+15005550011")
FROM_NUMBER = os.getenv("FROM_NUMBER", "+15555550100")
OTP_CODE = os.getenv("OTP_CODE")


def fail(message):
    print(f"FAIL: {message}", file=sys.stderr)
    sys.exit(1)


class MocksmsClient(TwilioHttpClient):
    """Redirects Twilio API traffic into the local mocksms sandbox."""

    def request(self, method, url, params=None, data=None, headers=None,
                auth=None, timeout=None, allow_redirects=False):
        parts = urlsplit(url)
        if parts.hostname and parts.hostname.endswith("twilio.com"):
            url = urlunsplit((
                urlsplit(BASE_URL).scheme,
                urlsplit(BASE_URL).netloc,
                "/twilio" + parts.path,
                parts.query,
                parts.fragment,
            ))
        return super().request(method, url, params, data, headers, auth,
                               timeout, allow_redirects)


def main():
    if not OTP_CODE:
        fail("OTP_CODE is required (start mocksms with the same --otp-code value)")

    client = Client(ACCOUNT_SID, AUTH_TOKEN, http_client=MocksmsClient())

    service = client.verify.v2.services.create(friendly_name="mocksms-e2e-python")
    if not service.sid.startswith("VA"):
        fail(f"expected VA service sid, got {service.sid}")
    print(f"service: {service.sid}")

    message = client.messages.create(
        to=TO_NUMBER,
        from_=FROM_NUMBER,
        body="Hello from the mocksms Python snippet",
        status_callback=f"{BASE_URL}/e2e-hook",
    )
    if not message.sid.startswith("SM"):
        fail(f"expected SM message sid, got {message.sid}")
    print(f"message: {message.sid} status={message.status}")

    verification = client.verify.v2.services(service.sid).verifications.create(
        to=TO_NUMBER, channel="sms",
    )
    if verification.status != "pending":
        fail(f"expected pending verification, got {verification.status}")
    print(f"verification: {verification.sid}")

    check = client.verify.v2.services(service.sid).verification_checks.create(
        to=TO_NUMBER, code=OTP_CODE,
    )
    if check.valid is not True or check.status != "approved":
        fail(f"expected approved check, got valid={check.valid} status={check.status}")
    print(f"check: valid={check.valid} status={check.status}")
    print("E2E-OK python")


if __name__ == "__main__":
    main()
