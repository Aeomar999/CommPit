package core

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/phone"
)

type testServiceStore struct {
	Store
	mu           sync.Mutex
	messages     map[string]*Message
	batches      map[string]*Batch
	unsubscribes map[string]bool
	statusEvents []*StatusEvent
}

func newTestServiceStore() *testServiceStore {
	return &testServiceStore{
		messages:     make(map[string]*Message),
		batches:      make(map[string]*Batch),
		unsubscribes: make(map[string]bool),
	}
}

func (s *testServiceStore) CreateMessage(ctx context.Context, m *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *m
	s.messages[m.ID] = &cp
	return nil
}

func (s *testServiceStore) GetMessage(ctx context.Context, projectID, id string) (*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.messages[id]
	if !ok || m.ProjectID != projectID {
		return nil, NewNotFound("message not found", "id")
	}
	cp := *m
	return &cp, nil
}

func (s *testServiceStore) UpdateMessage(ctx context.Context, m *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *m
	s.messages[m.ID] = &cp
	return nil
}

func (s *testServiceStore) ListMessages(ctx context.Context, projectID string, filter MessageFilter) ([]*Message, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*Message
	for _, m := range s.messages {
		if m.ProjectID != projectID {
			continue
		}
		if filter.To != nil && m.To != *filter.To {
			continue
		}
		cp := *m
		result = append(result, &cp)
	}
	return result, "", nil
}

func (s *testServiceStore) CreateBatch(ctx context.Context, b *Batch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *b
	s.batches[b.ID] = &cp
	return nil
}

func (s *testServiceStore) GetBatch(ctx context.Context, projectID, id string) (*Batch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.batches[id]
	if !ok || b.ProjectID != projectID {
		return nil, NewNotFound("batch not found", "id")
	}
	cp := *b
	return &cp, nil
}

func (s *testServiceStore) UpdateBatch(ctx context.Context, b *Batch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *b
	s.batches[b.ID] = &cp
	return nil
}

func (s *testServiceStore) RecomputeBatchCounts(ctx context.Context, projectID, batchID string) (*Batch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.batches[batchID]
	if !ok || b.ProjectID != projectID {
		return nil, NewNotFound("batch not found", "id")
	}
	counts := make(map[string]int)
	total := 0
	for _, m := range s.messages {
		if m.ProjectID == projectID && m.BatchID != nil && *m.BatchID == batchID {
			counts[string(m.Status)]++
			total++
		}
	}
	b.Counts = counts
	b.Total = total
	cp := *b
	return &cp, nil
}

func (s *testServiceStore) CreateStatusEvent(ctx context.Context, e *StatusEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusEvents = append(s.statusEvents, e)
	return nil
}

func (s *testServiceStore) IsUnsubscribed(ctx context.Context, projectID, number string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.unsubscribes[projectID+":"+number], nil
}

func (s *testServiceStore) CreateUnsubscribe(ctx context.Context, u *Unsubscribe) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unsubscribes[u.ProjectID+":"+u.Number] = true
	return nil
}

func (s *testServiceStore) DeleteUnsubscribe(ctx context.Context, projectID, number string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.unsubscribes, projectID+":"+number)
	return nil
}

func (s *testServiceStore) Transaction(ctx context.Context, fn func(txStore Store) error) error {
	return fn(s)
}

func (s *testServiceStore) ListInFlightMessages(ctx context.Context) ([]*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*Message
	for _, m := range s.messages {
		if m.Status == StatusQueued || m.Status == StatusSent {
			cp := *m
			result = append(result, &cp)
		}
	}
	return result, nil
}

func setupTestService() (*Service, *testServiceStore, *FakeClock) {
	store := newTestServiceStore()
	clock := NewFakeClock()
	clock.Set(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	sim := NewSimulator()
	bus := &mockBus{}

	svc := NewService(ServiceConfig{
		Store:     store,
		Bus:       bus,
		Simulator: sim,
		Clock:     clock,
		StepDelay: 0,
		PhoneMode: phone.ModeValid,
	})
	return svc, store, clock
}

func TestRecipient_Normalization(t *testing.T) {
	svc, store, _ := setupTestService()
	ctx := context.Background()
	projectID := "prj_test1"

	// Sending to formatted number +1 (500) 555-0006 should store normalized E.164
	resp, err := svc.SendMessage(ctx, projectID, SendRequest{
		Channel:  ChannelSMS,
		From:     "+15005550000",
		To:       []string{"+1 (500) 555-0006"},
		BodyText: "Hello formatted recipient",
	})
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if resp.Message.To != "+15005550006" {
		t.Errorf("expected normalized To '+15005550006', got %q", resp.Message.To)
	}

	saved, err := store.GetMessage(ctx, projectID, resp.Message.ID)
	if err != nil {
		t.Fatalf("GetMessage failed: %v", err)
	}
	if saved.To != "+15005550006" {
		t.Errorf("expected stored To '+15005550006', got %q", saved.To)
	}
}

func TestRecipient_StopThenStartReenablesSending(t *testing.T) {
	svc, _, _ := setupTestService()
	ctx := context.Background()
	projectID := "prj_test2"
	recipient := "+1 (500) 555-0006"
	normalizedRecipient := "+15005550006"

	// 1. Initial send succeeds
	_, err := svc.SendMessage(ctx, projectID, SendRequest{
		Channel:  ChannelSMS,
		From:     "+15005550000",
		To:       []string{recipient},
		BodyText: "First message",
	})
	if err != nil {
		t.Fatalf("initial send failed: %v", err)
	}

	// 2. Inbound STOP from that recipient
	inboundMsg, err := svc.ReceiveInbound(ctx, projectID, InboundRequest{
		From: recipient,
		To:   "+15005550000",
		Body: "STOP",
	})
	if err != nil {
		t.Fatalf("ReceiveInbound STOP failed: %v", err)
	}
	if inboundMsg.Status != StatusReceived {
		t.Errorf("expected inbound message status 'received', got %q", inboundMsg.Status)
	}
	if inboundMsg.From != normalizedRecipient {
		t.Errorf("expected normalized From %q, got %q", normalizedRecipient, inboundMsg.From)
	}

	// 3. Outbound send to that recipient must now fail with unsubscribed
	_, err = svc.SendMessage(ctx, projectID, SendRequest{
		Channel:  ChannelSMS,
		From:     "+15005550000",
		To:       []string{recipient},
		BodyText: "Message after STOP",
	})
	if err == nil {
		t.Fatal("expected SendMessage after STOP to fail, but it succeeded")
	}
	if !IsError(err, ErrCodeUnsubscribed) {
		t.Fatalf("expected error ErrCodeUnsubscribed, got %v", err)
	}

	// 4. Inbound START from that recipient must be accepted and resubscribe
	inboundStart, err := svc.ReceiveInbound(ctx, projectID, InboundRequest{
		From: recipient,
		To:   "+15005550000",
		Body: "START",
	})
	if err != nil {
		t.Fatalf("ReceiveInbound START failed: %v", err)
	}
	if inboundStart.Status != StatusReceived {
		t.Errorf("expected inbound START status 'received', got %q", inboundStart.Status)
	}

	// 5. Outbound send must succeed again after START
	_, err = svc.SendMessage(ctx, projectID, SendRequest{
		Channel:  ChannelSMS,
		From:     "+15005550000",
		To:       []string{recipient},
		BodyText: "Message after START",
	})
	if err != nil {
		t.Fatalf("send after START failed: %v", err)
	}
}

func TestRecipient_CallbackURLOnlyStoredWhenProvided(t *testing.T) {
	svc, _, _ := setupTestService()
	ctx := context.Background()
	projectID := "prj_test3"

	// Message without callback URL
	resp1, err := svc.SendMessage(ctx, projectID, SendRequest{
		Channel:     ChannelSMS,
		From:        "+15005550000",
		To:          []string{"+15005550006"},
		BodyText:    "No callback",
		CallbackURL: "",
	})
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if resp1.Message.CallbackURL != nil {
		t.Errorf("expected CallbackURL to be nil when empty, got %v", *resp1.Message.CallbackURL)
	}

	// Message with whitespace-only callback URL
	resp2, err := svc.SendMessage(ctx, projectID, SendRequest{
		Channel:     ChannelSMS,
		From:        "+15005550000",
		To:          []string{"+15005550006"},
		BodyText:    "Whitespace callback",
		CallbackURL: "   ",
	})
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if resp2.Message.CallbackURL != nil {
		t.Errorf("expected CallbackURL to be nil when whitespace, got %v", *resp2.Message.CallbackURL)
	}

	// Message with valid callback URL
	callback := "https://example.com/webhook"
	resp3, err := svc.SendMessage(ctx, projectID, SendRequest{
		Channel:     ChannelSMS,
		From:        "+15005550000",
		To:          []string{"+15005550006"},
		BodyText:    "With callback",
		CallbackURL: callback,
	})
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if resp3.Message.CallbackURL == nil || *resp3.Message.CallbackURL != callback {
		t.Errorf("expected CallbackURL %q, got %v", callback, resp3.Message.CallbackURL)
	}
}

func TestRecipient_BatchRejectedRecipients(t *testing.T) {
	svc, store, _ := setupTestService()
	ctx := context.Background()
	projectID := "prj_test4"

	// Batch with:
	// 1. Valid number: +1 (500) 555-0006
	// 2. Invalid number: not-a-number
	// 3. Sim rejected number: +15005550001 (invalid number rule)
	// 4. Valid number: +15005550007
	resp, err := svc.SendMessage(ctx, projectID, SendRequest{
		Channel:  ChannelSMS,
		From:     "+15005550000",
		To:       []string{"+1 (500) 555-0006", "not-a-number", "+15005550001", "+15005550007"},
		BodyText: "Batch announcement",
	})
	if err != nil {
		t.Fatalf("SendBatch failed: %v", err)
	}
	if resp.Batch == nil {
		t.Fatal("expected batch response, got nil")
	}

	if resp.Batch.Total != 2 {
		t.Errorf("expected batch total 2, got %d", resp.Batch.Total)
	}
	if len(resp.Batch.Rejected) != 2 {
		t.Fatalf("expected 2 rejected recipients, got %d", len(resp.Batch.Rejected))
	}

	rejectedMap := make(map[string]BatchRejectedRecipient)
	for _, rej := range resp.Batch.Rejected {
		rejectedMap[rej.To] = rej
	}

	if rej, ok := rejectedMap["not-a-number"]; !ok {
		t.Errorf("expected 'not-a-number' in rejected")
	} else if rej.Code != "invalid_number" {
		t.Errorf("expected code 'invalid_number' for 'not-a-number', got %q", rej.Code)
	}

	if rej, ok := rejectedMap["+15005550001"]; !ok {
		t.Errorf("expected '+15005550001' in rejected")
	} else if rej.Code != "invalid_number" {
		t.Errorf("expected code 'invalid_number' for '+15005550001', got %q", rej.Code)
	}

	// Verify the 2 accepted messages exist in store with normalized numbers
	list, _, err := store.ListMessages(ctx, projectID, MessageFilter{})
	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 saved messages in batch, got %d", len(list))
	}
	toRecipients := []string{list[0].To, list[1].To}
	has0006 := toRecipients[0] == "+15005550006" || toRecipients[1] == "+15005550006"
	has0007 := toRecipients[0] == "+15005550007" || toRecipients[1] == "+15005550007"
	if !has0006 || !has0007 {
		t.Errorf("expected normalized recipients +15005550006 and +15005550007, got %v", toRecipients)
	}
}
