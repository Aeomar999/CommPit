// Twilio redirect snippet for mocksms (C#).
//
// Points the official Twilio SDK at a local mocksms sandbox by rewriting
// api.twilio.com and verify.twilio.com to <MOCKSMS_URL>/twilio, then sends
// an SMS and runs a full Verify check. Serves as documentation and as the
// C# leg of the M2-06 CI e2e run.
//
// Env: MOCKSMS_URL (default http://127.0.0.1:4010),
// TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN,
// TO_NUMBER (default +15005550014), FROM_NUMBER, OTP_CODE (required).
//
// Run: dotnet run (with OTP_CODE set, e.g. OTP_CODE=123456 dotnet run)
using System;
using System.Threading.Tasks;
using Twilio.Clients;
using Twilio.Http;
using Twilio.Rest.Api.V2010.Account;
using Twilio.Rest.Verify.V2;
using Twilio.Rest.Verify.V2.Service;
using Twilio.Types;

namespace MocksmsTwilioExample
{
    // Redirects Twilio API traffic into the local mocksms sandbox.
    public class MocksmsHttpClient : Twilio.Http.HttpClient
    {
        private readonly SystemNetHttpClient _inner = new SystemNetHttpClient();
        private readonly string _baseUrl;

        public MocksmsHttpClient(string baseUrl)
        {
            _baseUrl = baseUrl.TrimEnd('/');
        }

        private Request Rewrite(Request request)
        {
            var uri = request.Uri;
            var url = uri.AbsoluteUri;
            if (uri.Host.EndsWith("twilio.com"))
            {
                url = _baseUrl + "/twilio" + uri.AbsolutePath;
            }
            var rewritten = new Request(request.Method, url);
            foreach (var param in request.QueryParams)
            {
                rewritten.AddQueryParam(param.Key, param.Value);
            }
            foreach (var param in request.PostParams)
            {
                rewritten.AddPostParam(param.Key, param.Value);
            }
            foreach (var param in request.HeaderParams)
            {
                rewritten.AddHeaderParam(param.Key, param.Value);
            }
            rewritten.Username = request.Username;
            rewritten.Password = request.Password;
            rewritten.ContentType = request.ContentType;
            rewritten.Body = request.Body;
            return rewritten;
        }

        public override Response MakeRequest(Request request)
        {
            return _inner.MakeRequest(Rewrite(request));
        }

        public override async Task<Response> MakeRequestAsync(Request request)
        {
            return await _inner.MakeRequestAsync(Rewrite(request));
        }
    }

    public static class Program
    {
        private static string Env(string key, string fallback)
        {
            var value = Environment.GetEnvironmentVariable(key);
            return string.IsNullOrEmpty(value) ? fallback : value;
        }

        private static void Fail(string message)
        {
            Console.Error.WriteLine("FAIL: " + message);
            Environment.Exit(1);
        }

        public static void Main(string[] args)
        {
            var baseUrl = Env("MOCKSMS_URL", "http://127.0.0.1:4010").TrimEnd('/');
            var accountSid = Env("TWILIO_ACCOUNT_SID", "ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX");
            var authToken = Env("TWILIO_AUTH_TOKEN", "testtoken123");
            var toNumber = Env("TO_NUMBER", "+15005550014");
            var fromNumber = Env("FROM_NUMBER", "+15555550100");
            var otpCode = Environment.GetEnvironmentVariable("OTP_CODE");
            if (string.IsNullOrEmpty(otpCode))
            {
                Fail("OTP_CODE is required (start mocksms with the same --otp-code value)");
            }

            var restClient = new TwilioRestClient(
                accountSid,
                authToken,
                accountSid,
                httpClient: new MocksmsHttpClient(baseUrl));

            var service = ServiceResource.Create(
                friendlyName: "mocksms-e2e-csharp",
                client: restClient);
            if (service.Sid == null || !service.Sid.StartsWith("VA"))
            {
                Fail("expected VA service sid, got " + service.Sid);
            }
            Console.WriteLine("service: " + service.Sid);

            var message = MessageResource.Create(
                to: new PhoneNumber(toNumber),
                from: new PhoneNumber(fromNumber),
                body: "Hello from the mocksms C# snippet",
                statusCallback: new Uri(baseUrl + "/e2e-hook"),
                client: restClient);
            if (message.Sid == null || !message.Sid.StartsWith("SM"))
            {
                Fail("expected SM message sid, got " + message.Sid);
            }
            Console.WriteLine("message: " + message.Sid + " status=" + message.Status);

            var verification = VerificationResource.Create(
                pathServiceSid: service.Sid,
                to: toNumber,
                channel: "sms",
                client: restClient);
            if (verification.Status != "pending")
            {
                Fail("expected pending verification, got " + verification.Status);
            }
            Console.WriteLine("verification: " + verification.Sid);

            var check = VerificationCheckResource.Create(
                pathServiceSid: service.Sid,
                code: otpCode,
                to: toNumber,
                client: restClient);
            if (check.Valid != true || check.Status != "approved")
            {
                Fail("expected approved check, got valid=" + check.Valid + " status=" + check.Status);
            }
            Console.WriteLine("check: valid=True status=" + check.Status);
            Console.WriteLine("E2E-OK csharp");
        }
    }
}
