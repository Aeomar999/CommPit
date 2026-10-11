package webhooks

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"strings"
	"sync"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// Formatters looks up provider webhook formatters by provider name.
// Adapters register their optional capabilities here at startup.
type Formatters struct {
	mu      sync.RWMutex
	status  map[string]adapterkit.StatusNotifier
	inbound map[string]adapterkit.InboundNotifier
}

// NewFormatters builds an empty formatter registry.
func NewFormatters() *Formatters {
	return &Formatters{
		status:  map[string]adapterkit.StatusNotifier{},
		inbound: map[string]adapterkit.InboundNotifier{},
	}
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

// RegisterInbound registers an inbound-message formatter for a provider.
func (f *Formatters) RegisterInbound(provider string, notifier adapterkit.InboundNotifier) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inbound[provider] = notifier
}

// InboundFor returns the inbound formatter for a provider, if any.
func (f *Formatters) InboundFor(provider string) (adapterkit.InboundNotifier, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	notifier, ok := f.inbound[provider]
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

// Start subscribes to status transitions and inbound arrivals.
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
	d.subs = append(d.subs, d.bus.Subscribe(string(core.EventMessageCreated), func(event core.Event) {
		msg, ok := event.Payload.(*core.Message)
		if !ok || msg.Direction != core.DirectionInbound {
			return
		}
		if err := d.DispatchInbound(context.Background(), msg); err != nil {
			slog.Warn("webhooks: dispatch inbound", "message", msg.ID, "err", err)
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
	delivery := &core.WebhookDelivery{
		ID:        core.NewWebhookDeliveryID(),
		ProjectID: msg.ProjectID,
		MessageID: &msg.ID,
		Kind:      "status",
		URL:       req.URL,
		Payload:   decodeBody(req.Body, contentType(req.Headers)),
		Headers:   copyHeaders(req.Headers),
		Provider:  msg.Provider,
		Attempt:   0,
		Status:    core.WebhookPending,
		CreatedAt: d.clock.Now(),
	}
	if err := d.store.CreateWebhookDelivery(ctx, delivery); err != nil {
		return err
	}
	// Wake the worker after the row exists, so the drain cannot miss it.
	d.bus.Publish(ctx, core.Event{
		Type:      core.EventWebhookQueued,
		Payload:   delivery,
		ProjectID: msg.ProjectID,
		Timestamp: d.clock.Now(),
	})
	return nil
}

// DispatchInbound builds the provider inbound webhook for a received
// message and queues it. A nil error with no row means no webhook applies.
func (d *Dispatcher) DispatchInbound(ctx context.Context, msg *core.Message) error {
	notifier, ok := d.formatters.InboundFor(msg.Provider)
	if !ok {
		return nil
	}
	project, err := d.store.GetProject(ctx, msg.ProjectID)
	if err != nil {
		return err
	}
	req, ok := notifier.InboundWebhook(*msg, *project)
	if !ok {
		return nil
	}
	delivery := &core.WebhookDelivery{
		ID:        core.NewWebhookDeliveryID(),
		ProjectID: msg.ProjectID,
		MessageID: &msg.ID,
		Kind:      "inbound",
		URL:       req.URL,
		Payload:   decodeBody(req.Body, contentType(req.Headers)),
		Headers:   copyHeaders(req.Headers),
		Provider:  msg.Provider,
		Attempt:   0,
		Status:    core.WebhookPending,
		CreatedAt: d.clock.Now(),
	}
	if err := d.store.CreateWebhookDelivery(ctx, delivery); err != nil {
		return err
	}
	d.bus.Publish(ctx, core.Event{
		Type:      core.EventWebhookQueued,
		Payload:   delivery,
		ProjectID: msg.ProjectID,
		Timestamp: d.clock.Now(),
	})
	return nil
}

func contentType(headers map[string]string) string {
	for key, value := range headers {
		if strings.EqualFold(key, "Content-Type") {
			return value
		}
	}
	return ""
}

func copyHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers))
	for key, value := range headers {
		out[key] = value
	}
	return out
}

// Redispatch rebuilds a delivery as a fresh record, re-signing the payload
// with the provider formatter when the message context still exists
// (design spec §7.5: replay re-signs). When it doesn't — no linked
// message, unknown provider, formatter declined — it clones the original
// row verbatim so the replay still sends.
func (d *Dispatcher) Redispatch(ctx context.Context, original *core.WebhookDelivery) (*core.WebhookDelivery, error) {
	fresh := &core.WebhookDelivery{
		ID:             core.NewWebhookDeliveryID(),
		ProjectID:      original.ProjectID,
		MessageID:      original.MessageID,
		VerificationID: original.VerificationID,
		Kind:           original.Kind,
		URL:            original.URL,
		Payload:        original.Payload,
		Headers:        original.Headers,
		Provider:       original.Provider,
		Attempt:        0,
		Status:         core.WebhookPending,
		CreatedAt:      d.clock.Now(),
	}
	if original.MessageID != nil {
		if rebuilt, ok := d.rebuild(ctx, original, fresh); ok {
			fresh = rebuilt
		}
	}
	if err := d.store.CreateWebhookDelivery(ctx, fresh); err != nil {
		return nil, err
	}
	d.bus.Publish(ctx, core.Event{
		Type:      core.EventWebhookQueued,
		Payload:   fresh,
		ProjectID: fresh.ProjectID,
		Timestamp: d.clock.Now(),
	})
	return fresh, nil
}

// rebuild re-runs the provider formatter for the delivery's message. It
// reports false when re-signing is impossible, leaving the verbatim clone.
func (d *Dispatcher) rebuild(ctx context.Context, original, fresh *core.WebhookDelivery) (*core.WebhookDelivery, bool) {
	msg, err := d.store.GetMessage(ctx, original.ProjectID, *original.MessageID)
	if err != nil {
		slog.Warn("webhooks: replay: load message, cloning verbatim", "delivery", original.ID, "err", err)
		return nil, false
	}
	notifier, ok := d.formatters.StatusFor(msg.Provider)
	if !ok {
		return nil, false
	}
	project, err := d.store.GetProject(ctx, msg.ProjectID)
	if err != nil {
		slog.Warn("webhooks: replay: load project, cloning verbatim", "delivery", original.ID, "err", err)
		return nil, false
	}
	event := &core.StatusEvent{ID: core.NewStatusEventID(), MessageID: msg.ID, Status: msg.Status, At: d.clock.Now()}
	req, ok := notifier.StatusWebhook(*msg, *event, *project)
	if !ok {
		return nil, false
	}
	fresh.URL = req.URL
	fresh.Payload = decodeBody(req.Body, req.Headers["Content-Type"])
	fresh.Headers = map[string]string{}
	for key, value := range req.Headers {
		fresh.Headers[key] = value
	}
	return fresh, true
}

// decodeBody restores a notifier body to the payload map: JSON bodies parse
// as JSON, anything else as a form (Twilio's status callbacks).
func decodeBody(body []byte, contentType string) map[string]interface{} {
	payload := map[string]interface{}{}
	if strings.Contains(contentType, "json") {
		if err := json.Unmarshal(body, &payload); err != nil {
			slog.Warn("webhooks: replay: decode JSON body", "err", err)
		}
		return payload
	}
	if values, err := url.ParseQuery(string(body)); err == nil {
		for key := range values {
			payload[key] = values.Get(key)
		}
	}
	return payload
}
