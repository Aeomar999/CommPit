package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/core"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Handlers struct {
	service         *core.Service
	projectResolver core.ProjectResolver
	sseHub          *SSEHub
}

func NewHandlers(service *core.Service, resolver core.ProjectResolver, eventBus *bus.EventBus) *Handlers {
	return &Handlers{
		service:         service,
		projectResolver: resolver,
		sseHub:          NewSSEHub(eventBus),
	}
}

func (h *Handlers) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Public health check
	r.Get("/healthz", h.Healthz)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(h.authMiddleware)

		// Send
		r.Post("/sms", h.SendSMS)
		r.Post("/email", h.SendEmail)

		// Verifications
		r.Post("/verifications", h.StartVerification)
		r.Post("/verifications/{id}/check", h.CheckVerification)
		r.Post("/verifications/{id}/expire", h.ExpireVerification)
		r.Get("/verifications", h.ListVerifications)
		r.Get("/verifications/{id}", h.GetVerification)

		// Messages
		r.Get("/messages", h.ListMessages)
		r.Delete("/messages", h.DeleteMessages)
		r.Get("/messages/{id}", h.GetMessage)
		r.Get("/messages/{id}/raw", h.GetMessageRaw)
		r.Get("/messages/wait", h.WaitForMessage)

		// Test helpers
		r.Get("/otp/latest", h.GetLatestOTP)
		r.Get("/emails/latest", h.GetLatestEmail)

		// Inbound
		r.Post("/inbound", h.SimulateInbound)

		// Attachments
		r.Get("/attachments/{id}", h.GetAttachment)

		// Batches
		r.Get("/batches/{id}", h.GetBatch)

		// Request logs
		r.Get("/requests", h.ListRequestLogs)
		r.Get("/requests/{id}", h.GetRequestLog)

		// Webhooks
		r.Get("/webhooks", h.ListWebhooks)
		r.Post("/webhooks/{id}/replay", h.ReplayWebhook)

		// Projects
		r.Get("/projects", h.ListProjects)
		r.Get("/projects/{id}", h.GetProject)
		r.Patch("/projects/{id}", h.UpdateProject)
		r.Post("/projects/{id}/credentials", h.LinkCredential)

		// SSE
		r.Get("/events", h.SSEEvents)
	})

	return r
}

func (h *Handlers) Healthz(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, map[string]string{
		"status":  "ok",
		"version": "0.1.0",
	})
}

func (h *Handlers) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		var projectID string
		var err error

		if auth != "" && len(auth) > 7 && auth[:7] == "Bearer " {
			key := auth[7:]
			projectID, err = h.projectResolver.Resolve(r.Context(), "native", key)
		}

		if projectID == "" {
			projectID = r.URL.Query().Get("project")
		}

		if projectID == "" {
			projectID, err = h.projectResolver.Resolve(r.Context(), "native", "default")
			if err != nil {
				h.error(w, r, core.NewValidationError("authentication required", ""), http.StatusUnauthorized)
				return
			}
		}

		ctx := context.WithValue(r.Context(), "projectID", projectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handlers) getProjectID(r *http.Request) string {
	return r.Context().Value("projectID").(string)
}

func (h *Handlers) error(w http.ResponseWriter, r *http.Request, err error, status int) {
	var ce *core.Error
	if core.IsError(err, "") {
		ce = err.(*core.Error)
	} else {
		ce = core.NewInternal(err.Error())
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    ce.Code,
			"message": ce.Message,
			"field":   ce.Field,
		},
	})
}

func (h *Handlers) SendSMS(w http.ResponseWriter, r *http.Request) {
	var req SendSMSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.error(w, r, core.NewValidationError("invalid JSON", ""), http.StatusBadRequest)
		return
	}

	var to []string
	if req.To.union != nil {
		var single string
		if err := json.Unmarshal(req.To.union, &single); err == nil && single != "" {
			to = []string{single}
		} else {
			var multi []string
			if err := json.Unmarshal(req.To.union, &multi); err == nil {
				to = multi
			}
		}
	}

	if len(to) == 0 {
		h.error(w, r, core.NewValidationError("recipient required", "to"), http.StatusBadRequest)
		return
	}

	projectID := h.getProjectID(r)

	sendReq := core.SendRequest{
		Channel:     core.ChannelSMS,
		From:        req.From,
		To:          to,
		BodyText:    req.Body,
		CallbackURL: "",
		Provider:    "native",
	}

	if req.CallbackUrl != nil {
		sendReq.CallbackURL = *req.CallbackUrl
	}

	resp, err := h.service.SendMessage(r.Context(), projectID, sendReq)
	if err != nil {
		h.error(w, r, err, http.StatusBadRequest)
		return
	}

	if resp.Message != nil {
		w.WriteHeader(http.StatusCreated)
		render.JSON(w, r, resp.Message)
	} else if resp.Batch != nil {
		w.WriteHeader(http.StatusAccepted)
		render.JSON(w, r, resp.Batch)
	}
}

func (h *Handlers) SendEmail(w http.ResponseWriter, r *http.Request) {
	var req SendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.error(w, r, core.NewValidationError("invalid JSON", ""), http.StatusBadRequest)
		return
	}

	to := make([]string, len(req.To))
	for i, e := range req.To {
		to[i] = string(e)
	}

	cc := make([]string, 0)
	if req.Cc != nil {
		cc = make([]string, len(*req.Cc))
		for i, e := range *req.Cc {
			cc[i] = string(e)
		}
	}

	bcc := make([]string, 0)
	if req.Bcc != nil {
		bcc = make([]string, len(*req.Bcc))
		for i, e := range *req.Bcc {
			bcc[i] = string(e)
		}
	}

	var attachments []core.AttachmentInput
	if req.Attachments != nil {
		for _, a := range *req.Attachments {
			attachments = append(attachments, core.AttachmentInput{
				Filename:      a.Filename,
				ContentType:   a.ContentType,
				ContentBase64: base64.StdEncoding.EncodeToString(a.ContentBase64),
				InlineCID:     "",
			})
			if a.InlineCid != nil {
				attachments[len(attachments)-1].InlineCID = *a.InlineCid
			}
		}
	}

	projectID := h.getProjectID(r)

	sendReq := core.SendRequest{
		Channel:     core.ChannelEmail,
		From:        string(req.From),
		To:          to,
		CC:          cc,
		BCC:         bcc,
		Subject:     req.Subject,
		BodyText:    "",
		BodyHTML:    "",
		Attachments: attachments,
		CallbackURL: "",
		Provider:    "native",
	}

	if req.Text != nil {
		sendReq.BodyText = *req.Text
	}
	if req.Html != nil {
		sendReq.BodyHTML = *req.Html
	}
	if req.CallbackUrl != nil {
		sendReq.CallbackURL = *req.CallbackUrl
	}

	resp, err := h.service.SendMessage(r.Context(), projectID, sendReq)
	if err != nil {
		h.error(w, r, err, http.StatusBadRequest)
		return
	}

	if resp.Message != nil {
		w.WriteHeader(http.StatusCreated)
		render.JSON(w, r, resp.Message)
	} else if resp.Batch != nil {
		w.WriteHeader(http.StatusAccepted)
		render.JSON(w, r, resp.Batch)
	}
}

func (h *Handlers) StartVerification(w http.ResponseWriter, r *http.Request) {
	var req StartVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.error(w, r, core.NewValidationError("invalid JSON", ""), http.StatusBadRequest)
		return
	}

	projectID := h.getProjectID(r)

	codeLength := 6
	if req.CodeLength != nil {
		codeLength = *req.CodeLength
	}

	ttl := 600
	if req.TtlSeconds != nil {
		ttl = *req.TtlSeconds
	}

	maxAttempts := 5
	if req.MaxAttempts != nil {
		maxAttempts = *req.MaxAttempts
	}

	verReq := core.VerificationRequest{
		To:          req.To,
		Channel:     core.Channel(req.Channel),
		CodeLength:  codeLength,
		TTLSeconds:  ttl,
		MaxAttempts: maxAttempts,
		Provider:    "native",
	}

	resp, err := h.service.StartVerification(r.Context(), projectID, verReq)
	if err != nil {
		h.error(w, r, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	render.JSON(w, r, VerificationResponse{
		Verification: convertVerification(resp.Verification),
		Message:      convertMessage(resp.Message),
	})
}

func (h *Handlers) CheckVerification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.error(w, r, core.NewValidationError("verification ID required", "id"), http.StatusBadRequest)
		return
	}

	var req CheckVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.error(w, r, core.NewValidationError("invalid JSON", ""), http.StatusBadRequest)
		return
	}

	projectID := h.getProjectID(r)

	resp, err := h.service.CheckVerification(r.Context(), projectID, id, core.CheckVerificationRequest{
		Code: req.Code,
	})
	if err != nil {
		h.error(w, r, err, http.StatusNotFound)
		return
	}

	render.JSON(w, r, CheckVerificationResponse{
		Valid:  resp.Valid,
		Status: convertCheckStatus(resp.Status),
	})
}

func (h *Handlers) ExpireVerification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.error(w, r, core.NewValidationError("verification ID required", "id"), http.StatusBadRequest)
		return
	}

	projectID := h.getProjectID(r)

	v, err := h.Store().GetVerification(r.Context(), projectID, id)
	if err != nil {
		h.error(w, r, err, http.StatusNotFound)
		return
	}

	v.Status = core.VerificationExpired
	if err := h.service.Store().UpdateVerification(r.Context(), v); err != nil {
		h.error(w, r, err, http.StatusInternalServerError)
		return
	}

	h.service.Bus().Publish(r.Context(), core.Event{
		Type:      core.EventVerificationUpdated,
		Payload:   v,
		ProjectID: projectID,
		Timestamp: h.service.Clock().Now(),
	})

	render.JSON(w, r, v)
}

func (h *Handlers) GetVerification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.error(w, r, core.NewValidationError("verification ID required", "id"), http.StatusBadRequest)
		return
	}

	projectID := h.getProjectID(r)

	v, err := h.Store().GetVerification(r.Context(), projectID, id)
	if err != nil {
		h.error(w, r, err, http.StatusNotFound)
		return
	}

	render.JSON(w, r, convertVerification(v))
}

func (h *Handlers) ListVerifications(w http.ResponseWriter, r *http.Request) {
	projectID := h.getProjectID(r)

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		// parse limit
	}

	verifications, nextCursor, err := h.Store().ListVerifications(r.Context(), projectID, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		h.error(w, r, err, http.StatusInternalServerError)
		return
	}

	converted := make([]*Verification, len(verifications))
	for i, v := range verifications {
		converted[i] = convertVerification(v)
	}

	render.JSON(w, r, map[string]interface{}{
		"verifications": converted,
		"next_cursor":   nextCursor,
	})
}

func (h *Handlers) ListMessages(w http.ResponseWriter, r *http.Request) {
	projectID := h.getProjectID(r)

	filter := core.MessageFilter{
		Limit:  50,
		Cursor: r.URL.Query().Get("cursor"),
	}

	if ch := r.URL.Query().Get("channel"); ch != "" {
		c := core.Channel(ch)
		filter.Channel = &c
	}
	if to := r.URL.Query().Get("to"); to != "" {
		filter.To = &to
	}
	if from := r.URL.Query().Get("from"); from != "" {
		filter.From = &from
	}
	if status := r.URL.Query().Get("status"); status != "" {
		s := core.MessageStatus(status)
		filter.Status = &s
	}
	if batchID := r.URL.Query().Get("batch_id"); batchID != "" {
		filter.BatchID = &batchID
	}
	if direction := r.URL.Query().Get("direction"); direction != "" {
		d := core.Direction(direction)
		filter.Direction = &d
	}
	if since := r.URL.Query().Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			filter.Since = &t
		}
	}

	messages, nextCursor, err := h.Store().ListMessages(r.Context(), projectID, filter)
	if err != nil {
		h.error(w, r, err, http.StatusInternalServerError)
		return
	}

	converted := make([]*Message, len(messages))
	for i, m := range messages {
		converted[i] = convertMessage(m)
	}

	render.JSON(w, r, map[string]interface{}{
		"messages":    converted,
		"next_cursor": nextCursor,
	})
}

func (h *Handlers) GetMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.error(w, r, core.NewValidationError("message ID required", "id"), http.StatusBadRequest)
		return
	}

	projectID := h.getProjectID(r)

	msg, err := h.Store().GetMessage(r.Context(), projectID, id)
	if err != nil {
		h.error(w, r, err, http.StatusNotFound)
		return
	}

	events, _ := h.Store().GetStatusEvents(r.Context(), id)

	convertedEvents := make([]*StatusEvent, len(events))
	for i, e := range events {
		convertedEvents[i] = &StatusEvent{
			Id:        &e.ID,
			MessageId: &e.MessageID,
			Status:    ptr(StatusEventStatus(e.Status)),
			ErrorCode: e.ErrorCode,
			At:        &e.At,
		}
	}

	render.JSON(w, r, map[string]interface{}{
		"message":       convertMessage(msg),
		"status_events": convertedEvents,
	})
}

func (h *Handlers) GetMessageRaw(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.error(w, r, core.NewValidationError("message ID required", "id"), http.StatusBadRequest)
		return
	}

	projectID := h.getProjectID(r)

	msg, err := h.service.Store().GetMessage(r.Context(), projectID, id)
	if err != nil {
		h.error(w, r, err, http.StatusNotFound)
		return
	}

	if msg.RawBlobID == nil {
		h.error(w, r, core.NewValidationError("no raw content available", ""), http.StatusNotFound)
		return
	}

	reader, err := h.service.BlobStore().Get(r.Context(), *msg.RawBlobID)
	if err != nil {
		h.error(w, r, err, http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "message/rfc822")
	w.Header().Set("Content-Disposition", "attachment; filename=\"message.eml\"")

	data := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	w.Write(data)
}

func (h *Handlers) DeleteMessages(w http.ResponseWriter, r *http.Request) {
	projectID := h.getProjectID(r)

	err := h.service.Store().DeleteMessages(r.Context(), projectID)
	if err != nil {
		h.error(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) WaitForMessage(w http.ResponseWriter, r *http.Request) {
	projectID := h.getProjectID(r)

	to := r.URL.Query().Get("to")
	channelStr := r.URL.Query().Get("channel")
	sinceStr := r.URL.Query().Get("since")
	timeoutStr := r.URL.Query().Get("timeout")

	timeout := 10 * time.Second
	if timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr + "s"); err == nil {
			timeout = d
		}
	}

	var channel *core.Channel
	if channelStr != "" {
		c := core.Channel(channelStr)
		channel = &c
	}

	var since *time.Time
	if sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = &t
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.error(w, r, core.NewInternal("wait timeout"), http.StatusRequestTimeout)
			return
		case <-ticker.C:
			messages, _, _ := h.service.Store().ListMessages(ctx, projectID, core.MessageFilter{
				To:      &to,
				Channel: channel,
				Since:   since,
				Limit:   1,
			})
			if len(messages) > 0 {
				render.JSON(w, r, messages[0])
				return
			}
		}
	}
}

func (h *Handlers) GetLatestOTP(w http.ResponseWriter, r *http.Request) {
	to := r.URL.Query().Get("to")
	if to == "" {
		h.error(w, r, core.NewValidationError("to parameter required", "to"), http.StatusBadRequest)
		return
	}

	h.error(w, r, core.NewValidationError("not implemented", ""), http.StatusNotImplemented)
}

func (h *Handlers) GetLatestEmail(w http.ResponseWriter, r *http.Request) {
	to := r.URL.Query().Get("to")
	if to == "" {
		h.error(w, r, core.NewValidationError("to parameter required", "to"), http.StatusBadRequest)
		return
	}

	h.error(w, r, core.NewValidationError("not implemented", ""), http.StatusNotImplemented)
}

func (h *Handlers) SimulateInbound(w http.ResponseWriter, r *http.Request) {
	var req InboundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.error(w, r, core.NewValidationError("invalid JSON", ""), http.StatusBadRequest)
		return
	}

	projectID := h.getProjectID(r)

	msg, err := h.service.ReceiveInbound(r.Context(), projectID, core.InboundRequest{
		From: req.From,
		To:   req.To,
		Body: req.Body,
	})
	if err != nil {
		h.error(w, r, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	render.JSON(w, r, msg)
}

func (h *Handlers) GetAttachment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.error(w, r, core.NewValidationError("attachment ID required", "id"), http.StatusBadRequest)
		return
	}

	att, err := h.Store().GetAttachment(r.Context(), id)
	if err != nil {
		h.error(w, r, err, http.StatusNotFound)
		return
	}

	reader, err := h.BlobStore().Get(r.Context(), att.BlobID)
	if err != nil {
		h.error(w, r, err, http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", att.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+att.Filename+"\"")

	data := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	w.Write(data)
}

func (h *Handlers) GetBatch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.error(w, r, core.NewValidationError("batch ID required", "id"), http.StatusBadRequest)
		return
	}

	projectID := h.getProjectID(r)

	batch, err := h.Store().GetBatch(r.Context(), projectID, id)
	if err != nil {
		h.error(w, r, err, http.StatusNotFound)
		return
	}

	render.JSON(w, r, map[string]interface{}{
		"id":         batch.ID,
		"project_id": batch.ProjectID,
		"provider":   batch.Provider,
		"channel":    batch.Channel,
		"total":      batch.Total,
		"counts":     batch.Counts,
		"created_at": batch.CreatedAt,
	})
}

func (h *Handlers) ListRequestLogs(w http.ResponseWriter, r *http.Request) {
	projectID := h.getProjectID(r)

	logs, nextCursor, err := h.Store().ListRequestLogs(r.Context(), projectID, 50, r.URL.Query().Get("cursor"))
	if err != nil {
		h.error(w, r, err, http.StatusInternalServerError)
		return
	}

	converted := make([]map[string]interface{}, len(logs))
	for i, l := range logs {
		converted[i] = map[string]interface{}{
			"id":              l.ID,
			"project_id":      l.ProjectID,
			"adapter":         l.Adapter,
			"method":          l.Method,
			"path":            l.Path,
			"request_headers": l.RequestHeaders,
			"request_body":    string(l.RequestBody),
			"response_status": l.ResponseStatus,
			"response_body":   string(l.ResponseBody),
			"duration_ms":     l.DurationMS,
			"created_at":      l.CreatedAt,
		}
	}

	render.JSON(w, r, map[string]interface{}{
		"logs":        converted,
		"next_cursor": nextCursor,
	})
}

func (h *Handlers) GetRequestLog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.error(w, r, core.NewValidationError("request log ID required", "id"), http.StatusBadRequest)
		return
	}

	log, err := h.Store().GetRequestLog(r.Context(), id)
	if err != nil {
		h.error(w, r, err, http.StatusNotFound)
		return
	}

	render.JSON(w, r, map[string]interface{}{
		"id":              log.ID,
		"project_id":      log.ProjectID,
		"adapter":         log.Adapter,
		"method":          log.Method,
		"path":            log.Path,
		"request_headers": log.RequestHeaders,
		"request_body":    string(log.RequestBody),
		"response_status": log.ResponseStatus,
		"response_body":   string(log.ResponseBody),
		"duration_ms":     log.DurationMS,
		"created_at":      log.CreatedAt,
	})
}

func (h *Handlers) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	_ = h.getProjectID(r)

	webhooks, err := h.Store().ListPendingWebhooks(r.Context(), 50)
	if err != nil {
		h.error(w, r, err, http.StatusInternalServerError)
		return
	}

	converted := make([]map[string]interface{}, len(webhooks))
	for i, w := range webhooks {
		converted[i] = map[string]interface{}{
			"id":              w.ID,
			"project_id":      w.ProjectID,
			"message_id":      w.MessageID,
			"verification_id": w.VerificationID,
			"kind":            w.Kind,
			"url":             w.URL,
			"payload":         w.Payload,
			"headers":         w.Headers,
			"attempt":         w.Attempt,
			"status":          w.Status,
			"response_status": w.ResponseStatus,
			"response_body":   w.ResponseBody,
			"next_retry_at":   w.NextRetryAt,
			"created_at":      w.CreatedAt,
		}
	}

	render.JSON(w, r, map[string]interface{}{
		"deliveries":  converted,
		"next_cursor": nil,
	})
}

func (h *Handlers) ReplayWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.error(w, r, core.NewValidationError("webhook ID required", "id"), http.StatusBadRequest)
		return
	}

	webhook, err := h.Store().GetWebhookDelivery(r.Context(), id)
	if err != nil {
		h.error(w, r, err, http.StatusNotFound)
		return
	}

	newDelivery := &core.WebhookDelivery{
		ID:             core.NewWebhookDeliveryID(),
		ProjectID:      webhook.ProjectID,
		MessageID:      webhook.MessageID,
		VerificationID: webhook.VerificationID,
		Kind:           webhook.Kind,
		URL:            webhook.URL,
		Payload:        webhook.Payload,
		Headers:        webhook.Headers,
		Attempt:        webhook.Attempt + 1,
		Status:         core.WebhookPending,
		CreatedAt:      h.service.Clock().Now(),
	}

	if err := h.Store().CreateWebhookDelivery(r.Context(), newDelivery); err != nil {
		h.error(w, r, err, http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, map[string]interface{}{
		"id":              newDelivery.ID,
		"project_id":      newDelivery.ProjectID,
		"message_id":      newDelivery.MessageID,
		"verification_id": newDelivery.VerificationID,
		"kind":            newDelivery.Kind,
		"url":             newDelivery.URL,
		"payload":         newDelivery.Payload,
		"headers":         newDelivery.Headers,
		"attempt":         newDelivery.Attempt,
		"status":          newDelivery.Status,
		"response_status": newDelivery.ResponseStatus,
		"response_body":   newDelivery.ResponseBody,
		"next_retry_at":   newDelivery.NextRetryAt,
		"created_at":      newDelivery.CreatedAt,
	})
}

func (h *Handlers) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, nextCursor, err := h.Store().ListProjects(r.Context(), 50, r.URL.Query().Get("cursor"))
	if err != nil {
		h.error(w, r, err, http.StatusInternalServerError)
		return
	}

	converted := make([]map[string]interface{}, len(projects))
	for i, p := range projects {
		converted[i] = map[string]interface{}{
			"id":         p.ID,
			"name":       p.Name,
			"settings":   p.Settings,
			"created_at": p.CreatedAt,
		}
	}

	render.JSON(w, r, map[string]interface{}{
		"projects":    converted,
		"next_cursor": nextCursor,
	})
}

func (h *Handlers) GetProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.error(w, r, core.NewValidationError("project ID required", "id"), http.StatusBadRequest)
		return
	}

	project, err := h.Store().GetProject(r.Context(), id)
	if err != nil {
		h.error(w, r, err, http.StatusNotFound)
		return
	}

	render.JSON(w, r, map[string]interface{}{
		"id":         project.ID,
		"name":       project.Name,
		"settings":   project.Settings,
		"created_at": project.CreatedAt,
	})
}

func (h *Handlers) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.error(w, r, core.NewValidationError("project ID required", "id"), http.StatusBadRequest)
		return
	}

	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.error(w, r, core.NewValidationError("invalid JSON", ""), http.StatusBadRequest)
		return
	}

	project, err := h.Store().GetProject(r.Context(), id)
	if err != nil {
		h.error(w, r, err, http.StatusNotFound)
		return
	}

	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Settings != nil {
		project.Settings = *req.Settings
	}

	if err := h.Store().UpdateProject(r.Context(), project); err != nil {
		h.error(w, r, err, http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, map[string]interface{}{
		"id":         project.ID,
		"name":       project.Name,
		"settings":   project.Settings,
		"created_at": project.CreatedAt,
	})
}

func (h *Handlers) LinkCredential(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.error(w, r, core.NewValidationError("project ID required", "id"), http.StatusBadRequest)
		return
	}

	var req LinkCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.error(w, r, core.NewValidationError("invalid JSON", ""), http.StatusBadRequest)
		return
	}

	_, err := h.projectResolver.Resolve(r.Context(), req.Provider, req.Key)
	if err != nil {
		h.error(w, r, err, http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, map[string]string{"message": "credential linked"})
}

func (h *Handlers) SSEEvents(w http.ResponseWriter, r *http.Request) {
	h.sseHub.SSEHandler(w, r)
}

// Store accessor for generated interface compatibility
func (h *Handlers) Store() core.Store {
	return h.service.Store()
}

func (h *Handlers) BlobStore() core.BlobStore {
	return h.service.BlobStore()
}

func (h *Handlers) Bus() core.Bus {
	return h.service.Bus()
}

func (h *Handlers) Clock() core.Clock {
	return h.service.Clock()
}

func convertVerification(v *core.Verification) *Verification {
	if v == nil {
		return nil
	}
	return &Verification{
		Id:          &v.ID,
		ProjectId:   &v.ProjectID,
		Provider:    &v.Provider,
		ProviderRef: &v.ProviderRef,
		ServiceRef:  v.ServiceRef,
		To:          &v.To,
		Channel:     ptr(VerificationChannel(v.Channel)),
		Code:        &v.Code,
		Status:      ptr(VerificationStatus(v.Status)),
		Attempts:    &v.Attempts,
		MaxAttempts: &v.MaxAttempts,
		ExpiresAt:   &v.ExpiresAt,
		MessageId:   &v.MessageID,
		CreatedAt:   &v.CreatedAt,
	}
}

func convertMessage(m *core.Message) *Message {
	if m == nil {
		return nil
	}
	return &Message{
		Id:             &m.ID,
		ProjectId:      &m.ProjectID,
		BatchId:        m.BatchID,
		Channel:        ptr(MessageChannel(m.Channel)),
		Direction:      ptr(MessageDirection(m.Direction)),
		Provider:       &m.Provider,
		ProviderRef:    &m.ProviderRef,
		From:           &m.From,
		To:             &m.To,
		Cc:             &m.CC,
		Bcc:            &m.BCC,
		Subject:        &m.Subject,
		BodyText:       &m.BodyText,
		BodyHtml:       &m.BodyHTML,
		Encoding:       ptr(MessageEncoding(m.Encoding)),
		Segments:       &m.Segments,
		Status:         ptr(MessageStatus(m.Status)),
		ErrorCode:      m.ErrorCode,
		ErrorMessage:   m.ErrorMessage,
		CallbackUrl:    m.CallbackURL,
		ExtractedCodes: &m.ExtractedCodes,
		ExtractedLinks: &m.ExtractedLinks,
		PrimaryLink:    m.PrimaryLink,
		CreatedAt:      &m.CreatedAt,
		UpdatedAt:      &m.UpdatedAt,
	}
}

func ptr[T any](v T) *T {
	return &v
}

func convertCheckStatus(status core.VerificationStatus) CheckVerificationResponseStatus {
	return CheckVerificationResponseStatus(status)
}
