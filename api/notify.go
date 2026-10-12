package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// signatureHeader is the native webhook signature header. The value carries
// a "sha256=" prefix (GitHub-style) followed by the hex HMAC-SHA256 of the
// raw JSON body, keyed with the project's API key (design spec §7).
const signatureHeader = "X-Mocksms-Signature"

// SignNativePayload signs a native webhook body: "sha256=<hex HMAC-SHA256>".
func SignNativePayload(body []byte, apiKey string) string {
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// statusPayload is the native status-callback body: the event name plus the
// full message as stored (design spec §7).
type statusPayload struct {
	Event   string       `json:"event"`
	Message core.Message `json:"message"`
}

// Notifier builds native-provider status webhooks. It holds the store only
// to resolve the project's API key for signing; the StatusWebhook method
// itself keeps the spec §7.3 signature.
type Notifier struct {
	store core.Store
}

// NewNotifier wires a native status notifier around the store.
func NewNotifier(store core.Store) *Notifier {
	return &Notifier{store: store}
}

// Compile-time checks that Notifier satisfies the dispatcher contracts.
var (
	_ adapterkit.StatusNotifier  = (*Notifier)(nil)
	_ adapterkit.InboundNotifier = (*Notifier)(nil)
)

// signingKey returns the project's oldest native API key — the stable
// choice when several keys exist. Empty when the project has no key, in
// which case webhooks go out unsigned (design spec §7).
func (n *Notifier) signingKey(projectID string) string {
	// Event-driven path: no request context exists. The dispatcher uses
	// context.Background() the same way for its own store reads.
	creds, err := n.store.ListCredentials(context.Background(), projectID)
	if err != nil {
		slog.Warn("native: signing key: list credentials", "project", projectID, "err", err)
		return ""
	}
	oldest := ""
	var oldestAt time.Time
	for _, cred := range creds {
		if cred == nil || cred.Provider != "native" || cred.Key == "" {
			continue
		}
		if oldest == "" || cred.CreatedAt.Before(oldestAt) {
			oldest, oldestAt = cred.Key, cred.CreatedAt
		}
	}
	return oldest
}

// inboundPayload is the native inbound body (design spec §7).
type inboundPayload struct {
	Event   string       `json:"event"`
	Message core.Message `json:"message"`
}

// InboundWebhook builds the signed JSON inbound notification for a native
// inbound message (design spec §7). It returns false for other providers
// or directions, and when native.inbound_url is not configured.
func (n *Notifier) InboundWebhook(msg core.Message, project core.Project) (*adapterkit.WebhookRequest, bool) {
	if msg.Provider != "native" {
		return nil, false
	}
	if msg.Direction != core.DirectionInbound {
		return nil, false
	}
	var webhookURL string
	if project.Settings != nil {
		webhookURL, _ = project.Settings["native.inbound_url"].(string)
	}
	if webhookURL == "" {
		return nil, false
	}

	body, err := json.Marshal(inboundPayload{Event: "message.inbound", Message: msg})
	if err != nil {
		slog.Warn("native: encode inbound webhook", "message", msg.ID, "err", err)
		return nil, false
	}

	headers := map[string]string{"Content-Type": "application/json"}
	if key := n.signingKey(msg.ProjectID); key != "" {
		headers[signatureHeader] = SignNativePayload(body, key)
	} else {
		slog.Warn("native: project has no API key; sending unsigned inbound webhook", "project", project.ID)
	}
	return &adapterkit.WebhookRequest{URL: webhookURL, Body: body, Headers: headers}, true
}

// StatusWebhook builds the signed JSON status callback for a native
// message transition. It returns false when no webhook applies.
func (n *Notifier) StatusWebhook(msg core.Message, event core.StatusEvent, project core.Project) (*adapterkit.WebhookRequest, bool) {
	if msg.Provider != "native" {
		return nil, false
	}
	if msg.CallbackURL == nil || *msg.CallbackURL == "" {
		return nil, false
	}

	body, err := json.Marshal(statusPayload{Event: string(core.EventMessageStatus), Message: msg})
	if err != nil {
		slog.Warn("native: encode status webhook", "message", msg.ID, "err", err)
		return nil, false
	}

	headers := map[string]string{"Content-Type": "application/json"}
	if key := n.signingKey(msg.ProjectID); key != "" {
		headers[signatureHeader] = SignNativePayload(body, key)
	} else {
		slog.Warn("native: project has no API key; sending unsigned status webhook", "project", project.ID)
	}
	return &adapterkit.WebhookRequest{URL: *msg.CallbackURL, Body: body, Headers: headers}, true
}
