package bus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/core"
)

func TestEventBus_PublishSubscribe(t *testing.T) {
	bus := NewEventBus()

	var received []core.Event
	var mu sync.Mutex

	sub := bus.Subscribe(string(core.EventMessageCreated), func(e core.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	ctx := context.Background()
	event := core.Event{
		Type:      core.EventMessageCreated,
		Payload:   "test",
		ProjectID: "prj_123",
		Timestamp: time.Now(),
	}

	bus.Publish(ctx, event)

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if len(received) != 1 {
		t.Errorf("expected 1 event, got %d", len(received))
	}
	mu.Unlock()

	sub.Unsubscribe()

	bus.Publish(ctx, event)
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if len(received) != 1 {
		t.Errorf("expected still 1 event after unsubscribe, got %d", len(received))
	}
	mu.Unlock()
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus()

	var count1, count2 int
	var mu sync.Mutex

	sub1 := bus.Subscribe(string(core.EventMessageStatus), func(e core.Event) {
		mu.Lock()
		count1++
		mu.Unlock()
	})

	sub2 := bus.Subscribe(string(core.EventMessageStatus), func(e core.Event) {
		mu.Lock()
		count2++
		mu.Unlock()
	})
	_ = sub2

	ctx := context.Background()
	event := core.Event{
		Type:      core.EventMessageStatus,
		Payload:   "test",
		ProjectID: "prj_123",
		Timestamp: time.Now(),
	}

	bus.Publish(ctx, event)
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if count1 != 1 {
		t.Errorf("sub1 expected 1 event, got %d", count1)
	}
	if count2 != 1 {
		t.Errorf("sub2 expected 1 event, got %d", count2)
	}
	mu.Unlock()

	sub1.Unsubscribe()

	bus.Publish(ctx, event)
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if count1 != 1 {
		t.Errorf("sub1 expected still 1 event, got %d", count1)
	}
	if count2 != 2 {
		t.Errorf("sub2 expected 2 events, got %d", count2)
	}
	mu.Unlock()
}

func TestEventBus_DifferentEventTypes(t *testing.T) {
	bus := NewEventBus()

	var received []core.EventType
	var mu sync.Mutex

	bus.Subscribe(string(core.EventMessageCreated), func(e core.Event) {
		mu.Lock()
		received = append(received, e.Type)
		mu.Unlock()
	})

	bus.Subscribe(string(core.EventVerificationUpdated), func(e core.Event) {
		mu.Lock()
		received = append(received, e.Type)
		mu.Unlock()
	})

	ctx := context.Background()
	bus.Publish(ctx, core.Event{Type: core.EventMessageCreated, ProjectID: "prj_1", Timestamp: time.Now()})
	bus.Publish(ctx, core.Event{Type: core.EventVerificationUpdated, ProjectID: "prj_1", Timestamp: time.Now()})
	bus.Publish(ctx, core.Event{Type: core.EventBatchUpdated, ProjectID: "prj_1", Timestamp: time.Now()})

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if len(received) != 2 {
		t.Errorf("expected 2 events, got %d", len(received))
	}
	hasCreated := false
	hasVerification := false
	for _, t := range received {
		if t == core.EventMessageCreated {
			hasCreated = true
		}
		if t == core.EventVerificationUpdated {
			hasVerification = true
		}
	}
	if !hasCreated {
		t.Error("expected EventMessageCreated")
	}
	if !hasVerification {
		t.Error("expected EventVerificationUpdated")
	}
	mu.Unlock()
}

func TestEventBus_UnsubscribeAll(t *testing.T) {
	bus := NewEventBus()

	var count int
	sub1 := bus.Subscribe(string(core.EventMessageCreated), func(e core.Event) { count++ })
	sub2 := bus.Subscribe(string(core.EventMessageCreated), func(e core.Event) { count++ })

	ctx := context.Background()
	bus.Publish(ctx, core.Event{Type: core.EventMessageCreated, ProjectID: "prj_1", Timestamp: time.Now()})
	time.Sleep(10 * time.Millisecond)

	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}

	sub1.Unsubscribe()
	sub2.Unsubscribe()

	bus.Publish(ctx, core.Event{Type: core.EventMessageCreated, ProjectID: "prj_1", Timestamp: time.Now()})
	time.Sleep(10 * time.Millisecond)

	if count != 2 {
		t.Errorf("expected still 2, got %d", count)
	}
}

func TestEventBus_ClosedBus(t *testing.T) {
	bus := NewEventBus()

	var count int
	sub := bus.Subscribe(string(core.EventMessageCreated), func(e core.Event) { count++ })

	// Can't easily test closed bus since we don't expose Close()
	// Just verify subscription works
	ctx := context.Background()
	bus.Publish(ctx, core.Event{Type: core.EventMessageCreated, ProjectID: "prj_1", Timestamp: time.Now()})
	time.Sleep(10 * time.Millisecond)

	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	sub.Unsubscribe()
}

func TestEventBus_ConcurrentPublishSubscribe(t *testing.T) {
	bus := NewEventBus()

	var wg sync.WaitGroup
	var mu sync.Mutex
	received := 0

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sub := bus.Subscribe(string(core.EventMessageCreated), func(e core.Event) {
				mu.Lock()
				received++
				mu.Unlock()
			})
			time.Sleep(time.Millisecond)
			sub.Unsubscribe()
		}()
	}

	ctx := context.Background()
	for i := 0; i < 100; i++ {
		bus.Publish(ctx, core.Event{Type: core.EventMessageCreated, ProjectID: "prj_1", Timestamp: time.Now()})
	}

	wg.Wait()
	time.Sleep(10 * time.Millisecond)

	// Should have received some events
	mu.Lock()
	if received == 0 {
		t.Error("expected some events received")
	}
	mu.Unlock()
}
