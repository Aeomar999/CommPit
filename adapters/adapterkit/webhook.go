package adapterkit

import (
	"net/http"

	"github.com/Aeomar999/CommPit/core"
)

// WebhookRequest is a provider-formatted outbound webhook: the destination
// URL, the encoded body bytes, and the headers to send (including the
// provider's signature header). Notifiers build it; the webhooks worker
// delivers it.
type WebhookRequest struct {
	URL     string
	Body    []byte
	Headers map[string]string
}

// StatusNotifier is an optional adapter capability (spec §7.3): build the
// status-callback request for a message transition. It returns false when
// no webhook applies (e.g. no callback URL was stored).
type StatusNotifier interface {
	StatusWebhook(msg core.Message, event core.StatusEvent, project core.Project) (*WebhookRequest, bool)
}

// InboundNotifier is an optional adapter capability (spec §7.3): build the
// inbound-message webhook for a received message.
type InboundNotifier interface {
	InboundWebhook(msg core.Message, project core.Project) (*WebhookRequest, bool)
}

// ReplyParser is an optional adapter capability (spec §7.3): parse outbound
// replies (e.g. TwiML) from the app server's response to an inbound webhook.
type ReplyParser interface {
	ParseReply(resp *http.Response) ([]core.OutboundReply, error)
}
