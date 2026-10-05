package core

import (
	"time"

	"github.com/oklog/ulid/v2"
)

const (
	ProjectIDPrefix         = "prj_"
	MessageIDPrefix         = "msg_"
	BatchIDPrefix           = "bat_"
	VerificationIDPrefix    = "vrf_"
	WebhookDeliveryIDPrefix = "whd_"
	RequestLogIDPrefix      = "req_"
	AttachmentIDPrefix      = "att_"
	BlobIDPrefix            = "blob_"
	StatusEventIDPrefix     = "sev_"
)

func NewProjectID() string {
	return ProjectIDPrefix + ulid.Make().String()
}

func NewStatusEventID() string {
	return StatusEventIDPrefix + ulid.Make().String()
}

func NewMessageID() string {
	return MessageIDPrefix + ulid.Make().String()
}

func NewBatchID() string {
	return BatchIDPrefix + ulid.Make().String()
}

func NewVerificationID() string {
	return VerificationIDPrefix + ulid.Make().String()
}

func NewWebhookDeliveryID() string {
	return WebhookDeliveryIDPrefix + ulid.Make().String()
}

func NewRequestLogID() string {
	return RequestLogIDPrefix + ulid.Make().String()
}

func NewAttachmentID() string {
	return AttachmentIDPrefix + ulid.Make().String()
}

func NewBlobID() string {
	return BlobIDPrefix + ulid.Make().String()
}

type Channel string

const (
	ChannelSMS   Channel = "sms"
	ChannelEmail Channel = "email"
)

type Direction string

const (
	DirectionOutbound Direction = "outbound"
	DirectionInbound  Direction = "inbound"
)

type MessageStatus string

const (
	StatusQueued      MessageStatus = "queued"
	StatusSent        MessageStatus = "sent"
	StatusDelivered   MessageStatus = "delivered"
	StatusUndelivered MessageStatus = "undelivered"
	StatusFailed      MessageStatus = "failed"
	StatusReceived    MessageStatus = "received"
)

type VerificationStatus string

const (
	VerificationPending     VerificationStatus = "pending"
	VerificationApproved    VerificationStatus = "approved"
	VerificationCanceled    VerificationStatus = "canceled"
	VerificationExpired     VerificationStatus = "expired"
	VerificationMaxAttempts VerificationStatus = "max_attempts"
)

type WebhookDeliveryStatus string

const (
	WebhookPending   WebhookDeliveryStatus = "pending"
	WebhookSucceeded WebhookDeliveryStatus = "succeeded"
	WebhookFailed    WebhookDeliveryStatus = "failed"
)

type Project struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Settings  map[string]interface{} `json:"settings"`
	CreatedAt time.Time              `json:"created_at"`
}

type Credential struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	Key       string    `json:"key"`
	ProjectID string    `json:"project_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID             string        `json:"id"`
	ProjectID      string        `json:"project_id"`
	BatchID        *string       `json:"batch_id,omitempty"`
	Channel        Channel       `json:"channel"`
	Direction      Direction     `json:"direction"`
	Provider       string        `json:"provider"`
	ProviderRef    string        `json:"provider_ref"`
	From           string        `json:"from"`
	To             string        `json:"to"`
	CC             []string      `json:"cc,omitempty"`
	BCC            []string      `json:"bcc,omitempty"`
	Subject        string        `json:"subject,omitempty"`
	BodyText       string        `json:"body_text,omitempty"`
	BodyHTML       string        `json:"body_html,omitempty"`
	RawBlobID      *string       `json:"raw_blob_id,omitempty"`
	Encoding       string        `json:"encoding,omitempty"`
	Segments       int           `json:"segments"`
	Status         MessageStatus `json:"status"`
	ErrorCode      *string       `json:"error_code,omitempty"`
	ErrorMessage   *string       `json:"error_message,omitempty"`
	CallbackURL    *string       `json:"callback_url,omitempty"`
	ExtractedCodes []string      `json:"extracted_codes,omitempty"`
	ExtractedLinks []string      `json:"extracted_links,omitempty"`
	PrimaryLink    *string       `json:"primary_link,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type StatusEvent struct {
	ID        string        `json:"id"`
	MessageID string        `json:"message_id"`
	Status    MessageStatus `json:"status"`
	ErrorCode *string       `json:"error_code,omitempty"`
	At        time.Time     `json:"at"`
}

type Batch struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"project_id"`
	Provider  string         `json:"provider"`
	Channel   Channel        `json:"channel"`
	Total     int            `json:"total"`
	Counts    map[string]int `json:"counts"`
	CreatedAt time.Time      `json:"created_at"`
}

type Verification struct {
	ID          string             `json:"id"`
	ProjectID   string             `json:"project_id"`
	Provider    string             `json:"provider"`
	ProviderRef string             `json:"provider_ref"`
	ServiceRef  *string            `json:"service_ref,omitempty"`
	To          string             `json:"to"`
	Channel     Channel            `json:"channel"`
	Code        string             `json:"code"`
	Status      VerificationStatus `json:"status"`
	Attempts    int                `json:"attempts"`
	MaxAttempts int                `json:"max_attempts"`
	ExpiresAt   time.Time          `json:"expires_at"`
	MessageID   string             `json:"message_id"`
	CreatedAt   time.Time          `json:"created_at"`
}

type Unsubscribe struct {
	ProjectID string    `json:"project_id"`
	Number    string    `json:"number"`
	At        time.Time `json:"at"`
}

type Attachment struct {
	ID          string `json:"id"`
	MessageID   string `json:"message_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	BlobID      string `json:"blob_id"`
	InlineCID   string `json:"inline_cid,omitempty"`
}

type WebhookDelivery struct {
	ID             string                 `json:"id"`
	ProjectID      string                 `json:"project_id"`
	MessageID      *string                `json:"message_id,omitempty"`
	VerificationID *string                `json:"verification_id,omitempty"`
	Kind           string                 `json:"kind"`
	URL            string                 `json:"url"`
	Payload        map[string]interface{} `json:"payload"`
	Headers        map[string]string      `json:"headers"`
	Attempt        int                    `json:"attempt"`
	Status         WebhookDeliveryStatus  `json:"status"`
	ResponseStatus *int                   `json:"response_status,omitempty"`
	ResponseBody   *string                `json:"response_body,omitempty"`
	NextRetryAt    *time.Time             `json:"next_retry_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

type RequestLog struct {
	ID             string            `json:"id"`
	ProjectID      string            `json:"project_id"`
	Adapter        string            `json:"adapter"`
	Method         string            `json:"method"`
	Path           string            `json:"path"`
	RequestHeaders map[string]string `json:"request_headers"`
	RequestBody    []byte            `json:"request_body"`
	ResponseStatus int               `json:"response_status"`
	ResponseBody   []byte            `json:"response_body"`
	DurationMS     int64             `json:"duration_ms"`
	CreatedAt      time.Time         `json:"created_at"`
}

type SendRequest struct {
	Channel     Channel
	From        string
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	BodyText    string
	BodyHTML    string
	Attachments []AttachmentInput
	CallbackURL string
	Provider    string
	ProviderRef string
	RawBlobID   *string
}

type AttachmentInput struct {
	Filename      string
	ContentType   string
	ContentBase64 string
	InlineCID     string
}

type SendResponse struct {
	Message *Message
	Batch   *Batch
}

type VerificationRequest struct {
	To          string
	Channel     Channel
	CodeLength  int
	TTLSeconds  int
	MaxAttempts int
	Provider    string
	ProviderRef string
	ServiceRef  *string
}

type VerificationResponse struct {
	Verification *Verification
	Message      *Message
}

type CheckVerificationRequest struct {
	Code string
}

type CheckVerificationResponse struct {
	Valid  bool
	Status VerificationStatus
}

type InboundRequest struct {
	From string
	To   string
	Body string
}
