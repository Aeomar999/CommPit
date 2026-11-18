package sendgrid

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// sendGridAddress is one recipient entry.
type sendGridAddress struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// sendGridPersonalization is one personalization block.
type sendGridPersonalization struct {
	To      []sendGridAddress `json:"to"`
	CC      []sendGridAddress `json:"cc"`
	BCC     []sendGridAddress `json:"bcc"`
	Subject string            `json:"subject"`
}

// sendGridContent is one content part.
type sendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// sendGridMail is the request body for POST /v3/mail/send.
type sendGridMail struct {
	Personalizations []sendGridPersonalization `json:"personalizations"`
	From             sendGridAddress           `json:"from"`
	ReplyTo          *sendGridAddress          `json:"reply_to"`
	Subject          string                    `json:"subject"`
	Content          []sendGridContent         `json:"content"`
	Categories       []string                  `json:"categories"`
	CustomArgs       map[string]string         `json:"custom_args"`
	SendAt           int64                     `json:"send_at"`
}

// newMessageID generates a SendGrid-format message id.
func newMessageID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "msg_fallback"
	}
	return "msg_" + hex.EncodeToString(buf[:])
}

func emailsOf(addrs []sendGridAddress) []string {
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if e := strings.TrimSpace(a.Email); e != "" {
			out = append(out, e)
		}
	}
	return out
}

// sendMail handles POST /v3/mail/send.
func (a *Adapter) sendMail(w http.ResponseWriter, r *http.Request) {
	var body sendGridMail
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.writeServiceError(w, core.NewValidationError("invalid JSON body", "body"))
		return
	}

	projectID := adapterkit.ProjectIDFromContext(r.Context())
	if projectID == "" {
		a.writeServiceError(w, core.NewInternal("missing project context"))
		return
	}
	a.recordAPIKey(r.Context(), projectID, adapterkit.CredentialFromContext(r.Context()))

	if len(body.Personalizations) == 0 {
		a.writeServiceError(w, core.NewValidationError("personalizations is required", "personalizations"))
		return
	}
	p := body.Personalizations[0]

	to := emailsOf(p.To)
	if len(to) == 0 {
		a.writeServiceError(w, core.NewValidationError("to is required", "to"))
		return
	}

	subject := body.Subject
	if subject == "" {
		subject = p.Subject
	}

	var bodyText, bodyHTML string
	for _, c := range body.Content {
		switch strings.ToLower(c.Type) {
		case "text/plain":
			bodyText = c.Value
		case "text/html":
			bodyHTML = c.Value
		}
	}

	msgID := newMessageID()
	req := core.SendRequest{
		Channel:     core.ChannelEmail,
		From:        body.From.Email,
		To:          to,
		CC:          emailsOf(p.CC),
		BCC:         emailsOf(p.BCC),
		Subject:     subject,
		BodyText:    bodyText,
		BodyHTML:    bodyHTML,
		Provider:    "sendgrid",
		ProviderRef: msgID,
	}

	resp, err := a.service.SendMessage(r.Context(), projectID, req)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	if resp == nil || resp.Message == nil {
		a.writeServiceError(w, core.NewInternal("send failed"))
		return
	}

	w.Header().Set("X-Message-Id", resp.Message.ProviderRef)
	w.WriteHeader(http.StatusAccepted)
}
