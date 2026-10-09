package twilio_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/adapters/twilio"
	"github.com/Aeomar999/CommPit/core"
)

func TestTwilio_SignRequest(t *testing.T) {
	// Ground-truth vector from Twilio's docs, confirmed with the official
	// Python SDK's RequestValidator.
	params := map[string]string{
		"CallSid": "CA1234567890ABCDE",
		"Caller":  "+12349013030",
		"Digits":  "1234",
		"From":    "+12349013030",
		"To":      "+18005551212",
	}
	got := twilio.SignRequest("https://mycompany.com/myapp.php?foo=1&bar=2", params, "12345")
	if got != "0/KCTR6DLpKmkAf8muzZqo1nDgQ=" {
		t.Errorf("signature mismatch, got %q", got)
	}
}

func statusProject(settings map[string]interface{}) core.Project {
	return core.Project{ID: "prj_test", Name: "test", Settings: settings, CreatedAt: time.Now()}
}

func statusMessage() core.Message {
	callback := "https://example.com/twilio-status"
	code := "30005"
	return core.Message{
		ID:          core.NewMessageID(),
		ProjectID:   "prj_test",
		Channel:     core.ChannelSMS,
		Direction:   core.DirectionOutbound,
		Provider:    "twilio",
		ProviderRef: "SM1234567890abcdef1234567890abcdef",
		From:        "+15555550100",
		To:          "+15005550006",
		BodyText:    "Hi",
		Status:      core.StatusDelivered,
		ErrorCode:   &code,
		CallbackURL: &callback,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestTwilio_StatusWebhook(t *testing.T) {
	adapter := twilio.New(nil)

	t.Run("builds signed form payload", func(t *testing.T) {
		msg := statusMessage()
		event := core.StatusEvent{ID: core.NewStatusEventID(), MessageID: msg.ID, Status: core.StatusDelivered, At: time.Now()}
		project := statusProject(map[string]interface{}{"twilio.auth_token": "tok123"})

		req, ok := adapter.StatusWebhook(msg, event, project)
		if !ok {
			t.Fatalf("expected a webhook request")
		}
		if req.URL != "https://example.com/twilio-status" {
			t.Errorf("expected callback URL, got %q", req.URL)
		}
		if ct := req.Headers["Content-Type"]; ct != "application/x-www-form-urlencoded" {
			t.Errorf("expected form content type, got %q", ct)
		}
		sig, ok := req.Headers["X-Twilio-Signature"]
		if !ok || sig == "" {
			t.Fatalf("expected signature header, got %v", req.Headers)
		}
		values, err := url.ParseQuery(string(req.Body))
		if err != nil {
			t.Fatalf("body is not form-encoded: %v", err)
		}
		for key, want := range map[string]string{
			"MessageSid":    "SM1234567890abcdef1234567890abcdef",
			"MessageStatus": "delivered",
			"SmsSid":        "SM1234567890abcdef1234567890abcdef",
			"SmsStatus":     "delivered",
			"AccountSid":    "",
			"From":          "+15555550100",
			"To":            "+15005550006",
			"ApiVersion":    "2010-04-01",
			"ErrorCode":     "30005",
		} {
			if values.Get(key) != want {
				t.Errorf("param %s: expected %q, got %q", key, want, values.Get(key))
			}
		}
		// The signature must verify against the same algorithm Twilio documents.
		unsigned := map[string]string{}
		for key := range values {
			unsigned[key] = values.Get(key)
		}
		if want := twilio.SignRequest("https://example.com/twilio-status", unsigned, "tok123"); want != sig {
			t.Errorf("signature mismatch: header %q, recomputed %q", sig, want)
		}
	})

	t.Run("no callback means no webhook", func(t *testing.T) {
		msg := statusMessage()
		msg.CallbackURL = nil
		event := core.StatusEvent{Status: core.StatusDelivered}
		if _, ok := adapter.StatusWebhook(msg, event, statusProject(nil)); ok {
			t.Errorf("expected no webhook without callback URL")
		}
	})

	t.Run("other providers ignored", func(t *testing.T) {
		msg := statusMessage()
		msg.Provider = "termii"
		event := core.StatusEvent{Status: core.StatusDelivered}
		if _, ok := adapter.StatusWebhook(msg, event, statusProject(nil)); ok {
			t.Errorf("expected no webhook for termii message")
		}
	})

	t.Run("observed AC token signs without warning path", func(t *testing.T) {
		msg := statusMessage()
		event := core.StatusEvent{Status: core.StatusSent}
		project := statusProject(map[string]interface{}{
			"twilio.account_sid":         "AC999",
			"twilio.observed_auth_token": "isseen",
		})
		req, ok := adapter.StatusWebhook(msg, event, project)
		if !ok {
			t.Fatalf("expected a webhook request")
		}
		values, _ := url.ParseQuery(string(req.Body))
		if values.Get("AccountSid") != "AC999" {
			t.Errorf("expected recorded account sid, got %q", values.Get("AccountSid"))
		}
		if want := twilio.SignRequest("https://example.com/twilio-status", mapOf(values), "isseen"); want != req.Headers["X-Twilio-Signature"] {
			t.Errorf("expected observed-token signature")
		}
	})

	t.Run("missing key sends unsigned", func(t *testing.T) {
		msg := statusMessage()
		event := core.StatusEvent{Status: core.StatusSent}
		req, ok := adapter.StatusWebhook(msg, event, statusProject(nil))
		if !ok {
			t.Fatalf("expected a webhook request")
		}
		if _, ok := req.Headers["X-Twilio-Signature"]; ok {
			t.Errorf("expected no signature without a key, got %v", req.Headers)
		}
	})
}

func mapOf(values url.Values) map[string]string {
	out := make(map[string]string, len(values))
	for key := range values {
		out[key] = values.Get(key)
	}
	return out
}

func TestTwilio_RecordCredentialOnSend(t *testing.T) {
	handler, store := setupTestTwilio(t)

	form := url.Values{"To": {testTo}, "From": {testFrom}, "Body": {"remember me"}}
	req := twilioRequest(t, http.MethodPost, "/2010-04-01/Accounts/"+testAccountSID+"/Messages.json", form, true)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "twilio", testAccountSID)
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	project, err := store.GetProject(context.Background(), projectID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if project.Settings["twilio.account_sid"] != testAccountSID {
		t.Errorf("expected recorded account sid, got %v", project.Settings)
	}
	if project.Settings["twilio.observed_auth_token"] != testAuthToken {
		t.Errorf("expected recorded auth token, got %v", project.Settings)
	}
	if !strings.Contains(rec.Body.String(), `"sid":"SM`) {
		t.Errorf("expected message sid in response: %s", rec.Body.String())
	}
}
