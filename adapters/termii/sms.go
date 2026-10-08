package termii

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// sendRequest mirrors Termii's SMS JSON body. To accepts one recipient or
// an array; Type and Channel are accepted and ignored.
type sendRequest struct {
	To      json.RawMessage `json:"to"`
	From    string          `json:"from"`
	Sms     string          `json:"sms"`
	Type    string          `json:"type"`
	Channel string          `json:"channel"`
}

// rejectedRecipient mirrors a core batch rejection in Termii's send shape.
type rejectedRecipient struct {
	To      string `json:"to"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// sendResponse mirrors Termii's send response. The sandbox has no billing,
// so balance and user are always null. Bulk responses add code "ok".
type sendResponse struct {
	MessageID int64               `json:"message_id"`
	Message   string              `json:"message"`
	Balance   *float64            `json:"balance"`
	User      *string             `json:"user"`
	Code      *string             `json:"code,omitempty"`
	Rejected  []rejectedRecipient `json:"rejected,omitempty"`
}

// parseRecipients decodes Termii's string-or-array To field.
func parseRecipients(raw json.RawMessage) ([]string, *core.Error) {
	if len(raw) == 0 {
		return nil, core.NewValidationError("recipient is required", "to")
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		if strings.TrimSpace(single) == "" {
			return nil, core.NewValidationError("recipient is required", "to")
		}
		return []string{single}, nil
	}
	var multi []string
	if err := json.Unmarshal(raw, &multi); err != nil || len(multi) == 0 {
		return nil, core.NewValidationError("recipient is required", "to")
	}
	return multi, nil
}

// checkSender enforces the project's sender allow-list. An absent or empty
// list accepts every sender (logged so the inspector can surface it once
// the M2-13 UI lands).
func (a *Adapter) checkSender(ctx context.Context, projectID, from string) *core.Error {
	project, err := a.service.Store().GetProject(ctx, projectID)
	if err != nil {
		return core.NewInternal("failed to load project: " + err.Error())
	}
	raw, ok := project.Settings[senderAllowListKey]
	if !ok {
		slog.Warn("termii sender allow-list empty; accepting every sender", "project", projectID)
		return nil
	}
	entries, ok := raw.([]interface{})
	if !ok || len(entries) == 0 {
		slog.Warn("termii sender allow-list empty; accepting every sender", "project", projectID)
		return nil
	}
	for _, entry := range entries {
		if s, ok := entry.(string); ok && s == from {
			return nil
		}
	}
	return core.NewInvalidSender("sender ID not in allow-list", "from")
}

// sendHandler serves one Termii SMS endpoint. Multi-recipient requests flow
// through the core batch path; bulk responses additionally carry code "ok".
func (a *Adapter) sendHandler(maxRecipients int, bulk bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req sendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			a.WriteError(w, core.NewValidationError("invalid JSON", ""))
			return
		}

		recipients, verr := parseRecipients(req.To)
		if verr != nil {
			a.WriteError(w, verr)
			return
		}
		if len(recipients) > maxRecipients {
			a.WriteError(w, core.NewValidationError(
				"too many recipients (max "+strconv.Itoa(maxRecipients)+")", "to"))
			return
		}
		if strings.TrimSpace(req.From) == "" {
			a.WriteError(w, core.NewValidationError("sender is required", "from"))
			return
		}
		if strings.TrimSpace(req.Sms) == "" {
			a.WriteError(w, core.NewValidationError("message body is required", "sms"))
			return
		}

		projectID := adapterkit.ProjectID(r)
		if verr := a.checkSender(r.Context(), projectID, req.From); verr != nil {
			a.WriteError(w, verr)
			return
		}

		id, err := newTermiiID()
		if err != nil {
			a.WriteError(w, core.NewInternal("failed to allocate message id"))
			return
		}

		resp, err := a.service.SendMessage(r.Context(), projectID, core.SendRequest{
			Channel:     core.ChannelSMS,
			From:        req.From,
			To:          recipients,
			BodyText:    req.Sms,
			Provider:    a.Name(),
			ProviderRef: strconv.FormatInt(id, 10),
		})
		if err != nil {
			a.writeServiceError(w, err)
			return
		}

		out := sendResponse{MessageID: id, Message: "Successfully Sent"}
		if bulk || (resp != nil && resp.Batch != nil) {
			code := "ok"
			out.Code = &code
		}
		if resp != nil && resp.Batch != nil && len(resp.Batch.Rejected) > 0 {
			out.Rejected = make([]rejectedRecipient, 0, len(resp.Batch.Rejected))
			for _, rej := range resp.Batch.Rejected {
				out.Rejected = append(out.Rejected, rejectedRecipient{
					To:      rej.To,
					Code:    rej.Code,
					Message: rej.Message,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}
