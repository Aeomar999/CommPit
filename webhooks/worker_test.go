package webhooks

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

type sentCall struct {
	url     string
	payload []byte
	headers map[string]string
}

func setupWorkerTest(t *testing.T, send func() (int, []byte, error)) (*Worker, core.Store, *core.FakeClock, core.Bus, chan sentCall, chan core.Event) {
	t.Helper()
	store, err := sqlite.NewStore(":memory:", 1)
	if err != nil {
		t.Fatalf("sqlite.NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	ctx := context.Background()
	prjID := core.NewProjectID()
	if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "hooks", Settings: map[string]interface{}{}, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	eventBus := bus.NewEventBus()
	clock := core.NewFakeClock()
	clock.Set(time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC))

	sent := make(chan sentCall, 16)
	sender := SenderFunc(func(ctx context.Context, url string, payload []byte, headers map[string]string) (int, []byte, error) {
		sent <- sentCall{url: url, payload: payload, headers: headers}
		return send()
	})

	events := make(chan core.Event, 8)
	sub := eventBus.Subscribe(string(core.EventWebhookDelivered), func(e core.Event) {
		events <- e
	})
	t.Cleanup(sub.Unsubscribe)

	worker := NewWorker(store, eventBus, clock, sender, Config{})
	return worker, store, clock, eventBus, sent, events
}

func seedDelivery(t *testing.T, store core.Store, clock *core.FakeClock, projectID string) *core.WebhookDelivery {
	t.Helper()
	delivery := &core.WebhookDelivery{
		ID:        core.NewWebhookDeliveryID(),
		ProjectID: projectID,
		Kind:      "status",
		URL:       "https://example.com/hook",
		Payload:   map[string]interface{}{"event": "message.status"},
		Headers:   map[string]string{"Content-Type": "application/json"},
		Attempt:   0,
		Status:    core.WebhookPending,
		CreatedAt: clock.Now(),
	}
	if err := store.CreateWebhookDelivery(context.Background(), delivery); err != nil {
		t.Fatalf("CreateWebhookDelivery: %v", err)
	}
	return delivery
}

func projectOf(t *testing.T, store core.Store) string {
	t.Helper()
	projects, _, err := store.ListProjects(context.Background(), 10, "")
	if err != nil || len(projects) == 0 {
		t.Fatalf("ListProjects: %v", err)
	}
	return projects[0].ID
}

func storedDelivery(t *testing.T, store core.Store, id string) *core.WebhookDelivery {
	t.Helper()
	stored, err := store.GetWebhookDelivery(context.Background(), id)
	if err != nil {
		t.Fatalf("GetWebhookDelivery: %v", err)
	}
	return stored
}

func waitSend(t *testing.T, sent chan sentCall) sentCall {
	t.Helper()
	select {
	case call := <-sent:
		return call
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for webhook send")
		return sentCall{}
	}
}

func waitEvent(t *testing.T, events chan core.Event) core.Event {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for webhook.delivered event")
		return core.Event{}
	}
}

func TestWorker_DeliversDue(t *testing.T) {
	worker, store, clock, _, sent, events := setupWorkerTest(t, func() (int, []byte, error) {
		return http.StatusOK, []byte(`ok`), nil
	})
	prjID := projectOf(t, store)
	seedDelivery(t, store, clock, prjID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Start(ctx)
	defer worker.Stop()

	call := waitSend(t, sent)
	if call.url != "https://example.com/hook" {
		t.Errorf("expected hook URL, got %q", call.url)
	}

	event := waitEvent(t, events)
	delivery, ok := event.Payload.(*core.WebhookDelivery)
	if !ok {
		t.Fatalf("expected *WebhookDelivery payload, got %T", event.Payload)
	}
	if delivery.Status != core.WebhookSucceeded {
		t.Errorf("expected succeeded, got %q", delivery.Status)
	}
	if delivery.Attempt != 1 {
		t.Errorf("expected attempt 1, got %d", delivery.Attempt)
	}

	stored := storedDelivery(t, store, delivery.ID)
	if stored.Status != core.WebhookSucceeded || stored.ResponseStatus == nil || *stored.ResponseStatus != http.StatusOK {
		t.Errorf("unexpected stored outcome: %+v", stored)
	}
	if stored.ResponseBody == nil || *stored.ResponseBody != "ok" {
		t.Errorf("expected stored response body, got %+v", stored.ResponseBody)
	}
}

// gateFixture runs a worker whose sender parks inside every send until the
// test releases it. A received receipt therefore pins the worker: the store
// shows exactly the previous attempt's recorded outcome, with no timing
// assumptions.
// gateFixture runs a worker whose sender parks inside every send until the
// test releases it. Tests must seed rows BEFORE starting the worker:
// Start drains immediately, and a row created after an empty drain would
// otherwise sleep until the next bus event.
type gateFixture struct {
	worker  *Worker
	store   core.Store
	clock   *core.FakeClock
	sent    chan sentCall
	release chan struct{}
	events  chan core.Event
}

func setupGateWorker(t *testing.T, send func() (int, []byte, error)) *gateFixture {
	t.Helper()
	store, err := sqlite.NewStore(":memory:", 1)
	if err != nil {
		t.Fatalf("sqlite.NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	ctx := context.Background()
	prjID := core.NewProjectID()
	if err := store.CreateProject(ctx, &core.Project{ID: prjID, Name: "hooks", Settings: map[string]interface{}{}, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	eventBus := bus.NewEventBus()
	clock := core.NewFakeClock()
	clock.Set(time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC))

	sent := make(chan sentCall, 16)
	release := make(chan struct{})
	sender := SenderFunc(func(ctx context.Context, url string, payload []byte, headers map[string]string) (int, []byte, error) {
		sent <- sentCall{url: url, payload: payload, headers: headers}
		<-release
		return send()
	})

	events := make(chan core.Event, 8)
	sub := eventBus.Subscribe(string(core.EventWebhookDelivered), func(e core.Event) {
		events <- e
	})
	t.Cleanup(sub.Unsubscribe)

	worker := NewWorker(store, eventBus, clock, sender, Config{})
	return &gateFixture{worker: worker, store: store, clock: clock, sent: sent, release: release, events: events}
}

func TestWorker_RetrySequence(t *testing.T) {
	fix := setupGateWorker(t, func() (int, []byte, error) {
		return 0, nil, errors.New("connection refused")
	})
	prjID := projectOf(t, fix.store)
	seed := seedDelivery(t, fix.store, fix.clock, prjID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fix.worker.Start(ctx)
	defer fix.worker.Stop()

	start := fix.clock.Now()
	waits := []time.Duration{time.Second, 5 * time.Second, 30 * time.Second, 2 * time.Minute, 10 * time.Minute}
	elapsed := time.Duration(0)
	// Six sends total: assert the recorded state behind each parked send,
	// then release it and advance to the next scheduled retry.
	for i := 0; i <= len(waits); i++ {
		waitSend(t, fix.sent)
		stored := storedDelivery(t, fix.store, seed.ID)
		if stored.Attempt != i {
			t.Fatalf("send %d: expected recorded Attempt %d, got %d", i+1, i, stored.Attempt)
		}
		if i > 0 {
			if stored.Status != core.WebhookPending {
				t.Fatalf("send %d: expected pending, got %q", i+1, stored.Status)
			}
			if stored.NextRetryAt == nil || !stored.NextRetryAt.Equal(start.Add(elapsed)) {
				t.Fatalf("send %d: expected next retry %v, got %v", i+1, start.Add(elapsed), stored.NextRetryAt)
			}
		}
		fix.release <- struct{}{}
		if i < len(waits) {
			fix.clock.BlockUntilWaiters(1)
			fix.clock.Advance(waits[i])
			elapsed += waits[i]
		}
	}

	// Sixth send failed: terminal failure with an event.
	event := waitEvent(t, fix.events)
	delivery, ok := event.Payload.(*core.WebhookDelivery)
	if !ok || delivery.Status != core.WebhookFailed {
		t.Fatalf("expected failed delivery event, got %+v", event.Payload)
	}
	stored := storedDelivery(t, fix.store, seed.ID)
	if stored.Attempt != 6 || stored.Status != core.WebhookFailed || stored.NextRetryAt != nil {
		t.Errorf("expected failed after 6 attempts, got %+v", stored)
	}
}

func TestWorker_Non2xxRetries(t *testing.T) {
	fix := setupGateWorker(t, func() (int, []byte, error) {
		return http.StatusInternalServerError, []byte("boom"), nil
	})
	seed := seedDelivery(t, fix.store, fix.clock, projectOf(t, fix.store))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fix.worker.Start(ctx)
	defer fix.worker.Stop()

	waitSend(t, fix.sent)
	stored := storedDelivery(t, fix.store, seed.ID)
	if stored.Attempt != 0 {
		t.Fatalf("parked send: expected recorded Attempt 0, got %d", stored.Attempt)
	}
	fix.release <- struct{}{}

	fix.clock.BlockUntilWaiters(1)
	fix.clock.Advance(time.Hour)
	waitSend(t, fix.sent)
	stored = storedDelivery(t, fix.store, seed.ID)
	if stored.Status != core.WebhookPending || stored.Attempt != 1 {
		t.Errorf("expected pending after attempt 1, got %+v", stored)
	}
	if stored.ResponseStatus == nil || *stored.ResponseStatus != http.StatusInternalServerError {
		t.Errorf("expected recorded 500 response, got %+v", stored)
	}
	if stored.NextRetryAt == nil {
		t.Errorf("expected a scheduled retry, got %+v", stored)
	}
	fix.release <- struct{}{}
}

func TestWorker_BusWake(t *testing.T) {
	worker, store, clock, eventBus, sent, events := setupWorkerTest(t, func() (int, []byte, error) {
		return http.StatusOK, []byte(`ok`), nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Start(ctx)
	defer worker.Stop()

	// A row created after Start is picked up via bus wake-up, with no clock
	// movement. The terminal event proves the send fully finished.
	seedDelivery(t, store, clock, projectOf(t, store))
	eventBus.Publish(ctx, core.Event{Type: core.EventMessageStatus, ProjectID: projectOf(t, store), Timestamp: clock.Now()})
	waitSend(t, sent)
	event := waitEvent(t, events)
	if delivery, ok := event.Payload.(*core.WebhookDelivery); !ok || delivery.Status != core.WebhookSucceeded {
		t.Fatalf("expected succeeded delivery event, got %+v", event.Payload)
	}
}

func TestWorker_StopIdempotent(t *testing.T) {
	worker, _, _, _, _, _ := setupWorkerTest(t, func() (int, []byte, error) {
		return http.StatusOK, []byte(`ok`), nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Start(ctx)
	worker.Stop()
	worker.Stop()
}
