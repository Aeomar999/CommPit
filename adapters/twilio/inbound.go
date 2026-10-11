package twilio

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/url"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// inboundURLKey is the project setting standing in for the per-number SMS
// URL (spec §7.4): inbound messages are POSTed form-encoded to it.
const inboundURLKey = "twilio.inbound_url"

// Compile-time checks that Adapter satisfies the inbound contracts.
var (
	_ adapterkit.InboundNotifier = (*Adapter)(nil)
	_ adapterkit.ReplyParser     = (*Adapter)(nil)
)

// InboundWebhook builds the form-encoded inbound-SMS callback (spec §7.4).
// It returns false for non-Twilio or outbound messages, and when no
// inbound URL is configured.
func (a *Adapter) InboundWebhook(msg core.Message, project core.Project) (*adapterkit.WebhookRequest, bool) {
	if msg.Provider != a.Name() {
		return nil, false
	}
	if msg.Direction != core.DirectionInbound {
		return nil, false
	}
	var webhookURL string
	if project.Settings != nil {
		webhookURL, _ = project.Settings[inboundURLKey].(string)
	}
	if webhookURL == "" {
		return nil, false
	}

	var accountSID string
	if project.Settings != nil {
		accountSID, _ = project.Settings[settingAccountSID].(string)
	}
	params := map[string]string{
		"MessageSid": msg.ProviderRef,
		"AccountSid": accountSID,
		"From":       msg.From,
		"To":         msg.To,
		"Body":       msg.BodyText,
		"NumMedia":   "0",
	}

	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}
	headers := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	if key, _ := signingKey(project); key != "" {
		headers["X-Twilio-Signature"] = SignRequest(webhookURL, params, key)
	}
	return &adapterkit.WebhookRequest{URL: webhookURL, Body: []byte(form.Encode()), Headers: headers}, true
}

// twimlResponse mirrors the subset of TwiML an app server returns to an
// inbound webhook: <Response> carrying <Message> replies (spec §7.4).
type twimlResponse struct {
	XMLName  xml.Name       `xml:"Response"`
	Messages []twimlMessage `xml:"Message"`
}

type twimlMessage struct {
	To   string `xml:"to,attr"`
	Body string `xml:",chardata"`
}

// maxTwimlBody caps TwiML response parsing at 1 MB.
const maxTwimlBody = 1024 * 1024

// ParseReply turns <Response><Message> elements into outbound replies in
// the thread. Messages without a `to` attribute leave To empty for the
// caller (the worker) to default to the inbound sender; every reply mints
// a fresh SM ProviderRef.
func (a *Adapter) ParseReply(resp *http.Response) ([]core.OutboundReply, error) {
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxTwimlBody))
	if err != nil {
		return nil, err
	}
	var doc twimlResponse
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	replies := make([]core.OutboundReply, 0, len(doc.Messages))
	for _, m := range doc.Messages {
		ref, err := newSID("SM")
		if err != nil {
			return nil, err
		}
		replies = append(replies, core.OutboundReply{
			To:          m.To,
			Body:        m.Body,
			Provider:    a.Name(),
			ProviderRef: ref,
		})
	}
	return replies, nil
}
