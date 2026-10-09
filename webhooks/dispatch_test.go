package webhooks

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

type fakeStatusNotifier struct {
	request *adapterkit.WebhookRequest
	ok      bool
}

func (f *fakeStatusNotifier) StatusWebhook(msg core.Message, event core.StatusEvent, project core.Project) (*adapterkit.WebhookRequest, bool) {
	return f.request, f.ok
}

func setupDispatchTest(t *testing.T) (core.Store, *core.FakeClock, *Formatters, string) {
	t.Helper()
	store, err := sqlite.NewStore(":memory:", 1)
	if err != nil {
		t.Fatalf("sqlite.NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	ctx := context.Background()
	prjID := core.NewProjectID()
	if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "dispatch", Settings: map[string]interface{}{}, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	clock := core.NewFakeClock()
	clock.Set(time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC))
	return store, clock, NewFormatters(), prjID
}

func seedStatusMessage(t *testing.T, store core.Store, clock *core.FakeClock, projectID, provider, callback string) *core.Message {
	t.Helper()
	var callbackURL *string
	if callback != "" {
		callbackURL = &callback
	}
	msg := &core.Message{
		ID:          core.NewMessageID(),
		ProjectID:   projectID,
		Channel:     core.ChannelSMS,
		Direction:   core.DirectionOutbound,
		Provider:    provider,
		ProviderRef: "SMabc",
		From:        "+15555550100",
		To:          "+15005550006",
		BodyText:    "Hi",
		Status:      core.StatusDelivered,
		CallbackURL: callbackURL,
		Segments:    1,
		Encoding:    "gsm7",
		CreatedAt:   clock.Now(),
		UpdatedAt:   clock.Now(),
	}
	if err := store.CreateMessage(context.Background(), msg); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	return msg
}

func TestDispatcher_DispatchStatus(t *testing.T) {
	store, clock, formatters, prjID := setupDispatchTest(t)
	formatters.RegisterStatus("twilio", &fakeStatusNotifier{
		request: &adapterkit.WebhookRequest{
			URL:     "https://example.com/twilio-status",
			Body:    []byte(url.Values{"MessageSid": {"SMabc"}, "MessageStatus": {"delivered"}}.Encode()),
			Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded", "X-Twilio-Signature": "sig"},
		},
		ok: true,
	})
	dispatcher := NewDispatcher(store, bus.NewEventBus(), clock, formatters)

	msg := seedStatusMessage(t, store, clock, prjID, "twilio", "https://example.com/twilio-status")
	event := &core.StatusEvent{ID: core.NewStatusEventID(), MessageID: msg.ID, Status: core.StatusDelivered, At: clock.Now()}

	if err := dispatcher.DispatchStatus(context.Background(), msg, event); err != nil {
		t.Fatalf("DispatchStatus: %v", err)
	}

	deliveries, err := store.ListPendingWebhooks(context.Background(), clock.Now(), 10)
	if err != nil {
		t.Fatalf("ListPendingWebhooks: %v", err)
	}
	if len(deliveries) != 1 {
		t.Fatalf("expected 1 queued delivery, got %d", len(deliveries))
	}
	got := deliveries[0]
	if got.URL != "https://example.com/twilio-status" {
		t.Errorf("expected callback URL, got %q", got.URL)
	}
	if got.Payload["MessageSid"] != "SMabc" || got.Payload["MessageStatus"] != "delivered" {
		t.Errorf("unexpected payload: %v", got.Payload)
	}
	if got.Headers["X-Twilio-Signature"] != "sig" {
		t.Errorf("expected signature header, got %v", got.Headers)
	}
	if got.MessageID == nil || *got.MessageID != msg.ID || got.Kind != "status" || got.Attempt != 0 {
		t.Errorf("unexpected delivery linkage: %+v", got)
	}
}

func TestDispatcher_Skips(t *testing.T) {
	store, clock, formatters, prjID := setupDispatchTest(t)
	formatters.RegisterStatus("twilio", &fakeStatusNotifier{ok: false})
	dispatcher := NewDispatcher(store, bus.NewEventBus(), clock, formatters)

	t.Run("no formatter for provider", func(t *testing.T) {
		msg := seedStatusMessage(t, store, clock, prjID, "termii", "https://example.com/x")
		event := &core.StatusEvent{ID: core.NewStatusEventID(), MessageID: msg.ID, Status: core.StatusDelivered, At: clock.Now()}
		if err := dispatcher.DispatchStatus(context.Background(), msg, event); err != nil {
			t.Fatalf("DispatchStatus: %v", err)
		}
	})

	t.Run("notifier declines", func(t *testing.T) {
		msg := seedStatusMessage(t, store, clock, prjID, "twilio", "")
		event := &core.StatusEvent{ID: core.NewStatusEventID(), MessageID: msg.ID, Status: core.StatusDelivered, At: clock.Now()}
		if err := dispatcher.DispatchStatus(context.Background(), msg, event); err != nil {
			t.Fatalf("DispatchStatus: %v", err)
		}
	})

	deliveries, err := store.ListPendingWebhooks(context.Background(), clock.Now(), 10)
	if err != nil {
		t.Fatalf("ListPendingWebhooks: %v", err)
	}
	if len(deliveries) != 0 {
		t.Errorf("expected no deliveries, got %d", len(deliveries))
	}
}

func TestDispatcher_BusSubscription(t *testing.T) {
	store, clock, formatters, prjID := setupDispatchTest(t)
	_ = prjID
	_ = clock
	_ = formatters
	_ = store
}
