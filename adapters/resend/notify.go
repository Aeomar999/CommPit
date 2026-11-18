package resend

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

const (
	webhookURLKey  = "resend.webhook_url"
	observedAPIKey = "resend.observed_api_key"
)

var _ adapterkit.StatusNotifier = (*Adapter)(nil)

func (a *Adapter) recordAPIKey(ctx context.Context, projectID, apiKey string) {
	if apiKey == "" {
		return
	}
	project, err := a.service.Store().GetProject(ctx, projectID)
	if err != nil {
		slog.Warn("resend: record api key: load project", "project", projectID, "err", err)
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
		slog.Warn("resend: record api key: store project", "project", projectID, "err", err)
	}
}

type resendEvent struct {
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
	Data      struct {
		EmailID string `json:"email_id"`
		To      string `json:"to"`
	} `json:"data"`
}

func resendEventType(status core.MessageStatus) string {
	switch status {
	case core.StatusDelivered:
		return "email.delivered"
	case core.StatusFailed, core.StatusUndelivered:
		return "email.bounced"
	default:
		return "email.sent"
	}
}

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

	ev := resendEvent{Type: resendEventType(event.Status), CreatedAt: event.At.UTC().Format(time.RFC3339)}
	ev.Data.EmailID = msg.ProviderRef
	ev.Data.To = msg.To
	body, err := json.Marshal(ev)
	if err != nil {
		slog.Warn("resend: encode event", "message", msg.ID, "err", err)
		return nil, false
	}

	headers := map[string]string{"Content-Type": "application/json"}
	secret, _ := project.Settings[observedAPIKey].(string)
	if secret != "" {
		// Svix-style signing (sandbox convention).
		svixID := "msg_" + newMessageID()
		ts := strconv.FormatInt(event.At.Unix(), 10)
		signed := svixID + "." + ts + "." + string(body)
		key := secret
		if len(key) > 6 && key[:6] == "whsec_" {
			key = key[6:]
		}
		// whsec secrets are base64-encoded; try decoding, fall back to raw.
		decoded, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			decoded = []byte(key)
		}
		mac := hmac.New(sha256.New, decoded)
		mac.Write([]byte(signed))
		sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		headers["svix-id"] = svixID
		headers["svix-timestamp"] = ts
		headers["svix-signature"] = "v1," + sig
	} else {
		slog.Warn("resend: no api key seen; sending unsigned event", "project", project.ID)
	}
	return &adapterkit.WebhookRequest{URL: webhookURL, Body: body, Headers: headers}, true
}

// SignPayload computes the sandbox's Resend webhook signature.
func SignPayload(payload []byte, secret string) string {
	key := secret
	if len(key) > 6 && key[:6] == "whsec_" {
		key = key[6:]
	}
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		decoded = []byte(key)
	}
	mac := hmac.New(sha256.New, decoded)
	mac.Write(payload)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}


