// Twilio redirect snippet for mocksms (Go).
//
// Points the official twilio-go SDK at a local mocksms sandbox by rewriting
// api.twilio.com and verify.twilio.com to <MOCKSMS_URL>/twilio, then sends
// an SMS and runs a full Verify check. Serves as documentation and as the
// Go leg of the M2-06 CI e2e run.
//
// Env: MOCKSMS_URL (default http://127.0.0.1:4010),
// TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN,
// TO_NUMBER (default +15005550013), FROM_NUMBER, OTP_CODE (required).
//
// Run: go mod download && OTP_CODE=123456 go run .
package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/twilio/twilio-go"
	"github.com/twilio/twilio-go/client"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	verify "github.com/twilio/twilio-go/rest/verify/v2"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fail(err any) {
	fmt.Fprintln(os.Stderr, "FAIL:", err)
	os.Exit(1)
}

// mocksmsClient redirects Twilio API traffic into the local sandbox.
type mocksmsClient struct {
	client.Client
	baseURL string
}

// SendRequest rewrites Twilio hosts to the local sandbox before delegating.
func (c *mocksmsClient) SendRequest(method string, rawURL string, data url.Values, headers map[string]interface{}, body ...byte) (*http.Response, error) {
	if u, err := url.Parse(rawURL); err == nil && strings.HasSuffix(u.Hostname(), "twilio.com") {
		base, err := url.Parse(c.baseURL)
		if err != nil {
			return nil, err
		}
		u.Scheme = base.Scheme
		u.Host = base.Host
		u.Path = "/twilio" + u.Path
		rawURL = u.String()
	}
	return c.Client.SendRequest(method, rawURL, data, headers, body...)
}

func main() {
	baseURL := strings.TrimRight(getenv("MOCKSMS_URL", "http://127.0.0.1:4010"), "/")
	accountSid := getenv("TWILIO_ACCOUNT_SID", "ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
	authToken := getenv("TWILIO_AUTH_TOKEN", "testtoken123")
	toNumber := getenv("TO_NUMBER", "+15005550013")
	fromNumber := getenv("FROM_NUMBER", "+15555550100")
	otpCode := os.Getenv("OTP_CODE")
	if otpCode == "" {
		fail("OTP_CODE is required (start mocksms with the same --otp-code value)")
	}

	custom := &mocksmsClient{
		Client: client.Client{
			Credentials: client.NewCredentials(accountSid, authToken),
		},
		baseURL: baseURL,
	}
	custom.SetAccountSid(accountSid)
	restClient := twilio.NewRestClientWithParams(twilio.ClientParams{Client: custom})

	svcParams := &verify.CreateServiceParams{}
	svcParams.SetFriendlyName("mocksms-e2e-go")
	service, err := restClient.VerifyV2.CreateService(svcParams)
	if err != nil {
		fail(err)
	}
	if service.Sid == nil || !strings.HasPrefix(*service.Sid, "VA") {
		fail(fmt.Sprintf("expected VA service sid, got %v", service.Sid))
	}
	fmt.Println("service:", *service.Sid)

	msgParams := &twilioApi.CreateMessageParams{}
	msgParams.SetTo(toNumber)
	msgParams.SetFrom(fromNumber)
	msgParams.SetBody("Hello from the mocksms Go snippet")
	msgParams.SetStatusCallback(baseURL + "/e2e-hook")
	message, err := restClient.Api.CreateMessage(msgParams)
	if err != nil {
		fail(err)
	}
	if message.Sid == nil || !strings.HasPrefix(*message.Sid, "SM") {
		fail(fmt.Sprintf("expected SM message sid, got %v", message.Sid))
	}
	fmt.Println("message:", *message.Sid, "status="+valueOr(message.Status, "?"))

	verParams := &verify.CreateVerificationParams{}
	verParams.SetTo(toNumber)
	verParams.SetChannel("sms")
	verification, err := restClient.VerifyV2.CreateVerification(*service.Sid, verParams)
	if err != nil {
		fail(err)
	}
	if verification.Status == nil || *verification.Status != "pending" {
		fail(fmt.Sprintf("expected pending verification, got %v", verification.Status))
	}
	fmt.Println("verification:", valueOr(verification.Sid, "?"))

	checkParams := &verify.CreateVerificationCheckParams{}
	checkParams.SetTo(toNumber)
	checkParams.SetCode(otpCode)
	check, err := restClient.VerifyV2.CreateVerificationCheck(*service.Sid, checkParams)
	if err != nil {
		fail(err)
	}
	if check.Valid == nil || !*check.Valid || check.Status == nil || *check.Status != "approved" {
		fail(fmt.Sprintf("expected approved check, got valid=%v status=%v", check.Valid, check.Status))
	}
	fmt.Println("check: valid=true status=approved")
	fmt.Println("E2E-OK go")
}

func valueOr(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}
