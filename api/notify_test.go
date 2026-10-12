package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

func TestSignNativePayload(t *testing.T) {
	// Golden vector computed with Python's hmac.new(key, body, sha256).
	body := []byte(`{"event":"message.status","message":{"id":"msg_01"}}`)
	want := "sha256=4224cade420b919b31fdce24b7c07b59e463bf8f15eff431d4a139d9e0842385"
	if got := SignNativePayload(body, "project-api-key"); got != want {
		t.Errorf("signature mismatch, got %q", got)
	}
}

func setupNotifyTest(t *testing.T) (core.Store, core.Project) {
	t.Helper()
	store, err := sqlite.NewStore(":memory:", 1)
	if err != nil {
		t.Fatalf("sqlite.NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	ctx := context.Background()
	project := core.Project{ID: "prj_notify", Name: "notify", Settings: map[string]interface{}{}, CreatedAt: time.Now()}
	if err := store.CreateProject(ctx, &project); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	return store, project
}

func seedCredential(t *testing.T, store core.Store, projectID, provider, key string, at time.Time) {
	t.Helper()
	if err := store.CreateCredential(context.Background(), &core.Credential{
		ID:        core.NewRequestLogID(),
		Provider:  provider,
		Key:       key,
		ProjectID: projectID,
		CreatedAt: at,
	}); err != nil {
		t.Fatalf("CreateCredential: %v", err)
	}
}

func notifyMessage(projectID string) core.Message {
	callback := "https://example.com/native-hook"
	at := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	return core.Message{
		ID:          core.NewMessageID(),
		ProjectID:   projectID,
		Channel:     core.ChannelSMS,
		Direction:   core.DirectionOutbound,
		Provider:    "native",
		ProviderRef: "ref-1",
		From:        "+15555550100",
		To:          "+15005550006",
		BodyText:    "Hi",
		Status:      core.StatusDelivered,
		CallbackURL: &callback,
		Segments:    1,
		CreatedAt:   at,
		UpdatedAt:   at,
	}
}

func decodeNative(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var decoded map[string]interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("native body is not JSON: %v", err)
	}
	return decoded
}

func TestNotifier_InboundWebhook(t *testing.T) {
	newInbound := func() core.Message {
		msg := notifyMessage("prj_notify")
		msg.Direction = core.DirectionInbound
		msg.Status = core.StatusReceived
		return msg
	}

	t.Run("builds signed event payload", func(t *testing.T) {
		store, project := setupNotifyTest(t)
		base := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
		seedCredential(t, store, project.ID, "native", "hook-key", base)

		notifier := NewNotifier(store)
		msg := newInbound()
		req, ok := notifier.InboundWebhook(msg, dlrProjectShim(project, "https://example.com/native-inbound"))
		if !ok {
			t.Fatalf("expected a webhook request")
		}
		if req.URL != "https://example.com/native-inbound" {
			t.Errorf("expected inbound URL, got %q", req.URL)
		}
		decoded := decodeNative(t, req.Body)
		if decoded["event"] != "message.inbound" {
			t.Errorf("expected message.inbound event, got %v", decoded["event"])
		}
		if want := SignNativePayload(req.Body, "hook-key"); want != req.Headers["X-Mocksms-Signature"] {
			t.Errorf("signature mismatch: header %q recomputed %q", req.Headers["X-Mocksms-Signature"], want)
		}
	})

	t.Run("outbound messages ignored", func(t *testing.T) {
		store, project := setupNotifyTest(t)
		notifier := NewNotifier(store)
		msg := notifyMessage(project.ID)
		if _, ok := notifier.InboundWebhook(msg, projectWithInboundURL(project)); ok {
			t.Errorf("expected no webhook for outbound message")
		}
	})

	t.Run("no inbound URL means no webhook", func(t *testing.T) {
		store, project := setupNotifyTest(t)
		notifier := NewNotifier(store)
		if _, ok := notifier.InboundWebhook(newInbound(), project); ok {
			t.Errorf("expected no webhook without inbound URL")
		}
	})

	t.Run("other providers ignored", func(t *testing.T) {
		store, project := setupNotifyTest(t)
		notifier := NewNotifier(store)
		msg := newInbound()
		msg.Provider = "twilio"
		if _, ok := notifier.InboundWebhook(msg, projectWithInboundURL(project)); ok {
			t.Errorf("expected no webhook for twilio message")
		}
	})
}

func projectWithInboundURL(project core.Project) core.Project {
	if project.Settings == nil {
		project.Settings = map[string]interface{}{}
	}
	project.Settings["native.inbound_url"] = "https://example.com/native-inbound"
	return project
}

func dlrProjectShim(project core.Project, url string) core.Project {
	if project.Settings == nil {
		project.Settings = map[string]interface{}{}
	}
	project.Settings["native.inbound_url"] = url
	return project
}

func TestNotifier_StatusWebhook(t *testing.T) {
	t.Run("builds signed event payload with oldest key", func(t *testing.T) {
		store, project := setupNotifyTest(t)
		base := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
		seedCredential(t, store, project.ID, "native", "newer-key", base.Add(time.Hour))
		seedCredential(t, store, project.ID, "native", "oldest-key", base)
		seedCredential(t, store, project.ID, "twilio", "ACxxx", base.Add(-time.Hour))

		notifier := NewNotifier(store)
		msg := notifyMessage(project.ID)
		event := core.StatusEvent{ID: core.NewStatusEventID(), MessageID: msg.ID, Status: core.StatusDelivered, At: base}

		req, ok := notifier.StatusWebhook(msg, event, project)
		if !ok {
			t.Fatalf("expected a webhook request")
		}
		if req.URL != "https://example.com/native-hook" {
			t.Errorf("expected callback URL, got %q", req.URL)
		}
		decoded := decodeNative(t, req.Body)
		if decoded["event"] != "message.status" {
			t.Errorf("expected message.status event, got %v", decoded["event"])
		}
		inner, ok := decoded["message"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected message object, got %v", decoded["message"])
		}
		if inner["id"] != msg.ID || inner["body_text"] != "Hi" {
			t.Errorf("unexpected message payload: %v", inner)
		}
		if want := SignNativePayload(req.Body, "oldest-key"); want != req.Headers["X-Mocksms-Signature"] {
			t.Errorf("expected oldest-key signature, header %q recomputed %q",
				req.Headers["X-Mocksms-Signature"], want)
		}
	})

	t.Run("no callback means no webhook", func(t *testing.T) {
		store, project := setupNotifyTest(t)
		notifier := NewNotifier(store)
		msg := notifyMessage(project.ID)
		msg.CallbackURL = nil
		event := core.StatusEvent{Status: core.StatusDelivered}
		if _, ok := notifier.StatusWebhook(msg, event, project); ok {
			t.Errorf("expected no webhook without callback URL")
		}
	})

	t.Run("other providers ignored", func(t *testing.T) {
		store, project := setupNotifyTest(t)
		notifier := NewNotifier(store)
		msg := notifyMessage(project.ID)
		msg.Provider = "smtp"
		event := core.StatusEvent{Status: core.StatusDelivered}
		if _, ok := notifier.StatusWebhook(msg, event, project); ok {
			t.Errorf("expected no webhook for smtp message")
		}
	})

	t.Run("missing key sends unsigned", func(t *testing.T) {
		store, project := setupNotifyTest(t)
		notifier := NewNotifier(store)
		msg := notifyMessage(project.ID)
		event := core.StatusEvent{Status: core.StatusSent}
		req, ok := notifier.StatusWebhook(msg, event, project)
		if !ok {
			t.Fatalf("expected a webhook request")
		}
		if _, ok := req.Headers["X-Mocksms-Signature"]; ok {
			t.Errorf("expected no signature without a key, got %v", req.Headers)
		}
	})
}
