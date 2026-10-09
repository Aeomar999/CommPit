package webhooks

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/Aeomar999/CommPit/core"
)

// Worker delivers persistent webhook callbacks. Bus events wake it; there
// is no fixed polling interval. Retry waits ride the injected clock, so
// tests advance time deterministically.
type Worker struct {
	store  core.Store
	bus    core.Bus
	clock  core.Clock
	sender Sender
	cfg    Config

	subs     []core.Subscription
	notifyCh chan struct{}
	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// NewWorker wires a delivery worker. Zero Config values take defaults.
func NewWorker(store core.Store, bus core.Bus, clock core.Clock, sender Sender, cfg Config) *Worker {
	return &Worker{
		store:    store,
		bus:      bus,
		clock:    clock,
		sender:   sender,
		cfg:      cfg.withDefaults(),
		notifyCh: make(chan struct{}, 1),
		stopCh:   make(chan struct{}),
	}
}

// wakeEventTypes are the domain events that may have produced work.
var wakeEventTypes = []core.EventType{
	core.EventMessageCreated,
	core.EventMessageStatus,
	core.EventVerificationUpdated,
	core.EventBatchUpdated,
}

// Start subscribes to bus events, drains pre-existing rows, and runs the
// retry loop until Stop or ctx cancellation.
func (w *Worker) Start(ctx context.Context) {
	for _, eventType := range wakeEventTypes {
		w.subs = append(w.subs, w.bus.Subscribe(string(eventType), func(core.Event) {
			w.Notify()
		}))
	}
	w.wg.Add(1)
	go w.loop(ctx)
	// Drain rows left by producers that ran before Start (and rows that
	// survived a restart, since the queue is persistent).
	w.Notify()
}

// Notify wakes one drain, coalescing concurrent wake-ups.
func (w *Worker) Notify() {
	select {
	case w.notifyCh <- struct{}{}:
	default:
	}
}

// Stop unsubscribes and waits for the loop. Idempotent.
func (w *Worker) Stop() {
	w.stopOnce.Do(func() {
		close(w.stopCh)
		for _, sub := range w.subs {
			sub.Unsubscribe()
		}
	})
	w.wg.Wait()
}

func (w *Worker) loop(ctx context.Context) {
	defer w.wg.Done()
	var retryTimer <-chan time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-w.notifyCh:
			retryTimer = w.drain(ctx)
		case <-retryTimer:
			retryTimer = w.drain(ctx)
		}
	}
}

// drain sends every due delivery and returns a channel firing at the next
// future retry, or nil when nothing is scheduled.
func (w *Worker) drain(ctx context.Context) <-chan time.Time {
	deliveries, err := w.store.ListPendingWebhooks(ctx, w.clock.Now(), w.cfg.BatchSize)
	if err != nil {
		slog.Error("webhooks: list pending", "err", err)
		return nil
	}
	now := w.clock.Now()
	var earliest *time.Time
	fold := func(at time.Time) {
		if earliest == nil || at.Before(*earliest) {
			earliestCopy := at
			earliest = &earliestCopy
		}
	}
	for _, delivery := range deliveries {
		if delivery.NextRetryAt != nil && delivery.NextRetryAt.After(now) {
			fold(*delivery.NextRetryAt)
			continue
		}
		w.deliver(ctx, delivery)
		// deliver mutates the row in place; a newly scheduled retry must
		// arm a wake-up or it would sleep until the next bus event.
		if delivery.NextRetryAt != nil {
			fold(*delivery.NextRetryAt)
		}
	}
	if earliest == nil {
		return nil
	}
	wait := earliest.Sub(now)
	if wait < 0 {
		wait = 0
	}
	return w.clock.After(wait)
}

// deliver sends one delivery and records the attempt.
func (w *Worker) deliver(ctx context.Context, delivery *core.WebhookDelivery) {
	delivery.Attempt++

	payload, err := json.Marshal(delivery.Payload)
	if err != nil {
		slog.Error("webhooks: encode payload, failing delivery", "id", delivery.ID, "err", err)
		w.finish(ctx, delivery, 0, nil, false)
		return
	}

	sendCtx, cancel := context.WithTimeout(ctx, w.cfg.Timeout)
	defer cancel()
	status, body, err := w.sender.Send(sendCtx, delivery.URL, payload, delivery.Headers)
	if err == nil && status >= 200 && status < 300 {
		w.finish(ctx, delivery, status, body, true)
		return
	}
	if err != nil {
		slog.Warn("webhooks: send failed", "id", delivery.ID, "attempt", delivery.Attempt, "err", err)
	}
	w.finish(ctx, delivery, status, body, false)
}

// finish persists the attempt outcome: success, another retry, or terminal
// failure. Terminal outcomes publish webhook.delivered for the live log.
func (w *Worker) finish(ctx context.Context, delivery *core.WebhookDelivery, status int, body []byte, ok bool) {
	delivery.ResponseStatus = &status
	text := string(body)
	delivery.ResponseBody = &text
	if ok {
		delivery.Status = core.WebhookSucceeded
		delivery.NextRetryAt = nil
	} else if delivery.Attempt >= w.cfg.MaxAttempts {
		delivery.Status = core.WebhookFailed
		delivery.NextRetryAt = nil
	} else {
		delivery.Status = core.WebhookPending
		waitIndex := delivery.Attempt - 1
		if waitIndex >= len(w.cfg.Backoff) {
			waitIndex = len(w.cfg.Backoff) - 1
		}
		next := w.clock.Now().Add(w.cfg.Backoff[waitIndex])
		delivery.NextRetryAt = &next
	}
	if err := w.store.UpdateWebhookDelivery(ctx, delivery); err != nil {
		slog.Error("webhooks: record attempt", "id", delivery.ID, "err", err)
		return
	}
	if delivery.Status != core.WebhookPending {
		w.bus.Publish(ctx, core.Event{
			Type:      core.EventWebhookDelivered,
			Payload:   delivery,
			ProjectID: delivery.ProjectID,
			Timestamp: w.clock.Now(),
		})
	}
}
