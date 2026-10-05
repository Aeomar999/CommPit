package core

import (
	"context"
	"io"
	"sync"
	"time"
)

type Store interface {
	CreateProject(ctx context.Context, project *Project) error
	GetProject(ctx context.Context, projectID string) (*Project, error)
	UpdateProject(ctx context.Context, project *Project) error
	ListProjects(ctx context.Context, limit int, cursor string) ([]*Project, string, error)
	DeleteProject(ctx context.Context, id string) error

	CreateCredential(ctx context.Context, cred *Credential) error
	GetCredential(ctx context.Context, provider, key string) (*Credential, error)
	ListCredentials(ctx context.Context, projectID string) ([]*Credential, error)
	DeleteCredential(ctx context.Context, id string) error

	CreateMessage(ctx context.Context, msg *Message) error
	GetMessage(ctx context.Context, projectID, messageID string) (*Message, error)
	UpdateMessage(ctx context.Context, msg *Message) error
	ListMessages(ctx context.Context, projectID string, filter MessageFilter) ([]*Message, string, error)
	DeleteMessages(ctx context.Context, projectID string) error
	DeleteMessage(ctx context.Context, projectID, messageID string) error
	ListInFlightMessages(ctx context.Context) ([]*Message, error)

	CreateStatusEvent(ctx context.Context, event *StatusEvent) error
	GetStatusEvents(ctx context.Context, messageID string) ([]*StatusEvent, error)

	CreateBatch(ctx context.Context, batch *Batch) error
	GetBatch(ctx context.Context, projectID, batchID string) (*Batch, error)
	UpdateBatch(ctx context.Context, batch *Batch) error
	ListBatches(ctx context.Context, projectID string, limit int, cursor string) ([]*Batch, string, error)
	RecomputeBatchCounts(ctx context.Context, projectID, batchID string) (*Batch, error)

	CreateVerification(ctx context.Context, v *Verification) error
	GetVerification(ctx context.Context, projectID, verificationID string) (*Verification, error)
	GetVerificationByProviderRef(ctx context.Context, projectID, providerRef string) (*Verification, error)
	UpdateVerification(ctx context.Context, v *Verification) error
	ListVerifications(ctx context.Context, projectID string, limit int, cursor string) ([]*Verification, string, error)

	CreateUnsubscribe(ctx context.Context, u *Unsubscribe) error
	DeleteUnsubscribe(ctx context.Context, projectID, number string) error
	IsUnsubscribed(ctx context.Context, projectID, number string) (bool, error)

	CreateAttachment(ctx context.Context, att *Attachment) error
	GetAttachment(ctx context.Context, attachmentID string) (*Attachment, error)
	ListAttachments(ctx context.Context, messageID string) ([]*Attachment, error)

	CreateWebhookDelivery(ctx context.Context, whd *WebhookDelivery) error
	GetWebhookDelivery(ctx context.Context, id string) (*WebhookDelivery, error)
	UpdateWebhookDelivery(ctx context.Context, whd *WebhookDelivery) error
	ListPendingWebhooks(ctx context.Context, limit int) ([]*WebhookDelivery, error)

	CreateRequestLog(ctx context.Context, log *RequestLog) error
	GetRequestLog(ctx context.Context, id string) (*RequestLog, error)
	ListRequestLogs(ctx context.Context, projectID string, limit int, cursor string) ([]*RequestLog, string, error)

	Transaction(ctx context.Context, fn func(Store) error) error
}

type BlobStore interface {
	Put(ctx context.Context, id string, r io.Reader) (int64, error)
	Get(ctx context.Context, id string) (io.ReadCloser, error)
	Delete(ctx context.Context, id string) error
}

type Bus interface {
	Publish(ctx context.Context, event Event)
	Subscribe(eventType string, handler EventHandler) Subscription
}

type EventType string

const (
	EventMessageCreated      EventType = "message.created"
	EventMessageStatus       EventType = "message.status"
	EventVerificationUpdated EventType = "verification.updated"
	EventBatchUpdated        EventType = "batch.updated"
	EventRequestLogged       EventType = "request.logged"
	EventWebhookDelivered    EventType = "webhook.delivered"
)

type Event struct {
	Type      EventType
	Payload   interface{}
	ProjectID string
	Timestamp time.Time
}

type EventHandler func(Event)

type Subscription interface {
	Unsubscribe()
}

type SimResult struct {
	Delay       time.Duration
	AsyncFail   *AsyncFail
	Hang        time.Duration
	RateLimited bool
}

type AsyncFail struct {
	ErrorCode    string
	ErrorMessage string
}

type SimMatch struct {
	To       string
	From     string
	Provider string
	Project  string
	Channel  Channel
}

type SimEffect struct {
	Reject    *Error
	FailAsync *AsyncFail
	Delay     time.Duration
	Hang      time.Duration
	RateLimit int
}

type SimRule struct {
	Match  SimMatch
	Effect SimEffect
}

type Simulator interface {
	Evaluate(ctx context.Context, projectID string, req SendRequest) (*Error, *SimResult)
	SetRules(rules []SimRule)
	SetLatency(latency time.Duration)
	SetFailureRate(rate float64)
}

// NoopSimulator is a no-op implementation of Simulator.
type NoopSimulator struct{}

// Evaluate implements Simulator.
func (NoopSimulator) Evaluate(ctx context.Context, projectID string, req SendRequest) (*Error, *SimResult) {
	return nil, &SimResult{}
}

// SetRules implements Simulator.
func (NoopSimulator) SetRules(_ []SimRule) {}

// SetLatency implements Simulator.
func (NoopSimulator) SetLatency(_ time.Duration) {}

// SetFailureRate implements Simulator.
func (NoopSimulator) SetFailureRate(_ float64) {}

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
	Sleep(d time.Duration)
}

type RealClock struct{}

func (RealClock) Now() time.Time                         { return time.Now() }
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
func (RealClock) Sleep(d time.Duration)                  { time.Sleep(d) }

type fakeTimer struct {
	target time.Time
	ch     chan time.Time
}

type FakeClock struct {
	mu     sync.Mutex
	cond   *sync.Cond
	now    time.Time
	timers []*fakeTimer
}

func NewFakeClock() *FakeClock {
	fc := &FakeClock{now: time.Now()}
	fc.cond = sync.NewCond(&fc.mu)
	return fc
}

func (fc *FakeClock) Now() time.Time {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.now
}

func (fc *FakeClock) After(d time.Duration) <-chan time.Time {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	ch := make(chan time.Time, 1)
	if d <= 0 {
		ch <- fc.now
		return ch
	}
	target := fc.now.Add(d)
	fc.timers = append(fc.timers, &fakeTimer{target: target, ch: ch})
	if fc.cond != nil {
		fc.cond.Broadcast()
	}
	return ch
}

func (fc *FakeClock) Sleep(d time.Duration) {
	fc.Advance(d)
}

func (fc *FakeClock) Advance(d time.Duration) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.now = fc.now.Add(d)
	var remaining []*fakeTimer
	for _, t := range fc.timers {
		if !t.target.After(fc.now) {
			t.ch <- fc.now
		} else {
			remaining = append(remaining, t)
		}
	}
	fc.timers = remaining
}

func (fc *FakeClock) Set(t time.Time) {
	fc.mu.Lock()
	diff := t.Sub(fc.now)
	fc.mu.Unlock()
	if diff > 0 {
		fc.Advance(diff)
	} else {
		fc.mu.Lock()
		fc.now = t
		fc.mu.Unlock()
	}
}

func (fc *FakeClock) Waiters() int {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return len(fc.timers)
}

func (fc *FakeClock) BlockUntilWaiters(count int) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	if fc.cond == nil {
		fc.cond = sync.NewCond(&fc.mu)
	}
	for len(fc.timers) < count {
		fc.cond.Wait()
	}
}

type ProjectResolver interface {
	Resolve(ctx context.Context, provider, key string) (string, error)
}

type MessageFilter struct {
	Channel   *Channel
	To        *string
	From      *string
	Status    *MessageStatus
	BatchID   *string
	Direction *Direction
	Since     *time.Time
	Limit     int
	Cursor    string
}

type OutboundReply struct {
	From        string
	To          string
	Body        string
	Provider    string
	ProviderRef string
}
