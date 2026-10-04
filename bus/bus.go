package bus

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/Aeomar999/CommPit/core"
	"github.com/oklog/ulid/v2"
)

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[core.EventType][]*subscription
	closed      atomic.Bool
}

type subscription struct {
	id        string
	handler   core.EventHandler
	eventType core.EventType
	bus       *EventBus
	closed    atomic.Bool
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[core.EventType][]*subscription),
	}
}

func (eb *EventBus) Publish(ctx context.Context, event core.Event) {
	eb.mu.RLock()
	subs := eb.subscribers[event.Type]
	// Copy the slice to avoid holding the lock while calling handlers
	subsCopy := make([]*subscription, len(subs))
	copy(subsCopy, subs)
	eb.mu.RUnlock()

	if len(subsCopy) == 0 {
		return
	}

	for _, sub := range subsCopy {
		if sub.closed.Load() {
			continue
		}
		sub.handler(event)
	}
}

func (eb *EventBus) Subscribe(eventType string, handler core.EventHandler) core.Subscription {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed.Load() {
		return &noopSubscription{}
	}

	et := core.EventType(eventType)
	sub := &subscription{
		id:        "sub_" + ulid.Make().String(),
		handler:   handler,
		eventType: et,
		bus:       eb,
	}

	eb.subscribers[et] = append(eb.subscribers[et], sub)

	return sub
}

func (s *subscription) Unsubscribe() {
	if !s.closed.CompareAndSwap(false, true) {
		return // Already unsubscribed
	}
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()

	subs := s.bus.subscribers[s.eventType]
	for i, sub := range subs {
		if sub == s {
			// Remove from slice
			s.bus.subscribers[s.eventType] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

type noopSubscription struct{}

func (n *noopSubscription) Unsubscribe() {}

func generateID() string {
	return "sub_" + ulid.Make().String()
}
