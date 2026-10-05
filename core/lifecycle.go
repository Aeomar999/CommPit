package core

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type LifecycleRunner struct {
	service          *Service
	clock            Clock
	stepDelay        time.Duration
	timers           map[string]*timerEntry
	mu               sync.Mutex
	stopCh           chan struct{}
	stopOnce         sync.Once
	wg               sync.WaitGroup
	batchMu          sync.Mutex
	lastBatchPublish map[string]time.Time
}

type timerEntry struct {
	messageID string
	cancel    context.CancelFunc
}

func NewLifecycleRunner(s *Service) *LifecycleRunner {
	return &LifecycleRunner{
		service:          s,
		clock:            s.clock,
		stepDelay:        s.stepDelay,
		timers:           make(map[string]*timerEntry),
		stopCh:           make(chan struct{}),
		lastBatchPublish: make(map[string]time.Time),
	}
}

func (lr *LifecycleRunner) Schedule(projectID, messageID string, simResult *SimResult) {
	lr.mu.Lock()
	select {
	case <-lr.stopCh:
		lr.mu.Unlock()
		return
	default:
	}
	lr.mu.Unlock()

	delay := lr.stepDelay
	if simResult != nil && simResult.Delay > 0 {
		delay += simResult.Delay
	}

	if delay == 0 {
		ctx := context.Background()
		for {
			status, err := lr.advance(ctx, projectID, messageID, simResult)
			if err != nil || isTerminalStatus(status) {
				break
			}
		}
		return
	}

	lr.armTimer(projectID, messageID, delay, simResult)
}

func (lr *LifecycleRunner) armTimer(projectID, messageID string, delay time.Duration, simResult *SimResult) {
	lr.mu.Lock()
	select {
	case <-lr.stopCh:
		lr.mu.Unlock()
		return
	default:
	}

	if existing, ok := lr.timers[messageID]; ok {
		existing.cancel()
		delete(lr.timers, messageID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	lr.timers[messageID] = &timerEntry{
		messageID: messageID,
		cancel:    cancel,
	}
	lr.mu.Unlock()

	timerCh := lr.clock.After(delay)

	lr.wg.Add(1)
	go func() {
		defer lr.wg.Done()
		select {
		case <-timerCh:
			lr.mu.Lock()
			delete(lr.timers, messageID)
			lr.mu.Unlock()

			status, err := lr.advance(ctx, projectID, messageID, simResult)
			if err == nil && !isTerminalStatus(status) {
				lr.armTimer(projectID, messageID, delay, simResult)
			}
		case <-ctx.Done():
			return
		case <-lr.stopCh:
			return
		}
	}()
}

func (lr *LifecycleRunner) advance(ctx context.Context, projectID, messageID string, simResult *SimResult) (MessageStatus, error) {
	lr.mu.Lock()
	select {
	case <-lr.stopCh:
		lr.mu.Unlock()
		return "", context.Canceled
	default:
	}
	lr.mu.Unlock()

	msg, err := lr.service.store.GetMessage(ctx, projectID, messageID)
	if err != nil {
		slog.Error("lifecycle runner: failed to get message", "project_id", projectID, "message_id", messageID, "error", err)
		return "", err
	}

	var nextStatus MessageStatus
	var errCode *string
	var errMsg *string

	switch msg.Status {
	case StatusQueued:
		nextStatus = StatusSent
	case StatusSent:
		if simResult != nil && simResult.AsyncFail != nil {
			nextStatus = StatusUndelivered
			errCode = &simResult.AsyncFail.ErrorCode
			errMsg = &simResult.AsyncFail.ErrorMessage
		} else {
			nextStatus = StatusDelivered
		}
	default:
		return msg.Status, nil
	}

	msg.Status = nextStatus
	msg.UpdatedAt = lr.clock.Now()
	if errCode != nil {
		msg.ErrorCode = errCode
	}
	if errMsg != nil {
		msg.ErrorMessage = errMsg
	}

	if err := lr.service.store.UpdateMessage(ctx, msg); err != nil {
		slog.Error("lifecycle runner: failed to update message", "project_id", projectID, "message_id", messageID, "error", err)
		return "", err
	}

	event := &StatusEvent{
		ID:        NewStatusEventID(),
		MessageID: msg.ID,
		Status:    nextStatus,
		ErrorCode: errCode,
		At:        lr.clock.Now(),
	}
	if err := lr.service.store.CreateStatusEvent(ctx, event); err != nil {
		slog.Error("lifecycle runner: failed to create status event", "project_id", projectID, "message_id", messageID, "error", err)
		return "", err
	}

	lr.service.bus.Publish(ctx, Event{
		Type:      EventMessageStatus,
		Payload:   event,
		ProjectID: msg.ProjectID,
		Timestamp: lr.clock.Now(),
	})

	if msg.BatchID != nil {
		lr.updateAndPublishBatch(ctx, msg.ProjectID, *msg.BatchID)
	}

	return nextStatus, nil
}

func (lr *LifecycleRunner) updateAndPublishBatch(ctx context.Context, projectID, batchID string) {
	batch, err := lr.service.store.RecomputeBatchCounts(ctx, projectID, batchID)
	if err != nil {
		slog.Error("lifecycle runner: failed to recompute batch counts", "project_id", projectID, "batch_id", batchID, "error", err)
		return
	}

	lr.batchMu.Lock()
	lastPublish := lr.lastBatchPublish[batchID]
	now := lr.clock.Now()

	completed := isBatchComplete(batch)
	shouldPublish := completed || lastPublish.IsZero() || now.Sub(lastPublish) >= 250*time.Millisecond

	if shouldPublish {
		lr.lastBatchPublish[batchID] = now
		lr.batchMu.Unlock()
		lr.service.bus.Publish(ctx, Event{
			Type:      EventBatchUpdated,
			Payload:   batch,
			ProjectID: projectID,
			Timestamp: now,
		})
	} else {
		lr.batchMu.Unlock()
	}
}

func isBatchComplete(batch *Batch) bool {
	if batch.Total == 0 {
		return true
	}
	delivered := batch.Counts[string(StatusDelivered)]
	undelivered := batch.Counts[string(StatusUndelivered)]
	failed := batch.Counts[string(StatusFailed)]
	return delivered+undelivered+failed >= batch.Total
}

func isTerminalStatus(status MessageStatus) bool {
	switch status {
	case StatusDelivered, StatusUndelivered, StatusFailed, StatusReceived:
		return true
	default:
		return false
	}
}

func (lr *LifecycleRunner) ResumeQueuedAndSent(ctx context.Context) {
	messages, err := lr.service.store.ListInFlightMessages(ctx)
	if err != nil {
		slog.Error("lifecycle runner: failed to list in-flight messages for resume", "error", err)
		return
	}
	for _, msg := range messages {
		lr.Schedule(msg.ProjectID, msg.ID, nil)
	}
}

func (lr *LifecycleRunner) Start() {
	// Lifecycle runner starts automatically when messages are scheduled
}

func (lr *LifecycleRunner) Stop() {
	lr.stopOnce.Do(func() {
		close(lr.stopCh)
		lr.mu.Lock()
		for _, entry := range lr.timers {
			entry.cancel()
		}
		lr.timers = make(map[string]*timerEntry)
		lr.mu.Unlock()
	})
	lr.wg.Wait()
}
