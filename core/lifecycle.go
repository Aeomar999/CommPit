package core

import (
	"context"
	"sync"
	"time"
)

type LifecycleRunner struct {
	service   *Service
	clock     Clock
	stepDelay time.Duration
	timers    map[string]*timerEntry
	mu        sync.Mutex
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

type timerEntry struct {
	messageID string
	timer     *time.Timer
}

func NewLifecycleRunner(s *Service) *LifecycleRunner {
	return &LifecycleRunner{
		service:   s,
		clock:     s.clock,
		stepDelay: s.stepDelay,
		timers:    make(map[string]*timerEntry),
		stopCh:    make(chan struct{}),
	}
}

func (lr *LifecycleRunner) Schedule(msg *Message, simResult *SimResult) {
	if lr.stepDelay == 0 {
		lr.advanceImmediately(msg)
		return
	}

	delay := lr.stepDelay
	if simResult != nil && simResult.Delay > 0 {
		delay += simResult.Delay
	}

	lr.mu.Lock()
	defer lr.mu.Unlock()

	timer := time.AfterFunc(delay, func() {
		lr.mu.Lock()
		delete(lr.timers, msg.ID)
		lr.mu.Unlock()
		lr.advance(msg)
	})

	lr.timers[msg.ID] = &timerEntry{
		messageID: msg.ID,
		timer:     timer,
	}
}

func (lr *LifecycleRunner) advanceImmediately(msg *Message) {
	for msg.Status == StatusQueued || msg.Status == StatusSent {
		lr.advance(msg)
		if lr.stepDelay > 0 {
			lr.clock.Sleep(lr.stepDelay)
		}
	}
}

func (lr *LifecycleRunner) advance(msg *Message) {
	var nextStatus MessageStatus
	var errCode *string

	switch msg.Status {
	case StatusQueued:
		nextStatus = StatusSent
	case StatusSent:
		nextStatus = StatusDelivered
	default:
		return
	}

	if nextStatus == StatusDelivered {
		if msg.Segments > 1 {
			nextStatus = StatusDelivered
		}
	}

	msg.Status = nextStatus
	msg.UpdatedAt = lr.clock.Now()

	ctx := context.Background()

	if err := lr.service.store.UpdateMessage(ctx, msg); err != nil {
		return
	}

	event := &StatusEvent{
		ID:        NewWebhookDeliveryID(),
		MessageID: msg.ID,
		Status:    nextStatus,
		ErrorCode: errCode,
		At:        lr.clock.Now(),
	}
	if err := lr.service.store.CreateStatusEvent(ctx, event); err != nil {
		return
	}

	lr.service.bus.Publish(ctx, Event{
		Type:      EventMessageStatus,
		Payload:   event,
		ProjectID: msg.ProjectID,
		Timestamp: lr.clock.Now(),
	})

	if msg.BatchID != nil {
		batch, err := lr.service.store.GetBatch(ctx, msg.ProjectID, *msg.BatchID)
		if err == nil {
			batch.Counts[string(msg.Status)]++
			if prevCount, ok := batch.Counts[string(prevStatus(msg.Status))]; ok && prevCount > 0 {
				batch.Counts[string(prevStatus(msg.Status))]--
			}
			lr.service.store.UpdateBatch(ctx, batch)
			lr.service.bus.Publish(ctx, Event{
				Type:      EventBatchUpdated,
				Payload:   batch,
				ProjectID: msg.ProjectID,
				Timestamp: lr.clock.Now(),
			})
		}
	}

	if nextStatus == StatusSent {
		delay := lr.stepDelay
		lr.mu.Lock()
		timer := time.AfterFunc(delay, func() {
			lr.mu.Lock()
			delete(lr.timers, msg.ID)
			lr.mu.Unlock()
			lr.advance(msg)
		})
		lr.timers[msg.ID] = &timerEntry{
			messageID: msg.ID,
			timer:     timer,
		}
		lr.mu.Unlock()
	}
}

func prevStatus(current MessageStatus) MessageStatus {
	switch current {
	case StatusSent:
		return StatusQueued
	case StatusDelivered, StatusUndelivered, StatusFailed:
		return StatusSent
	default:
		return current
	}
}

func (lr *LifecycleRunner) ResumeQueuedAndSent(ctx context.Context) {
	messages, _, err := lr.service.store.ListMessages(ctx, "", MessageFilter{
		Status: ptrStatus(StatusQueued),
		Limit:  1000,
	})
	if err == nil {
		for _, msg := range messages {
			lr.Schedule(msg, nil)
		}
	}

	messages, _, err = lr.service.store.ListMessages(ctx, "", MessageFilter{
		Status: ptrStatus(StatusSent),
		Limit:  1000,
	})
	if err == nil {
		for _, msg := range messages {
			lr.Schedule(msg, nil)
		}
	}
}

func ptrStatus(s MessageStatus) *MessageStatus {
	return &s
}

func (lr *LifecycleRunner) Start() {
	lr.wg.Add(1)
	go func() {
		defer lr.wg.Done()
		<-lr.stopCh
	}()
}

func (lr *LifecycleRunner) Stop() {
	close(lr.stopCh)
	lr.mu.Lock()
	for _, entry := range lr.timers {
		entry.timer.Stop()
	}
	lr.timers = make(map[string]*timerEntry)
	lr.mu.Unlock()
	lr.wg.Wait()
}
