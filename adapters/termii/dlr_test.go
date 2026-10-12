package termii_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/adapters/termii"
	"github.com/Aeomar999/CommPit/core"
)

func TestTermii_SignPayload(t *testing.T) {
	// Golden vector computed with Python's hmac.new(key, body, sha512).
	body := `{"channel":"generic","cost":null,"id":"12345","message":"Hi","message_id":"12345","receiver":"+15005550006","sender":"Me","sent_at":"2026-10-09T12:00:00.000000Z","status":"Delivered","type":"outbound"}`
	want := "5acad38313962f9b9c88d0e2901ab098a44ed978b7951b6167bd7aa480c9c33041446b14b8c4a2da1caa9dd395d91fde63e78467e08be6748ec8c1d2f477d05b"
	if got := termii.SignPayload([]byte(body), "secret-key"); got != want {
		t.Errorf("signature mismatch, got %q", got)
	}
}

func dlrProject(settings map[string]interface{}) core.Project {
	return core.Project{ID: "prj_test", Name: "test", Settings: settings, CreatedAt: time.Now()}
}

func dlrMessage() core.Message {
	sentAt := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	return core.Message{
		ID:          core.NewMessageID(),
		ProjectID:   "prj_test",
		Channel:     core.ChannelSMS,
		Direction:   core.DirectionOutbound,
		Provider:    "termii",
		ProviderRef: "123456789",
		From:        "Acme",
		To:          "+2348012345678",
		BodyText:    "Your code is 123456",
		Status:      core.StatusDelivered,
		Segments:    1,
		CreatedAt:   sentAt,
		UpdatedAt:   sentAt,
	}
}

func decodeDLR(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var decoded map[string]interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("DLR body is not JSON: %v", err)
	}
	return decoded
}

func TestTermii_StatusWebhook(t *testing.T) {
	adapter := termii.New(nil)

	t.Run("builds signed JSON delivery report", func(t *testing.T) {
		msg := dlrMessage()
		event := core.StatusEvent{ID: core.NewStatusEventID(), MessageID: msg.ID, Status: core.StatusDelivered, At: time.Now()}
		project := dlrProject(map[string]interface{}{
			"termii.webhook_url":      "https://example.com/termii-dlr",
			"termii.observed_api_key": "secret-key",
		})

		req, ok := adapter.StatusWebhook(msg, event, project)
		if !ok {
			t.Fatalf("expected a webhook request")
		}
		if req.URL != "https://example.com/termii-dlr" {
			t.Errorf("expected account-level webhook URL, got %q", req.URL)
		}
		if ct := req.Headers["Content-Type"]; ct != "application/json" {
			t.Errorf("expected JSON content type, got %q", ct)
		}
		decoded := decodeDLR(t, req.Body)
		for key, want := range map[string]interface{}{
			"type":       "outbound",
			"id":         "123456789",
			"message_id": "123456789",
			"receiver":   "+2348012345678",
			"sender":     "Acme",
			"message":    "Your code is 123456",
			"sent_at":    "2026-10-09T12:00:00.000000Z",
			"cost":       nil,
			"status":     "Delivered",
			"channel":    "generic",
		} {
			if decoded[key] != want {
				t.Errorf("field %s: expected %v, got %v", key, want, decoded[key])
			}
		}
		if want := termii.SignPayload(req.Body, "secret-key"); want != req.Headers["X-Termii-Signature"] {
			t.Errorf("signature mismatch: header %q, recomputed %q", req.Headers["X-Termii-Signature"], want)
		}
	})

	t.Run("status vocabulary", func(t *testing.T) {
		msg := dlrMessage()
		project := dlrProject(map[string]interface{}{
			"termii.webhook_url":      "https://example.com/termii-dlr",
			"termii.observed_api_key": "secret-key",
		})
		for status, want := range map[core.MessageStatus]string{
			core.StatusQueued:      "Message Sent",
			core.StatusSent:        "Message Sent",
			core.StatusDelivered:   "Delivered",
			core.StatusFailed:      "Message Failed",
			core.StatusUndelivered: "Message Failed",
		} {
			event := core.StatusEvent{ID: core.NewStatusEventID(), MessageID: msg.ID, Status: status, At: time.Now()}
			req, ok := adapter.StatusWebhook(msg, event, project)
			if !ok {
				t.Fatalf("status %s: expected a webhook request", status)
			}
			if got := decodeDLR(t, req.Body)["status"]; got != want {
				t.Errorf("status %s: expected %q, got %v", status, want, got)
			}
		}
	})

	t.Run("no webhook URL means no webhook", func(t *testing.T) {
		msg := dlrMessage()
		msg.CallbackURL = strptr("https://example.com/per-request")
		event := core.StatusEvent{Status: core.StatusDelivered}
		if _, ok := adapter.StatusWebhook(msg, event, dlrProject(nil)); ok {
			t.Errorf("expected no webhook without the account-level URL")
		}
	})

	t.Run("other providers ignored", func(t *testing.T) {
		msg := dlrMessage()
		msg.Provider = "twilio"
		event := core.StatusEvent{Status: core.StatusDelivered}
		project := dlrProject(map[string]interface{}{"termii.webhook_url": "https://example.com/x"})
		if _, ok := adapter.StatusWebhook(msg, event, project); ok {
			t.Errorf("expected no webhook for twilio message")
		}
	})

	t.Run("missing key sends unsigned", func(t *testing.T) {
		msg := dlrMessage()
		event := core.StatusEvent{Status: core.StatusSent}
		project := dlrProject(map[string]interface{}{"termii.webhook_url": "https://example.com/x"})
		req, ok := adapter.StatusWebhook(msg, event, project)
		if !ok {
			t.Fatalf("expected a webhook request")
		}
		if _, ok := req.Headers["X-Termii-Signature"]; ok {
			t.Errorf("expected no signature without a key, got %v", req.Headers)
		}
	})
}

func strptr(s string) *string { return &s }

func TestTermii_RecordAPIKeyOnSend(t *testing.T) {
	handler, store := setupTestTermii(t)

	req := termiiRequest(t, "/api/sms/send", map[string]any{
		"to":   testToNG,
		"from": testSender,
		"sms":  "remember me",
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
	project, err := store.GetProject(context.Background(), projectID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if project.Settings["termii.observed_api_key"] != testAPIKey {
		t.Errorf("expected recorded api key, got %v", project.Settings)
	}
}
