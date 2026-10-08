package termii_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/Aeomar999/CommPit/core"
)

var pinIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func sendTermiiOTP(t *testing.T, handler http.Handler, payload map[string]any) map[string]any {
	t.Helper()
	req := termiiRequest(t, "/api/sms/otp/send", payload, true)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	return decodeTermii(t, rec)
}

func TestTermii_OtpSend(t *testing.T) {
	handler, store := setupTestTermii(t)

	t.Run("send returns pin id and delivers templated text", func(t *testing.T) {
		resp := sendTermiiOTP(t, handler, map[string]any{
			"to":               testToNG,
			"from":             testSender,
			"message_text":     "Your pin is < 1234 >, valid for 10 minutes",
			"pin_placeholder":  "< 1234 >",
			"pin_length":       6,
			"pin_attempts":     3,
			"pin_time_to_live": 10,
		})
		pinID, _ := resp["pinId"].(string)
		if !pinIDPattern.MatchString(pinID) {
			t.Errorf("expected UUID pinId, got %v", pinID)
		}
		if resp["to"] != testToNG {
			t.Errorf("expected to %s, got %v", testToNG, resp["to"])
		}

		projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "termii", testAPIKey)
		if err != nil {
			t.Fatalf("resolve project: %v", err)
		}
		v, err := store.GetVerificationByProviderRef(context.Background(), projectID, pinID)
		if err != nil {
			t.Fatalf("GetVerificationByProviderRef: %v", err)
		}
		if len(v.Code) != 6 {
			t.Errorf("expected 6-digit code, got %q", v.Code)
		}
		if v.MaxAttempts != 3 {
			t.Errorf("expected max attempts 3, got %d", v.MaxAttempts)
		}
		msg, err := store.GetMessage(context.Background(), projectID, v.MessageID)
		if err != nil {
			t.Fatalf("GetMessage: %v", err)
		}
		if !strings.Contains(msg.BodyText, v.Code) || strings.Contains(msg.BodyText, "< 1234 >") {
			t.Errorf("expected substituted body with code, got %q", msg.BodyText)
		}
	})

	t.Run("pin length respected", func(t *testing.T) {
		resp := sendTermiiOTP(t, handler, map[string]any{
			"to": testToNG2, "from": testSender, "pin_length": 4,
		})
		projectID, _ := core.NewProjectResolver(store).Resolve(context.Background(), "termii", testAPIKey)
		v, err := store.GetVerificationByProviderRef(context.Background(), projectID, resp["pinId"].(string))
		if err != nil {
			t.Fatalf("GetVerificationByProviderRef: %v", err)
		}
		if len(v.Code) != 4 {
			t.Errorf("expected 4-digit code, got %q", v.Code)
		}
	})

	t.Run("missing fields rejected", func(t *testing.T) {
		for name, payload := range map[string]map[string]any{
			"missing to":   {"from": testSender},
			"missing from": {"to": testToNG},
		} {
			req := termiiRequest(t, "/api/sms/otp/send", payload, true)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s: expected status 400, got %d: %s", name, rec.Code, rec.Body.String())
			}
		}
	})
}

func TestTermii_OtpVerify(t *testing.T) {
	handler, store := setupTestTermii(t)
	resp := sendTermiiOTP(t, handler, map[string]any{
		"to": testToNG, "from": testSender,
	})
	pinID := resp["pinId"].(string)

	projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "termii", testAPIKey)
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	v, err := store.GetVerificationByProviderRef(context.Background(), projectID, pinID)
	if err != nil {
		t.Fatalf("GetVerificationByProviderRef: %v", err)
	}
	wrong := "000000"
	if wrong == v.Code {
		wrong = "111111"
	}

	verify := func(pinID, pin string) (int, map[string]any) {
		req := termiiRequest(t, "/api/sms/otp/verify", map[string]any{
			"pin_id": pinID, "pin": pin,
		}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code, decodeTermii(t, rec)
	}

	t.Run("wrong pin not verified", func(t *testing.T) {
		status, resp := verify(pinID, wrong)
		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %v", status, resp)
		}
		if resp["verified"] != false {
			t.Errorf("expected verified false, got %v", resp)
		}
	})

	t.Run("correct pin verified", func(t *testing.T) {
		status, resp := verify(pinID, v.Code)
		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %v", status, resp)
		}
		if resp["verified"] != true {
			t.Errorf("expected verified true, got %v", resp)
		}
		if resp["pinId"] != pinID {
			t.Errorf("expected pinId echo, got %v", resp)
		}
	})

	t.Run("unknown pin id 404", func(t *testing.T) {
		req := termiiRequest(t, "/api/sms/otp/verify", map[string]any{
			"pin_id": "00000000-0000-0000-0000-000000000000", "pin": "123456",
		}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("exhausted attempts 429", func(t *testing.T) {
		resp2 := sendTermiiOTP(t, handler, map[string]any{
			"to": testToNG2, "from": testSender, "pin_attempts": 1,
		})
		status, resp := verify(resp2["pinId"].(string), wrong)
		if status != http.StatusTooManyRequests {
			t.Fatalf("expected status 429, got %d: %v", status, resp)
		}
	})
}

func TestTermii_OtpGenerate(t *testing.T) {
	handler, store := setupTestTermii(t)

	t.Run("returns pin without sending", func(t *testing.T) {
		req := termiiRequest(t, "/api/sms/otp/generate", map[string]any{
			"pin_type": "NUMERIC", "pin_length": 8,
		}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		resp := decodeTermii(t, rec)
		pin, _ := resp["pin"].(string)
		if len(pin) != 8 {
			t.Errorf("expected 8-digit pin, got %q", pin)
		}

		projectID, _ := core.NewProjectResolver(store).Resolve(context.Background(), "termii", testAPIKey)
		msgs, _, err := store.ListMessages(context.Background(), projectID, core.MessageFilter{Limit: 10})
		if err != nil {
			t.Fatalf("ListMessages: %v", err)
		}
		if len(msgs) != 0 {
			t.Errorf("expected no messages sent, got %d", len(msgs))
		}
	})

	t.Run("unsupported pin type rejected", func(t *testing.T) {
		req := termiiRequest(t, "/api/sms/otp/generate", map[string]any{
			"pin_type": "ALPHANUMERIC",
		}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestTermii_EmailOtpSend(t *testing.T) {
	handler, store := setupTestTermii(t)

	t.Run("stores caller code on email verification", func(t *testing.T) {
		req := termiiRequest(t, "/api/email/otp/send", map[string]any{
			"email_address": "user@example.com",
			"code":          "731946",
		}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "termii", testAPIKey)
		if err != nil {
			t.Fatalf("resolve project: %v", err)
		}
		verifications, _, err := store.ListVerifications(context.Background(), projectID, 10, "")
		if err != nil {
			t.Fatalf("ListVerifications: %v", err)
		}
		if len(verifications) != 1 {
			t.Fatalf("expected 1 verification, got %d", len(verifications))
		}
		if verifications[0].Code != "731946" {
			t.Errorf("expected caller code stored, got %q", verifications[0].Code)
		}
		if verifications[0].Channel != core.ChannelEmail {
			t.Errorf("expected email channel, got %q", verifications[0].Channel)
		}
	})

	t.Run("bad email rejected", func(t *testing.T) {
		req := termiiRequest(t, "/api/email/otp/send", map[string]any{
			"email_address": "not-an-email",
			"code":          "731946",
		}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing code rejected", func(t *testing.T) {
		req := termiiRequest(t, "/api/email/otp/send", map[string]any{
			"email_address": "user@example.com",
		}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestTermii_OtpEndpointsRequireKey(t *testing.T) {
	handler, _ := setupTestTermii(t)
	for _, target := range []string{"/api/sms/otp/send", "/api/sms/otp/verify", "/api/sms/otp/generate", "/api/email/otp/send"} {
		req := termiiRequest(t, target, map[string]any{}, false)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: expected status 401, got %d: %s", target, rec.Code, rec.Body.String())
		}
	}
}
