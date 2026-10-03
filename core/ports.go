package core

import (
	"context"
	"io"
	"time"
)

type Store interface {
	CreateProject(ctx context.Context, project *Project) error
	GetProject(ctx context.Context, projectID string) (*Project, error)
	UpdateProject(ctx context.Context, project *Project) error
	ListProjects(ctx context.Context, limit int, cursor string) ([]*Project, string, error)

	CreateCredential(ctx context.Context, cred *Credential) error
	GetCredential(ctx context.Context, provider, key string) (*Credential, error)
	ListCredentials(ctx context.Context, projectID string) ([]*Credential, error)
	DeleteCredential(ctx context.Context, id string) error

	CreateMessage(ctx context.Context, msg *Message) error
	GetMessage(ctx context.Context, projectID, messageID string) (*Message, error)
	UpdateMessage(ctx context.Context, msg *Message) error
	ListMessages(ctx context.Context, projectID string, filter MessageFilter) ([]*Message, string, error)
	DeleteMessages(ctx context.Context, projectID string) error

	CreateStatusEvent(ctx context.Context, event *StatusEvent) error
	GetStatusEvents(ctx context.Context, messageID string) ([]*StatusEvent, error)

	CreateBatch(ctx context.Context, batch *Batch) error
	GetBatch(ctx context.Context, projectID, batchID string) (*Batch, error)
	UpdateBatch(ctx context.Context, batch *Batch) error
	ListBatches(ctx context.Context, projectID string, limit int, cursor string) ([]*Batch, string, error)

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

type Simulator interface {
	Evaluate(ctx context.Context, projectID string, req SendRequest) (*Error, *SimResult)
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

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
	Sleep(d time.Duration)
}

type RealClock struct{}

func (RealClock) Now() time.Time                         { return time.Now() }
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
func (RealClock) Sleep(d time.Duration)                  { time.Sleep(d) }

type FakeClock struct {
	now time.Time
}

func NewFakeClock() *FakeClock {
	return &FakeClock{now: time.Now()}
}

func (fc *FakeClock) Now() time.Time { return fc.now }
func (fc *FakeClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	go func() {
		<-time.After(d)
		ch <- fc.now.Add(d)
	}()
	return ch
}
func (fc *FakeClock) Sleep(d time.Duration)   { fc.now = fc.now.Add(d) }
func (fc *FakeClock) Advance(d time.Duration) { fc.now = fc.now.Add(d) }
func (fc *FakeClock) Set(t time.Time)         { fc.now = t }

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
