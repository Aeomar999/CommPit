package vonage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

type vonageSMSRequest struct {
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
	From      string `json:"from"`
	To        string `json:"to"`
	Text      string `json:"text"`
}

type vonageMessage struct {
	To               string `json:"to"`
	MessageID        string `json:"message-id"`
	Status           string `json:"status"`
	RemainingBalance string `json:"remaining-balance"`
	MessagePrice     string `json:"message-price"`
	Network          string `json:"network"`
}

type vonageSMSResponse struct {
	MessageCount int             `json:"message-count"`
	Messages     []vonageMessage `json:"messages"`
}

func (a *Adapter) sendSMS(w http.ResponseWriter, r *http.Request) {
	var body vonageSMSRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.writeServiceError(w, core.NewValidationError("invalid JSON body", "body"))
		return
	}
	projectID := adapterkit.ProjectIDFromContext(r.Context())
	if projectID == "" {
		a.writeServiceError(w, core.NewInternal("missing project context"))
		return
	}
	if body.To == "" {
		a.writeServiceError(w, core.NewValidationError("to is required", "to"))
		return
	}
	a.recordAPIKey(r.Context(), projectID, body.APIKey)

	msgID := "vonage_" + newID()
	req := core.SendRequest{
		Channel:     core.ChannelSMS,
		From:        body.From,
		To:          []string{body.To},
		BodyText:    body.Text,
		Provider:    "vonage",
		ProviderRef: msgID,
	}
	resp, err := a.service.SendMessage(r.Context(), projectID, req)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	status := "0"
	if resp == nil || resp.Message == nil {
		status = "1"
	}
	to := body.To
	if resp != nil && resp.Message != nil {
		to = resp.Message.To
	}
	ref := msgID
	if resp != nil && resp.Message != nil {
		ref = resp.Message.ProviderRef
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(vonageSMSResponse{
		MessageCount: 1,
		Messages: []vonageMessage{{
			To:               to,
			MessageID:        ref,
			Status:           status,
			RemainingBalance: "10.00",
			MessagePrice:     "0.00",
			Network:          "0",
		}},
	})
}

// Verify v2
type vonageVerifyStartRequest struct {
	Brand    string `json:"brand"`
	Workflow []struct {
		Channel string `json:"channel"`
		To      string `json:"to"`
	} `json:"workflow"`
}

func (a *Adapter) startVerify(w http.ResponseWriter, r *http.Request) {
	var body vonageVerifyStartRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.writeServiceError(w, core.NewValidationError("invalid JSON body", "body"))
		return
	}
	projectID := adapterkit.ProjectIDFromContext(r.Context())
	if projectID == "" {
		a.writeServiceError(w, core.NewInternal("missing project context"))
		return
	}
	to := ""
	if len(body.Workflow) > 0 {
		to = body.Workflow[0].To
	}
	if to == "" {
		a.writeServiceError(w, core.NewValidationError("to is required", "to"))
		return
	}

	// Reuse native verification flow.
	vr := core.VerificationRequest{
		To:          to,
		Channel:     core.ChannelSMS,
		CodeLength:  6,
		TTLSeconds:  300,
		MaxAttempts: 3,
		Provider:    "vonage",
	}
	v, err := a.service.StartVerification(r.Context(), projectID, vr)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	reqID := ""
	if v != nil && v.Verification != nil {
		reqID = v.Verification.ID
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"request_id": reqID,
		"status":     "PENDING",
	})
}

func (a *Adapter) checkVerify(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Path // /v2/verify/{request_id}
	parts := strings.Split(requestID, "/")
	rid := parts[len(parts)-1]
	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	projectID := adapterkit.ProjectIDFromContext(r.Context())
	if projectID == "" {
		a.writeServiceError(w, core.NewInternal("missing project context"))
		return
	}
	res, err := a.service.CheckVerification(r.Context(), projectID, rid, core.CheckVerificationRequest{Code: body.Code})
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	status := "COMPLETED"
	if !res.Valid {
		status = "FAILED"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"request_id": rid,
		"status":     status,
	})
}

func (a *Adapter) recordAPIKey(ctx context.Context, projectID, apiKey string) {
	// Vonage uses api_key/api_secret; nothing to record for signing (no webhooks in M7).
	_ = projectID
	_ = apiKey
}

// newID generates a Vonage-format id.
func newID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "fallback"
	}
	return hex.EncodeToString(buf[:])
}
