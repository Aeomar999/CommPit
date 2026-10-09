package webhooks

import (
	"context"
	"log/slog"
	"net/url"
	"sync"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// Formatters looks up provider webhook formatters by provider name.
// Adapters register their optional capabilities here at startup.
type Formatters struct {
	mu     sync.RWMutex
	status map[string]adapterkit.StatusNotifier
}

// NewFormatters builds an empty formatter registry.
func NewFormatters() *Formatters {
	return &Formatters{status: map[string]adapterkit.StatusNotifier{}}
}

// RegisterStatus registers a status-callback formatter for a provider.
func (f *Formatters) RegisterStatus(provider string, notifier adapterkit.StatusNotifier) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status[provider] = notifier
}

// StatusFor returns the status formatter for a provider, if any.
func (f *Formatters) StatusFor(provider string) (adapterkit.StatusNotifier, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	notifier, ok := f.status[provider]
	return notifier, ok
}

// Dispatcher turns domain events into queued webhook deliveries using the
// registered provider formatters. It creates rows; the Worker sends them.
type Dispatcher struct {
	store      core.Store
	bus        core.Bus
	clock      core.Clock
	formatters *Formatters

	subs     []core.Subscription
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewDispatcher wires event-to-delivery routing.
func NewDispatcher(store core.Store, bus core.Bus, clock core.Clock, formatters *Formatters) *Dispatcher {
	return &Dispatcher{
		store:      store,
		bus:        bus,
		clock:      clock,
		formatters: formatters,
		stopCh:     make(chan struct{}),
	}
}

// Start subscribes to status transitions. Idempotent enough for one call;
// Stop unsubscribes.
func (d *Dispatcher) Start() {
	d.subs = append(d.subs, d.bus.Subscribe(string(core.EventMessageStatus), func(event core.Event) {
		statusEvent, ok := event.Payload.(*core.StatusEvent)
		if !ok {
			return
		}
		msg, err := d.store.GetMessage(context.Background(), event.ProjectID, statusEvent.MessageID)
		if err != nil {
			slog.Warn("webhooks: dispatch status: load message", "message", statusEvent.MessageID, "err", err)
			return
		}
		if err := d.DispatchStatus(context.Background(), msg, statusEvent); err != nil {
			slog.Warn("webhooks: dispatch status", "message", msg.ID, "err", err)
		}
	}))
}

// Stop unsubscribes. Idempotent.
func (d *Dispatcher) Stop() {
	d.stopOnce.Do(func() {
		close(d.stopCh)
		for _, sub := range d.subs {
			sub.Unsubscribe()
		}
	})
}

// DispatchStatus builds the provider status webhook for a transition and
// queues it. A nil error with no row means no webhook applies.
func (d *Dispatcher) DispatchStatus(ctx context.Context, msg *core.Message, event *core.StatusEvent) error {
	notifier, ok := d.formatters.StatusFor(msg.Provider)
	if !ok {
		return nil
	}
	project, err := d.store.GetProject(ctx, msg.ProjectID)
	if err != nil {
		return err
	}
	req, ok := notifier.StatusWebhook(*msg, *event, *project)
	if !ok {
		return nil
	}
	payload := map[string]interface{}{}
	if values, err := url.ParseQuery(string(req.Body)); err == nil {
		for key := range values {
			payload[key] = values.Get(key)
		}
	}
	headers := map[string]string{}
	for key, value := range req.Headers {
		headers[key] = value
	}
	return d.store.CreateWebhookDelivery(ctx, &core.WebhookDelivery{
		ID:        core.NewWebhookDeliveryID(),
		ProjectID: msg.ProjectID,
		MessageID: &msg.ID,
		Kind:      "status",
		URL:       req.URL,
		Payload:   payload,
		Headers:   headers,
		Attempt:   0,
		Status:    core.WebhookPending,
		CreatedAt: d.clock.Now(),
	})
}
