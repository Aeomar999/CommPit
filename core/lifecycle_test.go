package core

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockLifecycleStore struct {
	Store
	mu           sync.Mutex
	messages     map[string]*Message
	statusEvents []*StatusEvent
	batches      map[string]*Batch
}

func newMockLifecycleStore() *mockLifecycleStore {
	return &mockLifecycleStore{
		messages: make(map[string]*Message),
		batches:  make(map[string]*Batch),
	}
}

func (m *mockLifecycleStore) GetMessage(ctx context.Context, projectID, messageID string) (*Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[messageID]
	if !ok || msg.ProjectID != projectID {
		return nil, NewNotFound("message not found", "id")
	}
	cp := *msg
	return &cp, nil
}

func (m *mockLifecycleStore) UpdateMessage(ctx context.Context, msg *Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *msg
	m.messages[msg.ID] = &cp
	return nil
}

func (m *mockLifecycleStore) CreateMessage(ctx context.Context, msg *Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *msg
	m.messages[msg.ID] = &cp
	return nil
}

func (m *mockLifecycleStore) CreateStatusEvent(ctx context.Context, e *StatusEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusEvents = append(m.statusEvents, e)
	return nil
}

func (m *mockLifecycleStore) GetStatusEvents(ctx context.Context, messageID string) ([]*StatusEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*StatusEvent
	for _, e := range m.statusEvents {
		if e.MessageID == messageID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockLifecycleStore) CreateBatch(ctx context.Context, b *Batch) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *b
	m.batches[b.ID] = &cp
	return nil
}

func (m *mockLifecycleStore) GetBatch(ctx context.Context, projectID, batchID string) (*Batch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.batches[batchID]
	if !ok || b.ProjectID != projectID {
		return nil, NewNotFound("batch not found", "id")
	}
	cp := *b
	return &cp, nil
}

func (m *mockLifecycleStore) UpdateBatch(ctx context.Context, b *Batch) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *b
	m.batches[b.ID] = &cp
	return nil
}

func (m *mockLifecycleStore) RecomputeBatchCounts(ctx context.Context, projectID, batchID string) (*Batch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	batch, ok := m.batches[batchID]
	if !ok || batch.ProjectID != projectID {
		return nil, NewNotFound("batch not found", "id")
	}

	counts := make(map[string]int)
	total := 0
	for _, msg := range m.messages {
		if msg.ProjectID == projectID && msg.BatchID != nil && *msg.BatchID == batchID {
			counts[string(msg.Status)]++
			total++
		}
	}
	batch.Counts = counts
	batch.Total = total
	cp := *batch
	return &cp, nil
}

func (m *mockLifecycleStore) ListInFlightMessages(ctx context.Context) ([]*Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*Message
	for _, msg := range m.messages {
		if msg.Status == StatusQueued || msg.Status == StatusSent {
			cp := *msg
			result = append(result, &cp)
		}
	}
	return result, nil
}

type mockBus struct {
	mu        sync.Mutex
	events    []Event
	publishCh chan Event
}

func (b *mockBus) Publish(ctx context.Context, event Event) {
	b.mu.Lock()
	b.events = append(b.events, event)
	ch := b.publishCh
	b.mu.Unlock()
	if ch != nil {
		ch <- event
	}
}

func (b *mockBus) Subscribe(eventType string, handler EventHandler) Subscription {
	return nil
}

func TestLifecycleRunner_StopIdempotent(t *testing.T) {
	store := newMockLifecycleStore()
	svc := &Service{
		clock:     RealClock{},
		stepDelay: 100 * time.Millisecond,
		store:     store,
		bus:       &mockBus{},
	}
	runner := NewLifecycleRunner(svc)

	msg := &Message{
		ID:        "msg_test123",
		ProjectID: "prj_test",
		Status:    StatusQueued,
	}
	_ = store.CreateMessage(context.Background(), msg)

	runner.Schedule(msg.ProjectID, msg.ID, nil)

	// Call Stop sequentially multiple times - must not panic.
	runner.Stop()
	runner.Stop()
	runner.Stop()
}

func TestLifecycleRunner_StopConcurrent(t *testing.T) {
	store := newMockLifecycleStore()
	svc := &Service{
		clock:     RealClock{},
		stepDelay: 100 * time.Millisecond,
		store:     store,
		bus:       &mockBus{},
	}
	runner := NewLifecycleRunner(svc)

	for i := 0; i < 10; i++ {
		msg := &Message{
			ID:        NewMessageID(),
			ProjectID: "prj_test",
			Status:    StatusQueued,
		}
		_ = store.CreateMessage(context.Background(), msg)
		runner.Schedule(msg.ProjectID, msg.ID, nil)
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

func TestLifecycleRunner_DelayZero_SingleTransitionEvents(t *testing.T) {
	store := newMockLifecycleStore()
	bus := &mockBus{}
	clock := NewFakeClock()

	svc := &Service{
		clock:     clock,
		stepDelay: 0,
		store:     store,
		bus:       bus,
	}
	runner := NewLifecycleRunner(svc)

	msg := &Message{
		ID:        NewMessageID(),
		ProjectID: "prj_test",
		Status:    StatusQueued,
	}
	if err := store.CreateMessage(context.Background(), msg); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}

	runner.Schedule(msg.ProjectID, msg.ID, nil)

	// Verify message in store is Delivered
	updated, err := store.GetMessage(context.Background(), msg.ProjectID, msg.ID)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if updated.Status != StatusDelivered {
		t.Errorf("expected StatusDelivered, got %s", updated.Status)
	}

	// Verify exactly two status events: sent then delivered
	events, err := store.GetStatusEvents(context.Background(), msg.ID)
	if err != nil {
		t.Fatalf("GetStatusEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected exactly 2 status events, got %d", len(events))
	}
	if events[0].Status != StatusSent {
		t.Errorf("event 0: expected StatusSent, got %s", events[0].Status)
	}
	if events[1].Status != StatusDelivered {
		t.Errorf("event 1: expected StatusDelivered, got %s", events[1].Status)
	}

	// Verify event ID prefix is "sev_"
	for idx, e := range events {
		if !strings.HasPrefix(e.ID, StatusEventIDPrefix) {
			t.Errorf("event %d: expected prefix %s, got %s", idx, StatusEventIDPrefix, e.ID)
		}
	}
}

func TestLifecycleRunner_DelayPositive_AdvancesOnlyWhenClockMoves(t *testing.T) {
	store := newMockLifecycleStore()
	publishCh := make(chan Event, 10)
	bus := &mockBus{publishCh: publishCh}
	clock := NewFakeClock()

	delay := 100 * time.Millisecond
	svc := &Service{
		clock:     clock,
		stepDelay: delay,
		store:     store,
		bus:       bus,
	}
	runner := NewLifecycleRunner(svc)
	defer runner.Stop()

	msg := &Message{
		ID:        NewMessageID(),
		ProjectID: "prj_test",
		Status:    StatusQueued,
	}
	if err := store.CreateMessage(context.Background(), msg); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}

	runner.Schedule(msg.ProjectID, msg.ID, nil)

	// Block until the initial timer is registered
	clock.BlockUntilWaiters(1)

	// Before clock advances, message must remain Queued
	current, err := store.GetMessage(context.Background(), msg.ProjectID, msg.ID)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if current.Status != StatusQueued {
		t.Errorf("expected StatusQueued before clock advances, got %s", current.Status)
	}

	events, _ := store.GetStatusEvents(context.Background(), msg.ID)
	if len(events) != 0 {
		t.Fatalf("expected 0 events before clock advance, got %d", len(events))
	}

	// Advance clock by delay (100ms) -> transitions to Sent
	clock.Advance(delay)

	// Wait for Sent status event
	select {
	case ev := <-publishCh:
		if ev.Type != EventMessageStatus {
			t.Fatalf("unexpected event: %v", ev.Type)
		}
		se, ok := ev.Payload.(*StatusEvent)
		if !ok || se.Status != StatusSent {
			t.Fatalf("expected StatusSent, got %v", se.Status)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for StatusSent event")
	}

	// Block until the next step timer is registered
	clock.BlockUntilWaiters(1)

	// Advance clock by delay again (100ms) -> transitions to Delivered
	clock.Advance(delay)

	// Wait for Delivered status event
	select {
	case ev := <-publishCh:
		if ev.Type != EventMessageStatus {
			t.Fatalf("unexpected event: %v", ev.Type)
		}
		se, ok := ev.Payload.(*StatusEvent)
		if !ok || se.Status != StatusDelivered {
			t.Fatalf("expected StatusDelivered, got %v", se.Status)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for StatusDelivered event")
	}

	current, _ = store.GetMessage(context.Background(), msg.ProjectID, msg.ID)
	if current.Status != StatusDelivered {
		t.Fatalf("expected StatusDelivered after 2nd clock advance, got %s", current.Status)
	}

	// Further clock advancement should not produce more events
	clock.Advance(delay)
	events, _ = store.GetStatusEvents(context.Background(), msg.ID)
	if len(events) != 2 {
		t.Errorf("expected exactly 2 status events, got %d", len(events))
	}
}

func TestLifecycleRunner_AsyncFail_EndsInUndelivered(t *testing.T) {
	store := newMockLifecycleStore()
	bus := &mockBus{}
	clock := NewFakeClock()

	svc := &Service{
		clock:     clock,
		stepDelay: 0,
		store:     store,
		bus:       bus,
	}
	runner := NewLifecycleRunner(svc)

	msg := &Message{
		ID:        NewMessageID(),
		ProjectID: "prj_test",
		Status:    StatusQueued,
	}
	if err := store.CreateMessage(context.Background(), msg); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}

	simResult := &SimResult{
		AsyncFail: &AsyncFail{
			ErrorCode:    "30008",
			ErrorMessage: "Unknown delivery failure",
		},
	}

	runner.Schedule(msg.ProjectID, msg.ID, simResult)

	updated, err := store.GetMessage(context.Background(), msg.ProjectID, msg.ID)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if updated.Status != StatusUndelivered {
		t.Fatalf("expected StatusUndelivered, got %s", updated.Status)
	}
	if updated.ErrorCode == nil || *updated.ErrorCode != "30008" {
		t.Errorf("expected ErrorCode 30008, got %v", updated.ErrorCode)
	}
	if updated.ErrorMessage == nil || *updated.ErrorMessage != "Unknown delivery failure" {
		t.Errorf("expected ErrorMessage 'Unknown delivery failure', got %v", updated.ErrorMessage)
	}

	events, _ := store.GetStatusEvents(context.Background(), msg.ID)
	if len(events) != 2 {
		t.Fatalf("expected 2 status events, got %d", len(events))
	}
	if events[1].Status != StatusUndelivered {
		t.Errorf("expected 2nd event StatusUndelivered, got %s", events[1].Status)
	}
	if events[1].ErrorCode == nil || *events[1].ErrorCode != "30008" {
		t.Errorf("expected 2nd event error code 30008, got %v", events[1].ErrorCode)
	}
}

func TestLifecycleRunner_ResumeQueuedAndSent(t *testing.T) {
	store := newMockLifecycleStore()
	bus := &mockBus{}
	clock := NewFakeClock()

	svc := &Service{
		clock:     clock,
		stepDelay: 0,
		store:     store,
		bus:       bus,
	}
	runner := NewLifecycleRunner(svc)

	msgQueued := &Message{ID: "msg_q", ProjectID: "prj_1", Status: StatusQueued}
	msgSent := &Message{ID: "msg_s", ProjectID: "prj_2", Status: StatusSent}
	msgDelivered := &Message{ID: "msg_d", ProjectID: "prj_3", Status: StatusDelivered}

	_ = store.CreateMessage(context.Background(), msgQueued)
	_ = store.CreateMessage(context.Background(), msgSent)
	_ = store.CreateMessage(context.Background(), msgDelivered)

	runner.ResumeQueuedAndSent(context.Background())

	q, _ := store.GetMessage(context.Background(), "prj_1", "msg_q")
	if q.Status != StatusDelivered {
		t.Errorf("expected msgQueued to be delivered, got %s", q.Status)
	}

	s, _ := store.GetMessage(context.Background(), "prj_2", "msg_s")
	if s.Status != StatusDelivered {
		t.Errorf("expected msgSent to be delivered, got %s", s.Status)
	}

	d, _ := store.GetMessage(context.Background(), "prj_3", "msg_d")
	if d.Status != StatusDelivered {
		t.Errorf("expected msgDelivered to remain delivered, got %s", d.Status)
	}
}

func TestLifecycleRunner_Batch100Messages_EndsWithCorrectCounts(t *testing.T) {
	store := newMockLifecycleStore()
	bus := &mockBus{}
	clock := NewFakeClock()

	svc := &Service{
		clock:     clock,
		stepDelay: 0,
		store:     store,
		bus:       bus,
	}
	runner := NewLifecycleRunner(svc)

	projectID := "prj_batch_test"
	batchID := NewBatchID()
	batch := &Batch{
		ID:        batchID,
		ProjectID: projectID,
		Total:     100,
		Counts:    map[string]int{string(StatusQueued): 100},
	}
	_ = store.CreateBatch(context.Background(), batch)

	var messageIDs []string
	for i := 0; i < 100; i++ {
		m := &Message{
			ID:        NewMessageID(),
			ProjectID: projectID,
			BatchID:   &batchID,
			Status:    StatusQueued,
		}
		_ = store.CreateMessage(context.Background(), m)
		messageIDs = append(messageIDs, m.ID)
	}

	var wg sync.WaitGroup
	for _, id := range messageIDs {
		wg.Add(1)
		go func(mID string) {
			defer wg.Done()
			runner.Schedule(projectID, mID, nil)
		}(id)
	}
	wg.Wait()

	finalBatch, err := store.GetBatch(context.Background(), projectID, batchID)
	if err != nil {
		t.Fatalf("GetBatch: %v", err)
	}

	if finalBatch.Total != 100 {
		t.Errorf("expected batch Total 100, got %d", finalBatch.Total)
	}
	if finalBatch.Counts[string(StatusDelivered)] != 100 {
		t.Errorf("expected batch delivered 100, got %d", finalBatch.Counts[string(StatusDelivered)])
	}
	if finalBatch.Counts[string(StatusQueued)] != 0 {
		t.Errorf("expected batch queued 0, got %d", finalBatch.Counts[string(StatusQueued)])
	}
	if finalBatch.Counts[string(StatusSent)] != 0 {
		t.Errorf("expected batch sent 0, got %d", finalBatch.Counts[string(StatusSent)])
	}
}
