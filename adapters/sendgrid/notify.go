package sendgrid

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// webhookURLKey is the account-level event-webhook destination (SendGrid
// configures one webhook URL per account in its dashboard, not per
// request). observedAPIKey stores the API key seen on live traffic for
// signing when no explicit secret is configured.
const (
	webhookURLKey  = "sendgrid.webhook_url"
	observedAPIKey = "sendgrid.observed_api_key"
)

// Compile-time check that Adapter satisfies the dispatcher contract.
var _ adapterkit.StatusNotifier = (*Adapter)(nil)

// recordAPIKey remembers the API key seen on a send for webhook signing.
// Writes only when the key changed.
func (a *Adapter) recordAPIKey(ctx context.Context, projectID, apiKey string) {
	if apiKey == "" {
		return
	}
	project, err := a.service.Store().GetProject(ctx, projectID)
	if err != nil {
		slog.Warn("sendgrid: record api key: load project", "project", projectID, "err", err)
		return
	}
	if project.Settings == nil {
		project.Settings = map[string]interface{}{}
	}
	if current, _ := project.Settings[observedAPIKey].(string); current == apiKey {
		return
	}
	project.Settings[observedAPIKey] = apiKey
	if err := a.service.Store().UpdateProject(ctx, project); err != nil {
		slog.Warn("sendgrid: record api key: store project", "project", projectID, "err", err)
	}
}

// sendGridEvent is one entry in SendGrid's event-webhook JSON array.
type sendGridEvent struct {
	Email      string `json:"email"`
	Timestamp  int64  `json:"timestamp"`
	Event      string `json:"event"`
	SGMessageID string `json:"sg_message_id"`
}

// sgEvent maps canonical statuses onto SendGrid's event vocabulary.
func sgEvent(status core.MessageStatus) string {
	switch status {
	case core.StatusDelivered:
		return "delivered"
	case core.StatusFailed, core.StatusUndelivered:
		return "bounce"
	default:
		return "processed"
	}
}

// StatusWebhook builds the JSON event array for a message transition
// (spec §6.4, §7.2). It returns false when the account-level webhook URL
// is not configured.
func (a *Adapter) StatusWebhook(msg core.Message, event core.StatusEvent, project core.Project) (*adapterkit.WebhookRequest, bool) {
	if msg.Provider != a.Name() {
		return nil, false
	}
	var webhookURL string
	if project.Settings != nil {
		webhookURL, _ = project.Settings[webhookURLKey].(string)
	}
	if webhookURL == "" {
		return nil, false
	}

	ev := sendGridEvent{
		Email:       msg.To,
		Timestamp:   event.At.Unix(),
		Event:       sgEvent(event.Status),
		SGMessageID: msg.ProviderRef,
	}
	body, err := json.Marshal([]sendGridEvent{ev})
	if err != nil {
		slog.Warn("sendgrid: encode event", "message", msg.ID, "err", err)
		return nil, false
	}

	headers := map[string]string{"Content-Type": "application/json"}
	secret, _ := project.Settings[observedAPIKey].(string)
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		headers["X-SendGrid-Signature"] = hex.EncodeToString(mac.Sum(nil))
	} else {
		slog.Warn("sendgrid: no api key seen; sending unsigned event", "project", project.ID)
	}
	return &adapterkit.WebhookRequest{URL: webhookURL, Body: body, Headers: headers}, true
}

// SignPayload computes the sandbox's SendGrid webhook signature: hex-encoded
// HMAC-SHA256 over the exact JSON bytes, keyed by the observed API key.
// SendGrid's real webhook uses ECDSA; this is a sandbox convention (see
// docs/fidelity.md).
func SignPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
