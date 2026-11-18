package resend

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// resendMail is the request body for POST /emails.
type resendMail struct {
	From     string   `json:"from"`
	To       []string `json:"to"`
	CC       []string `json:"cc"`
	BCC      []string `json:"bcc"`
	Subject  string   `json:"subject"`
	Text     string   `json:"text"`
	HTML     string   `json:"html"`
	ReplyTo  string   `json:"reply_to"`
}

func newMessageID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "resend_fallback"
	}
	return hex.EncodeToString(buf[:])
}

// sendEmail handles POST /emails.
func (a *Adapter) sendEmail(w http.ResponseWriter, r *http.Request) {
	var body resendMail
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.writeServiceError(w, core.NewValidationError("invalid JSON body", "body"))
		return
	}

	projectID := adapterkit.ProjectIDFromContext(r.Context())
	if projectID == "" {
		a.writeServiceError(w, core.NewInternal("missing project context"))
		return
	}

	if len(body.To) == 0 {
		a.writeServiceError(w, core.NewValidationError("to is required", "to"))
		return
	}

	a.recordAPIKey(r.Context(), projectID, adapterkit.CredentialFromContext(r.Context()))

	msgID := newMessageID()
	req := core.SendRequest{
		Channel:     core.ChannelEmail,
		From:        body.From,
		To:          body.To,
		CC:          body.CC,
		BCC:         body.BCC,
		Subject:     body.Subject,
		BodyText:    body.Text,
		BodyHTML:    body.HTML,
		Provider:    "resend",
		ProviderRef: msgID,
	}
	if body.ReplyTo != "" {
		req.CallbackURL = body.ReplyTo
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": resp.Message.ProviderRef})
}
