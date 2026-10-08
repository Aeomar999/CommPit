<?php

declare(strict_types=1);

// Twilio redirect snippet for mocksms (PHP).
//
// Points the official twilio/sdk at a local mocksms sandbox by rewriting
// api.twilio.com and verify.twilio.com to <MOCKSMS_URL>/twilio, then sends
// an SMS and runs a full Verify check. Serves as documentation and as the
// PHP leg of the M2-06 CI e2e run.
//
// Env: MOCKSMS_URL (default http://127.0.0.1:4010),
// TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN,
// TO_NUMBER (default +15005550012), FROM_NUMBER, OTP_CODE (required).
//
// Run: composer install && OTP_CODE=123456 php send_sms.php

require __DIR__ . '/vendor/autoload.php';

use Twilio\AuthStrategy\AuthStrategy;
use Twilio\Http\Client as HttpClientInterface;
use Twilio\Http\CurlClient;
use Twilio\Http\Response;
use Twilio\Rest\Client;

$baseUrl = rtrim(getenv('MOCKSMS_URL') ?: 'http://127.0.0.1:4010', '/');
$accountSid = getenv('TWILIO_ACCOUNT_SID') ?: 'ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX';
$authToken = getenv('TWILIO_AUTH_TOKEN') ?: 'testtoken123';
$toNumber = getenv('TO_NUMBER') ?: '+15005550012';
$fromNumber = getenv('FROM_NUMBER') ?: '+15555550100';
$otpCode = getenv('OTP_CODE');

function fail(string $message): void {
    fwrite(STDERR, "FAIL: $message\n");
    exit(1);
}

if (!$otpCode) {
    fail('OTP_CODE is required (start mocksms with the same --otp-code value)');
}

// Redirects Twilio API traffic into the local mocksms sandbox.
class MocksmsClient implements HttpClientInterface {
    private CurlClient $inner;
    private string $baseUrl;

    public function __construct(string $baseUrl) {
        $this->inner = new CurlClient();
        $this->baseUrl = rtrim($baseUrl, '/');
    }

    public function request(string $method, string $url,
                            array $params = [], array $data = [], array $headers = [],
                            ?string $user = null, ?string $password = null,
                            ?int $timeout = null, ?AuthStrategy $authStrategy = null): Response {
        $parts = parse_url($url);
        if (isset($parts['host']) && str_ends_with($parts['host'], 'twilio.com')) {
            $url = $this->baseUrl . '/twilio' . ($parts['path'] ?? '/')
                . (isset($parts['query']) ? '?' . $parts['query'] : '');
        }
        return $this->inner->request($method, $url, $params, $data, $headers, $user, $password, $timeout, $authStrategy);
    }
}

$client = new Client($accountSid, $authToken, $accountSid, null, new MocksmsClient($baseUrl));

$service = $client->verify->v2->services->create('mocksms-e2e-php');
if (!str_starts_with($service->sid, 'VA')) {
    fail('expected VA service sid, got ' . $service->sid);
}
echo "service: {$service->sid}\n";

$message = $client->messages->create($toNumber, [
    'from' => $fromNumber,
    'body' => 'Hello from the mocksms PHP snippet',
    'statusCallback' => $baseUrl . '/e2e-hook',
]);
if (!str_starts_with($message->sid, 'SM')) {
    fail('expected SM message sid, got ' . $message->sid);
}
echo "message: {$message->sid} status={$message->status}\n";

$verification = $client->verify->v2->services($service->sid)->verifications->create($toNumber, 'sms');
if ($verification->status !== 'pending') {
    fail('expected pending verification, got ' . $verification->status);
}
echo "verification: {$verification->sid}\n";

$check = $client->verify->v2->services($service->sid)->verificationChecks->create([
    'to' => $toNumber,
    'code' => $otpCode,
]);
if ($check->valid !== true || $check->status !== 'approved') {
    fail('expected approved check, got valid=' . var_export($check->valid, true) . ' status=' . $check->status);
}
echo "check: valid=1 status={$check->status}\n";
echo "E2E-OK php\n";
