package twilio_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Aeomar999/CommPit/core"
)

var (
	vaPattern = regexp.MustCompile(`^VA[0-9a-f]{32}$`)
	vePattern = regexp.MustCompile(`^VE[0-9a-f]{32}$`)
)

func verifyPath(sid string, parts ...string) string {
	p := "/v2/Services/" + sid
	for _, part := range parts {
		p += "/" + part
	}
	return p
}

func createVerifyService(t *testing.T, handler http.Handler, friendlyName string, codeLength int) map[string]any {
	t.Helper()
	form := url.Values{}
	if friendlyName != "" {
		form.Set("FriendlyName", friendlyName)
	}
	if codeLength != 0 {
		form.Set("CodeLength", strconv.Itoa(codeLength))
	}
	req := twilioRequest(t, http.MethodPost, "/v2/Services", form, true)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	return decodeJSON(t, rec)
}

func startVerifyVerification(t *testing.T, handler http.Handler, serviceSid, to, channel string) map[string]any {
	t.Helper()
	form := url.Values{"To": {to}}
	if channel != "" {
		form.Set("Channel", channel)
	}
	req := twilioRequest(t, http.MethodPost, verifyPath(serviceSid, "Verifications"), form, true)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	return decodeJSON(t, rec)
}

func verificationCode(t *testing.T, store core.Store, serviceSid, verifySid string) string {
	t.Helper()
	projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "twilio", testAccountSID)
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	v, err := store.GetVerificationByProviderRef(context.Background(), projectID, verifySid)
	if err != nil {
		t.Fatalf("GetVerificationByProviderRef: %v", err)
	}
	if v.ServiceRef == nil || *v.ServiceRef != serviceSid {
		t.Fatalf("expected service ref %s, got %v", serviceSid, v.ServiceRef)
	}
	return v.Code
}

func TestTwilio_CreateService(t *testing.T) {
	handler, _ := setupTestTwilio(t)

	t.Run("defaults", func(t *testing.T) {
		resp := createVerifyService(t, handler, "", 0)
		sid, _ := resp["sid"].(string)
		if !vaPattern.MatchString(sid) {
			t.Errorf("expected sid VA+32 hex, got %q", sid)
		}
		if resp["friendly_name"] != "mocksms" {
			t.Errorf("expected friendly_name mocksms, got %v", resp["friendly_name"])
		}
		if resp["code_length"] != float64(6) {
			t.Errorf("expected code_length 6, got %v", resp["code_length"])
		}
	})

	t.Run("custom friendly name and code length", func(t *testing.T) {
		resp := createVerifyService(t, handler, "my-app", 4)
		if resp["friendly_name"] != "my-app" {
			t.Errorf("expected friendly_name my-app, got %v", resp["friendly_name"])
		}
		if resp["code_length"] != float64(4) {
			t.Errorf("expected code_length 4, got %v", resp["code_length"])
		}
	})

	t.Run("invalid code length", func(t *testing.T) {
		form := url.Values{"CodeLength": {"3"}}
		req := twilioRequest(t, http.MethodPost, "/v2/Services", form, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
		if resp := decodeJSON(t, rec); resp["code"] != float64(60200) {
			t.Errorf("expected Twilio code 60200, got %v", resp["code"])
		}
	})
}

func TestTwilio_FetchService(t *testing.T) {
	handler, _ := setupTestTwilio(t)
	created := createVerifyService(t, handler, "fetch-me", 6)
	sid := created["sid"].(string)

	t.Run("fetch returns the service", func(t *testing.T) {
		req := twilioRequest(t, http.MethodGet, verifyPath(sid), nil, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if resp := decodeJSON(t, rec); resp["friendly_name"] != "fetch-me" {
			t.Errorf("expected friendly_name fetch-me, got %v", resp["friendly_name"])
		}
	})

	t.Run("unknown service returns 20404", func(t *testing.T) {
		req := twilioRequest(t, http.MethodGet, verifyPath("VA00000000000000000000000000000000"), nil, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
		}
		if resp := decodeJSON(t, rec); resp["code"] != float64(20404) {
			t.Errorf("expected Twilio code 20404, got %v", resp["code"])
		}
	})
}

func TestTwilio_CreateVerification(t *testing.T) {
	handler, store := setupTestTwilio(t)
	svc := createVerifyService(t, handler, "my-app", 6)
	serviceSid := svc["sid"].(string)

	t.Run("sms verification carries friendly name text", func(t *testing.T) {
		resp := startVerifyVerification(t, handler, serviceSid, testTo, "sms")
		sid, _ := resp["sid"].(string)
		if !vePattern.MatchString(sid) {
			t.Errorf("expected sid VE+32 hex, got %q", sid)
		}
		if resp["status"] != "pending" {
			t.Errorf("expected status pending, got %v", resp["status"])
		}
		if resp["valid"] != false {
			t.Errorf("expected valid false, got %v", resp["valid"])
		}
		if resp["service_sid"] != serviceSid {
			t.Errorf("expected service_sid %s, got %v", serviceSid, resp["service_sid"])
		}

		code := verificationCode(t, store, serviceSid, sid)
		if len(code) != 6 {
			t.Fatalf("expected 6-digit code, got %q", code)
		}
		projectID, _ := core.NewProjectResolver(store).Resolve(context.Background(), "twilio", testAccountSID)
		v, _ := store.GetVerificationByProviderRef(context.Background(), projectID, sid)
		msg, err := store.GetMessage(context.Background(), projectID, v.MessageID)
		if err != nil {
			t.Fatalf("GetMessage: %v", err)
		}
		if !strings.Contains(msg.BodyText, "my-app") || !strings.Contains(msg.BodyText, code) {
			t.Errorf("expected body with friendly name and code, got %q", msg.BodyText)
		}
	})

	t.Run("unknown service sid is auto-provisioned", func(t *testing.T) {
		resp := startVerifyVerification(t, handler, "VA99999999999999999999999999999999", testTo, "sms")
		if resp["service_sid"] != "VA99999999999999999999999999999999" {
			t.Errorf("expected auto-provisioned service sid, got %v", resp["service_sid"])
		}
	})

	t.Run("email channel", func(t *testing.T) {
		resp := startVerifyVerification(t, handler, serviceSid, "user@example.com", "email")
		if resp["channel"] != "email" {
			t.Errorf("expected channel email, got %v", resp["channel"])
		}
	})

	t.Run("invalid channel", func(t *testing.T) {
		form := url.Values{"To": {testTo}, "Channel": {"whatsapp"}}
		req := twilioRequest(t, http.MethodPost, verifyPath(serviceSid, "Verifications"), form, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
		if resp := decodeJSON(t, rec); resp["code"] != float64(60200) {
			t.Errorf("expected Twilio code 60200, got %v", resp["code"])
		}
	})

	t.Run("missing To", func(t *testing.T) {
		req := twilioRequest(t, http.MethodPost, verifyPath(serviceSid, "Verifications"), url.Values{}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestTwilio_VerificationCheck(t *testing.T) {
	check := func(t *testing.T, handler http.Handler, serviceSid string, form url.Values) (int, map[string]any) {
		t.Helper()
		req := twilioRequest(t, http.MethodPost, verifyPath(serviceSid, "VerificationCheck"), form, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code, decodeJSON(t, rec)
	}

	t.Run("wrong code stays pending", func(t *testing.T) {
		handler, store := setupTestTwilio(t)
		svc := createVerifyService(t, handler, "", 0)
		serviceSid := svc["sid"].(string)
		ver := startVerifyVerification(t, handler, serviceSid, testTo, "sms")
		code := verificationCode(t, store, serviceSid, ver["sid"].(string))
		wrong := "000000"
		if wrong == code {
			wrong = "111111"
		}
		status, resp := check(t, handler, serviceSid, url.Values{"To": {testTo}, "Code": {wrong}})
		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %v", status, resp)
		}
		if resp["valid"] != false || resp["status"] != "pending" {
			t.Errorf("expected valid=false pending, got %v", resp)
		}
	})

	t.Run("correct code approves by To and by sid", func(t *testing.T) {
		handler, store := setupTestTwilio(t)
		svc := createVerifyService(t, handler, "", 0)
		serviceSid := svc["sid"].(string)
		ver := startVerifyVerification(t, handler, serviceSid, testTo, "sms")
		code := verificationCode(t, store, serviceSid, ver["sid"].(string))

		status, resp := check(t, handler, serviceSid, url.Values{"To": {testTo}, "Code": {code}})
		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %v", status, resp)
		}
		if resp["valid"] != true || resp["status"] != "approved" {
			t.Errorf("expected valid=true approved, got %v", resp)
		}

		ver2 := startVerifyVerification(t, handler, serviceSid, testTo, "sms")
		code2 := verificationCode(t, store, serviceSid, ver2["sid"].(string))
		status, resp = check(t, handler, serviceSid, url.Values{"VerificationSid": {ver2["sid"].(string)}, "Code": {code2}})
		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %v", status, resp)
		}
		if resp["valid"] != true || resp["status"] != "approved" {
			t.Errorf("expected valid=true approved by sid, got %v", resp)
		}
	})

	t.Run("exhausted attempts return 60202", func(t *testing.T) {
		handler, store := setupTestTwilio(t)
		svc := createVerifyService(t, handler, "", 0)
		serviceSid := svc["sid"].(string)
		ver := startVerifyVerification(t, handler, serviceSid, testTo, "sms")
		code := verificationCode(t, store, serviceSid, ver["sid"].(string))
		wrong := "000000"
		if wrong == code {
			wrong = "111111"
		}
		for i := 0; i < 4; i++ {
			status, _ := check(t, handler, serviceSid, url.Values{"To": {testTo}, "Code": {wrong}})
			if status != http.StatusOK {
				t.Fatalf("check %d: expected status 200, got %d", i, status)
			}
		}
		status, resp := check(t, handler, serviceSid, url.Values{"To": {testTo}, "Code": {wrong}})
		if status != http.StatusTooManyRequests {
			t.Fatalf("expected status 429, got %d: %v", status, resp)
		}
		if resp["code"] != float64(60202) {
			t.Errorf("expected Twilio code 60202, got %v", resp["code"])
		}
	})

	t.Run("unknown sid returns 20404", func(t *testing.T) {
		handler, _ := setupTestTwilio(t)
		svc := createVerifyService(t, handler, "", 0)
		serviceSid := svc["sid"].(string)
		status, resp := check(t, handler, serviceSid, url.Values{"VerificationSid": {"VE00000000000000000000000000000000"}, "Code": {"123456"}})
		if status != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %v", status, resp)
		}
		if resp["code"] != float64(20404) {
			t.Errorf("expected Twilio code 20404, got %v", resp["code"])
		}
	})

	t.Run("missing code returns 60200", func(t *testing.T) {
		handler, _ := setupTestTwilio(t)
		svc := createVerifyService(t, handler, "", 0)
		serviceSid := svc["sid"].(string)
		status, resp := check(t, handler, serviceSid, url.Values{"To": {testTo}})
		if status != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %v", status, resp)
		}
		if resp["code"] != float64(60200) {
			t.Errorf("expected Twilio code 60200, got %v", resp["code"])
		}
	})
}

func TestTwilio_UpdateVerification(t *testing.T) {
	handler, _ := setupTestTwilio(t)
	svc := createVerifyService(t, handler, "", 0)
	serviceSid := svc["sid"].(string)

	update := func(t *testing.T, verifySid, status string) (int, map[string]any) {
		t.Helper()
		form := url.Values{}
		if status != "" {
			form.Set("Status", status)
		}
		req := twilioRequest(t, http.MethodPost, verifyPath(serviceSid, "Verifications", verifySid), form, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code, decodeJSON(t, rec)
	}

	t.Run("cancel moves to canceled and blocks checks", func(t *testing.T) {
		ver := startVerifyVerification(t, handler, serviceSid, testTo, "sms")
		sid := ver["sid"].(string)
		status, resp := update(t, sid, "canceled")
		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %v", status, resp)
		}
		if resp["status"] != "canceled" {
			t.Errorf("expected status canceled, got %v", resp["status"])
		}

		req := twilioRequest(t, http.MethodPost, verifyPath(serviceSid, "VerificationCheck"), url.Values{"VerificationSid": {sid}, "Code": {"123456"}}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404 after cancel, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("approve moves to approved", func(t *testing.T) {
		ver := startVerifyVerification(t, handler, serviceSid, "+15005550007", "sms")
		sid := ver["sid"].(string)
		status, resp := update(t, sid, "approved")
		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %v", status, resp)
		}
		if resp["status"] != "approved" {
			t.Errorf("expected status approved, got %v", resp["status"])
		}
	})

	t.Run("invalid status returns 60200", func(t *testing.T) {
		ver := startVerifyVerification(t, handler, serviceSid, "+15005550008", "sms")
		status, resp := update(t, ver["sid"].(string), "bogus")
		if status != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %v", status, resp)
		}
		if resp["code"] != float64(60200) {
			t.Errorf("expected Twilio code 60200, got %v", resp["code"])
		}
	})

	t.Run("fetch returns the verification", func(t *testing.T) {
		ver := startVerifyVerification(t, handler, serviceSid, "+15005550010", "sms")
		sid := ver["sid"].(string)
		req := twilioRequest(t, http.MethodGet, verifyPath(serviceSid, "Verifications", sid), nil, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if resp := decodeJSON(t, rec); resp["sid"] != sid {
			t.Errorf("expected sid %s, got %v", sid, resp["sid"])
		}
	})
}
