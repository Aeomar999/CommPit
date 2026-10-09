package twilio

import (
	"context"
	"log/slog"
	"net/url"
	"strings"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// Project settings keys for Twilio credential memory. The explicit
// auth_token wins; observed values are fallbacks recorded from live
// traffic (spec §7.4).
const (
	settingAccountSID = "twilio.account_sid"
	settingAuthToken  = "twilio.auth_token"
	settingObservedAC = "twilio.observed_auth_token"
	settingObservedSK = "twilio.observed_api_secret"
)

// recordCredential remembers the Basic-auth credential of a send in project
// settings so status webhooks can be signed later. AC usernames refresh the
// account SID and token; SK usernames refresh only the API secret fallback.
// Writes only when something changed.
func (a *Adapter) recordCredential(ctx context.Context, projectID, username, password string) {
	if username == "" {
		return
	}
	var accountSID, tokenKey string
	if strings.HasPrefix(username, "AC") {
		accountSID, tokenKey = username, password
	} else if strings.HasPrefix(username, "SK") {
		tokenKey = password
	} else {
		return
	}

	project, err := a.service.Store().GetProject(ctx, projectID)
	if err != nil {
		slog.Warn("twilio: record credential: load project", "project", projectID, "err", err)
		return
	}
	if project.Settings == nil {
		project.Settings = map[string]interface{}{}
	}
	changed := false
	setting := func(key, value string) {
		if value == "" {
			return
		}
		if current, _ := project.Settings[key].(string); current != value {
			project.Settings[key] = value
			changed = true
		}
	}
	if accountSID != "" {
		setting(settingAccountSID, accountSID)
		setting(settingObservedAC, tokenKey)
	} else {
		setting(settingObservedSK, tokenKey)
	}
	if !changed {
		return
	}
	if err := a.service.Store().UpdateProject(ctx, project); err != nil {
		slog.Warn("twilio: record credential: store project", "project", projectID, "err", err)
	}
}

// signingKey resolves the HMAC key per spec §7.4: explicit auth_token,
// else the token seen with an AC username, else the API secret. The second
// return reports the insecure SK fallback for inspector warnings.
func signingKey(project core.Project) (key string, skFallback bool) {
	settings := project.Settings
	if settings == nil {
		return "", false
	}
	if v, _ := settings[settingAuthToken].(string); v != "" {
		return v, false
	}
	if v, _ := settings[settingObservedAC].(string); v != "" {
		return v, false
	}
	if v, _ := settings[settingObservedSK].(string); v != "" {
		return v, true
	}
	return "", false
}

// StatusWebhook builds the form-encoded status callback for a message
// transition (spec §7.4). It returns false when no webhook applies.
func (a *Adapter) StatusWebhook(msg core.Message, event core.StatusEvent, project core.Project) (*adapterkit.WebhookRequest, bool) {
	if msg.Provider != a.Name() {
		return nil, false
	}
	if msg.CallbackURL == nil || *msg.CallbackURL == "" {
		return nil, false
	}

	var accountSID string
	if project.Settings != nil {
		accountSID, _ = project.Settings[settingAccountSID].(string)
	}
	params := map[string]string{
		"MessageSid":    msg.ProviderRef,
		"MessageStatus": string(event.Status),
		"SmsSid":        msg.ProviderRef,
		"SmsStatus":     string(event.Status),
		"AccountSid":    accountSID,
		"From":          msg.From,
		"To":            msg.To,
		"ApiVersion":    APIVersion,
	}
	if msg.ErrorCode != nil && *msg.ErrorCode != "" {
		params["ErrorCode"] = *msg.ErrorCode
	}

	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}
	headers := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	if key, skFallback := signingKey(project); key != "" {
		headers["X-Twilio-Signature"] = SignRequest(*msg.CallbackURL, params, key)
		if skFallback {
			slog.Warn("twilio: signing with API secret; configure twilio.auth_token", "project", project.ID)
		}
	} else {
		slog.Warn("twilio: no signing key known; sending unsigned status webhook", "project", project.ID)
	}
	return &adapterkit.WebhookRequest{
		URL:     *msg.CallbackURL,
		Body:    []byte(form.Encode()),
		Headers: headers,
	}, true
}
