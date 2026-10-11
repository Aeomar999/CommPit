package twilio_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/adapters/twilio"
	"github.com/Aeomar999/CommPit/core"
)

func inboundTwilioMessage() core.Message {
	return core.Message{
		ID:          core.NewMessageID(),
		ProjectID:   "prj_test",
		Channel:     core.ChannelSMS,
		Direction:   core.DirectionInbound,
		Provider:    "twilio",
		ProviderRef: "SM1234567890abcdef1234567890abcdef",
		From:        "+15005550006",
		To:          "+15555550100",
		BodyText:    "Hello back",
		Status:      core.StatusReceived,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestTwilio_InboundWebhook(t *testing.T) {
	adapter := twilio.New(nil)

	t.Run("builds signed form payload", func(t *testing.T) {
		msg := inboundTwilioMessage()
		project := statusProject(map[string]interface{}{
			"twilio.account_sid": "AC999",
			"twilio.auth_token":  "tok123",
			"twilio.inbound_url": "https://example.com/twilio-inbound",
		})

		req, ok := adapter.InboundWebhook(msg, project)
		if !ok {
			t.Fatalf("expected a webhook request")
		}
		if req.URL != "https://example.com/twilio-inbound" {
			t.Errorf("expected inbound URL, got %q", req.URL)
		}
		values, err := url.ParseQuery(string(req.Body))
		if err != nil {
			t.Fatalf("body is not form-encoded: %v", err)
		}
		for key, want := range map[string]string{
			"MessageSid": "SM1234567890abcdef1234567890abcdef",
			"AccountSid": "AC999",
			"From":       "+15005550006",
			"To":         "+15555550100",
			"Body":       "Hello back",
			"NumMedia":   "0",
		} {
			if values.Get(key) != want {
				t.Errorf("param %s: expected %q, got %q", key, want, values.Get(key))
			}
		}
		unsigned := map[string]string{}
		for key := range values {
			unsigned[key] = values.Get(key)
		}
		if want := twilio.SignRequest("https://example.com/twilio-inbound", unsigned, "tok123"); want != req.Headers["X-Twilio-Signature"] {
			t.Errorf("signature mismatch")
		}
	})

	t.Run("outbound messages ignored", func(t *testing.T) {
		msg := inboundTwilioMessage()
		msg.Direction = core.DirectionOutbound
		project := statusProject(map[string]interface{}{"twilio.inbound_url": "https://example.com/x"})
		if _, ok := adapter.InboundWebhook(msg, project); ok {
			t.Errorf("expected no webhook for outbound message")
		}
	})

	t.Run("no inbound URL means no webhook", func(t *testing.T) {
		msg := inboundTwilioMessage()
		if _, ok := adapter.InboundWebhook(msg, statusProject(nil)); ok {
			t.Errorf("expected no webhook without inbound URL")
		}
	})

	t.Run("other providers ignored", func(t *testing.T) {
		msg := inboundTwilioMessage()
		msg.Provider = "native"
		project := statusProject(map[string]interface{}{"twilio.inbound_url": "https://example.com/x"})
		if _, ok := adapter.InboundWebhook(msg, project); ok {
			t.Errorf("expected no webhook for native message")
		}
	})
}

func parseTwiml(t *testing.T, adapter *twilio.Adapter, body string, status int) ([]core.OutboundReply, error) {
	t.Helper()
	resp := &http.Response{
		StatusCode:    status,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
	defer resp.Body.Close()
	return adapter.ParseReply(resp)
}

func TestTwilio_ParseReply(t *testing.T) {
	adapter := twilio.New(nil)

	t.Run("messages with and without to", func(t *testing.T) {
		replies, err := parseTwiml(t, adapter,
			`<?xml version="1.0" encoding="UTF-8"?><Response><Message to="+15005550006">Hi there</Message><Message>Same to you</Message></Response>`, 200)
		if err != nil {
			t.Fatalf("ParseReply: %v", err)
		}
		if len(replies) != 2 {
			t.Fatalf("expected 2 replies, got %+v", replies)
		}
		if replies[0].To != "+15005550006" || replies[0].Body != "Hi there" {
			t.Errorf("first reply mismatch: %+v", replies[0])
		}
		if replies[1].To != "" || replies[1].Body != "Same to you" {
			t.Errorf("second reply mismatch: %+v", replies[1])
		}
		for i, r := range replies {
			if r.Provider != "twilio" {
				t.Errorf("reply %d: expected twilio provider, got %+v", i, r)
			}
			if len(r.ProviderRef) != 34 || r.ProviderRef[:2] != "SM" {
				t.Errorf("reply %d: expected SM provider ref, got %q", i, r.ProviderRef)
			}
		}
		if replies[0].ProviderRef == replies[1].ProviderRef {
			t.Errorf("expected distinct provider refs")
		}
	})

	t.Run("empty response means no replies", func(t *testing.T) {
		replies, err := parseTwiml(t, adapter, `<Response/>`, 200)
		if err != nil {
			t.Fatalf("ParseReply: %v", err)
		}
		if len(replies) != 0 {
			t.Errorf("expected no replies, got %+v", replies)
		}
	})

	t.Run("malformed XML errors", func(t *testing.T) {
		if _, err := parseTwiml(t, adapter, `<Response><Message>oops`, 200); err == nil {
			t.Errorf("expected error, got nil")
		}
	})
}
