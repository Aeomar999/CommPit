package bus

import (
	"context"
	"sync"

	"github.com/Aeomar999/CommPit/core"
)

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[core.EventType][]*subscription
	closed      bool
}

type subscription struct {
	id      string
	handler core.EventHandler
	closed  bool
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[core.EventType][]*subscription),
	}
}

func (eb *EventBus) Publish(ctx context.Context, event core.Event) {
	eb.mu.RLock()
	subs := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	if len(subs) == 0 {
		return
	}

	for _, sub := range subs {
		if sub.closed {
			continue
		}
		sub.handler(event)
	}
}

func (eb *EventBus) Subscribe(eventType string, handler core.EventHandler) core.Subscription {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return &noopSubscription{}
	}

	sub := &subscription{
		id:      generateID(),
		handler: handler,
	}

	eb.subscribers[core.EventType(eventType)] = append(eb.subscribers[core.EventType(eventType)], sub)

	return sub
}

func (s *subscription) Unsubscribe() {
	s.closed = true
}

type noopSubscription struct{}

func (n *noopSubscription) Unsubscribe() {}

func generateID() string {
	return "sub_" + randomString(16)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
