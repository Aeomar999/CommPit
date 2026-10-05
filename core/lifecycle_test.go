package core

import (
	"context"
	"sync"
	"testing"
	"time"
)

type dummyStore struct {
	Store
}

func (d *dummyStore) UpdateMessage(ctx context.Context, m *Message) error {
	return nil
}

func (d *dummyStore) CreateStatusEvent(ctx context.Context, e *StatusEvent) error {
	return nil
}

type dummyBus struct {
	Bus
}

func (d *dummyBus) Publish(ctx context.Context, event Event) {}

func TestLifecycleRunner_StopIdempotent(t *testing.T) {
	svc := &Service{
		clock:     RealClock{},
		stepDelay: 100 * time.Millisecond,
		store:     &dummyStore{},
		bus:       &dummyBus{},
	}
	runner := NewLifecycleRunner(svc)

	msg := &Message{
		ID:        "msg_test123",
		ProjectID: "prj_test",
		Status:    StatusQueued,
	}

	runner.Schedule(msg, nil)

	// Call Stop sequentially multiple times - must not panic.
	runner.Stop()
	runner.Stop()
	runner.Stop()
}

func TestLifecycleRunner_StopConcurrent(t *testing.T) {
	svc := &Service{
		clock:     RealClock{},
		stepDelay: 100 * time.Millisecond,
		store:     &dummyStore{},
		bus:       &dummyBus{},
	}
	runner := NewLifecycleRunner(svc)

	for i := 0; i < 10; i++ {
		msg := &Message{
			ID:        NewMessageID(),
			ProjectID: "prj_test",
			Status:    StatusQueued,
		}
		runner.Schedule(msg, nil)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runner.Stop()
		}()
	}
	wg.Wait()
}
