package termii

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"log/slog"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// webhookURLKey is the account-level DLR destination (spec §6.4 — Termii
// configures one webhook URL per account in its dashboard, not per
// request). observedAPIKey stores the api_key seen on live traffic for
// signing when no explicit secret is configured.
const (
	webhookURLKey   = "termii.webhook_url"
	observedAPIKey  = "termii.observed_api_key"
	signatureHeader = "X-Termii-Signature"
	outboundType    = "outbound"
	genericChannel  = "generic"
	sentAtLayout    = "2006-01-02T15:04:05.000000Z"
)

// deliveryReport mirrors Termii's outbound-message (DLR) webhook JSON
// (developer.termii.com/events-and-reports). The sandbox has no billing,
// so cost is always null, matching Termii's null balance in send
// responses.
type deliveryReport struct {
	Type      string  `json:"type"`
	ID        string  `json:"id"`
	MessageID string  `json:"message_id"`
	Receiver  string  `json:"receiver"`
	Sender    string  `json:"sender"`
	Message   string  `json:"message"`
	SentAt    string  `json:"sent_at"`
	Cost      *string `json:"cost"`
	Status    string  `json:"status"`
	Channel   string  `json:"channel"`
}

// SignPayload computes Termii's webhook signature: hex-encoded HMAC-SHA512
// over the exact JSON bytes, keyed by the account secret. The hex encoding
// is a sandbox assumption — Termii documents "HMAC SHA512" without naming
// an encoding (see docs/fidelity.md).
func SignPayload(payload []byte, secret string) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// recordAPIKey remembers the api_key seen on a send for DLR signing.
// Writes only when the key changed.
func (a *Adapter) recordAPIKey(ctx context.Context, projectID, apiKey string) {
	if apiKey == "" {
		return
	}
	project, err := a.service.Store().GetProject(ctx, projectID)
	if err != nil {
		slog.Warn("termii: record api key: load project", "project", projectID, "err", err)
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
		slog.Warn("termii: record api key: store project", "project", projectID, "err", err)
	}
}

// dlrStatus maps canonical statuses onto Termii's report vocabulary.
// Termii's first state is "Message Sent", so queued reports use it too.
func dlrStatus(status core.MessageStatus) string {
	switch status {
	case core.StatusDelivered:
		return "Delivered"
	case core.StatusFailed, core.StatusUndelivered:
		return "Message Failed"
	default:
		return "Message Sent"
	}
}

// StatusWebhook builds the JSON delivery report for a message transition
// (spec §6.4, §7.2). It returns false when the account-level webhook URL
// is not configured — Termii has no per-request callback.
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

	body, err := json.Marshal(deliveryReport{
		Type:      outboundType,
		ID:        msg.ProviderRef,
		MessageID: msg.ProviderRef,
		Receiver:  msg.To,
		Sender:    msg.From,
		Message:   msg.BodyText,
		SentAt:    msg.CreatedAt.UTC().Format(sentAtLayout),
		Cost:      nil,
		Status:    dlrStatus(event.Status),
		Channel:   genericChannel,
	})
	if err != nil {
		slog.Warn("termii: encode delivery report", "message", msg.ID, "err", err)
		return nil, false
	}

	headers := map[string]string{"Content-Type": "application/json"}
	secret, _ := project.Settings[observedAPIKey].(string)
	if secret != "" {
		headers[signatureHeader] = SignPayload(body, secret)
	} else {
		slog.Warn("termii: no api key seen; sending unsigned delivery report", "project", project.ID)
	}
	return &adapterkit.WebhookRequest{URL: webhookURL, Body: body, Headers: headers}, true
}

// Compile-time check that Adapter satisfies the dispatcher contract.
var _ adapterkit.StatusNotifier = (*Adapter)(nil)
