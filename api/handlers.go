package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Aeomar999/CommPit/config"
	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type projectKey struct{}

type Handlers struct {
	service         *core.Service
	projectResolver core.ProjectResolver
	sseHub          *SSEHub
	version         string
	securityCfg     *config.SecurityConfig
}

func NewHandlers(service *core.Service, resolver core.ProjectResolver, eventBus core.Bus, version string, securityCfg *config.SecurityConfig) *Handlers {
	return &Handlers{
		service:         service,
		projectResolver: resolver,
		sseHub:          NewSSEHub(eventBus),
		version:         version,
		securityCfg:     securityCfg,
	}
}

func (h *Handlers) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP) //nolint:staticcheck // RealIP is standard chi middleware, acceptable in local sandbox
	r.Use(chimiddleware.Recoverer)

	// Security middleware (Host allow-list, X-Mocksms header, UI auth)
	if h.securityCfg != nil {
		r.Use(middleware.SecurityMiddleware(h.securityCfg))
	}

	// Public health check
	r.Get("/healthz", h.Healthz)

	// Long-lived endpoints: mount outside the global timeout middleware
	// Use adapter functions to extract params from request
	r.Get("/events", func(w http.ResponseWriter, r *http.Request) {
		h.SSEEvents(w, r, SseEventsParams{Project: strPtr(r.URL.Query().Get("project"))})
	})
	r.Get("/messages/wait", func(w http.ResponseWriter, r *http.Request) {
		var since *time.Time
		if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
			if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
				since = &t
			}
		}
		var timeout *int
		if timeoutStr := r.URL.Query().Get("timeout"); timeoutStr != "" {
			if d, err := time.ParseDuration(timeoutStr + "s"); err == nil {
				sec := int(d.Seconds())
				timeout = &sec
			} else if d, err := time.ParseDuration(timeoutStr); err == nil {
				sec := int(d.Seconds())
				timeout = &sec
			}
		}
		h.WaitForMessage(w, r, WaitForMessageParams{
			To:      strPtr(r.URL.Query().Get("to")),
			Channel: (*WaitForMessageParamsChannel)(strPtr(r.URL.Query().Get("channel"))),
			Since:   since,
			Timeout: timeout,
			Project: strPtr(r.URL.Query().Get("project")),
		})
	})

	// Remaining endpoints with 60s timeout
	r.Group(func(r chi.Router) {
		r.Use(chimiddleware.Timeout(60 * time.Second))

		// Write endpoints - require Bearer auth, no ?project= allowed, use "default" project if no key
		r.Group(func(r chi.Router) {
			r.Use(h.authMiddlewareWrite)

			// Send
			r.Post("/sms", h.SendSMS)
			r.Post("/email", h.SendEmail)

			// Verifications write
			r.Post("/verifications", h.StartVerification)
			r.Post("/verifications/{id}/check", h.CheckVerification)
			r.Post("/verifications/{id}/expire", h.ExpireVerification)

			// Inbound
			r.Post("/inbound", h.SimulateInbound)

			// Projects write
			r.Patch("/projects/{id}", h.UpdateProject)
			r.Post("/projects/{id}/credentials", h.LinkCredential)
		})

		// Read/test endpoints - accept Bearer auth or ?project=, validate project exists
		r.Group(func(r chi.Router) {
			r.Use(h.authMiddlewareRead)

			// Verifications read
			r.Get("/verifications", func(w http.ResponseWriter, r *http.Request) {
				var limit *int
				if l := r.URL.Query().Get("limit"); l != "" {
					if v, err := strconv.Atoi(l); err == nil {
						limit = &v
					}
				}
				h.ListVerifications(w, r, ListVerificationsParams{
					Limit:   limit,
					Cursor:  strPtr(r.URL.Query().Get("cursor")),
					Project: strPtr(r.URL.Query().Get("project")),
				})
			})
			r.Get("/verifications/{id}", func(w http.ResponseWriter, r *http.Request) {
				h.GetVerification(w, r, chi.URLParam(r, "id"), GetVerificationParams{
					Project: strPtr(r.URL.Query().Get("project")),
				})
			})

			// Messages
			r.Get("/messages", func(w http.ResponseWriter, r *http.Request) {
				var limit *int
				if l := r.URL.Query().Get("limit"); l != "" {
					if v, err := strconv.Atoi(l); err == nil {
						limit = &v
					}
				}
				var since *time.Time
				if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
					if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
						since = &t
					}
				}
				h.ListMessages(w, r, ListMessagesParams{
					Channel:   (*ListMessagesParamsChannel)(strPtr(r.URL.Query().Get("channel"))),
					To:        strPtr(r.URL.Query().Get("to")),
					From:      strPtr(r.URL.Query().Get("from")),
					Status:    (*ListMessagesParamsStatus)(strPtr(r.URL.Query().Get("status"))),
					BatchId:   strPtr(r.URL.Query().Get("batch_id")),
					Direction: (*ListMessagesParamsDirection)(strPtr(r.URL.Query().Get("direction"))),
					Since:     since,
					Limit:     limit,
					Cursor:    strPtr(r.URL.Query().Get("cursor")),
					Project:   strPtr(r.URL.Query().Get("project")),
				})
			})
			r.Delete("/messages", func(w http.ResponseWriter, r *http.Request) {
				h.DeleteMessages(w, r, DeleteMessagesParams{
					Project: r.URL.Query().Get("project"),
				})
			})
			r.Get("/messages/{id}", func(w http.ResponseWriter, r *http.Request) {
				h.GetMessage(w, r, chi.URLParam(r, "id"), GetMessageParams{
					Project: strPtr(r.URL.Query().Get("project")),
				})
			})
			r.Get("/messages/{id}/raw", func(w http.ResponseWriter, r *http.Request) {
				h.GetMessageRaw(w, r, chi.URLParam(r, "id"), GetMessageRawParams{
					Project: strPtr(r.URL.Query().Get("project")),
				})
			})

			// Test helpers
			r.Get("/otp/latest", func(w http.ResponseWriter, r *http.Request) {
				var since *time.Time
				if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
					if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
						since = &t
					}
				}
				h.GetLatestOTP(w, r, GetLatestOTPParams{
					To:      r.URL.Query().Get("to"),
					Since:   since,
					Project: strPtr(r.URL.Query().Get("project")),
				})
			})
			r.Get("/emails/latest", func(w http.ResponseWriter, r *http.Request) {
				var since *time.Time
				if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
					if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
						since = &t
					}
				}
				h.GetLatestEmail(w, r, GetLatestEmailParams{
					To:      r.URL.Query().Get("to"),
					Since:   since,
					Project: strPtr(r.URL.Query().Get("project")),
				})
			})

			// Attachments
			r.Get("/attachments/{id}", func(w http.ResponseWriter, r *http.Request) {
				h.GetAttachment(w, r, chi.URLParam(r, "id"), GetAttachmentParams{
					Project: strPtr(r.URL.Query().Get("project")),
				})
			})

			// Batches
			r.Get("/batches/{id}", func(w http.ResponseWriter, r *http.Request) {
				h.GetBatch(w, r, chi.URLParam(r, "id"), GetBatchParams{
					Project: strPtr(r.URL.Query().Get("project")),
				})
			})

			// Request logs
			r.Get("/requests", func(w http.ResponseWriter, r *http.Request) {
				var limit *int
				if l := r.URL.Query().Get("limit"); l != "" {
					if v, err := strconv.Atoi(l); err == nil {
						limit = &v
					}
				}
				h.ListRequestLogs(w, r, ListRequestLogsParams{
					Limit:   limit,
					Cursor:  strPtr(r.URL.Query().Get("cursor")),
					Project: strPtr(r.URL.Query().Get("project")),
				})
			})
			r.Get("/requests/{id}", func(w http.ResponseWriter, r *http.Request) {
				h.GetRequestLog(w, r, chi.URLParam(r, "id"), GetRequestLogParams{
					Project: strPtr(r.URL.Query().Get("project")),
				})
			})

			// Webhooks
			r.Get("/webhooks", func(w http.ResponseWriter, r *http.Request) {
				var limit *int
				if l := r.URL.Query().Get("limit"); l != "" {
					if v, err := strconv.Atoi(l); err == nil {
						limit = &v
					}
				}
				h.ListWebhooks(w, r, ListWebhooksParams{
					Limit:   limit,
					Cursor:  strPtr(r.URL.Query().Get("cursor")),
					Project: strPtr(r.URL.Query().Get("project")),
				})
			})
			r.Post("/webhooks/{id}/replay", func(w http.ResponseWriter, r *http.Request) {
				h.ReplayWebhook(w, r, chi.URLParam(r, "id"), ReplayWebhookParams{
					Project: strPtr(r.URL.Query().Get("project")),
				})
			})

			// Projects read
			r.Get("/projects", func(w http.ResponseWriter, r *http.Request) {
				var limit *int
				if l := r.URL.Query().Get("limit"); l != "" {
					if v, err := strconv.Atoi(l); err == nil {
						limit = &v
					}
				}
				h.ListProjects(w, r, ListProjectsParams{
					Limit:  limit,
					Cursor: strPtr(r.URL.Query().Get("cursor")),
				})
			})
			r.Get("/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
				h.GetProject(w, r, chi.URLParam(r, "id"))
			})
		})
	})

	return r
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (h *Handlers) Healthz(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, map[string]string{
		"status":  "ok",
		"version": h.version,
	})
}

func (h *Handlers) authMiddlewareWrite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		var projectID string

		if auth != "" && len(auth) > 7 && auth[:7] == "Bearer " {
			key := auth[7:]
			var err error
			projectID, err = h.projectResolver.Resolve(r.Context(), "native", key)
			if err != nil {
				h.writeError(w, r, err)
				return
			}
		} else {
			// No Bearer token - use default project for write endpoints
			var err error
			projectID, err = h.projectResolver.Resolve(r.Context(), "native", "default")
			if err != nil {
				h.writeError(w, r, core.NewUnauthorized("authentication required"))
				return
			}
		}

		ctx := context.WithValue(r.Context(), projectKey{}, projectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handlers) authMiddlewareRead(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		var projectID string

		if auth != "" && len(auth) > 7 && auth[:7] == "Bearer " {
			key := auth[7:]
			var err error
			projectID, err = h.projectResolver.Resolve(r.Context(), "native", key)
			if err != nil {
				h.writeError(w, r, err)
				return
			}
		} else if projectQuery := r.URL.Query().Get("project"); projectQuery != "" {
			// ?project= only allowed on read/test endpoints
			// Verify the project exists
			_, err := h.service.Store().GetProject(r.Context(), projectQuery)
			if err != nil {
				h.writeError(w, r, core.NewNotFound("project not found", ""))
				return
			}
			projectID = projectQuery
		} else {
			// No auth and no project query - try default
			var err error
			projectID, err = h.projectResolver.Resolve(r.Context(), "native", "default")
			if err != nil {
				h.writeError(w, r, core.NewUnauthorized("authentication required"))
				return
			}
		}

		ctx := context.WithValue(r.Context(), projectKey{}, projectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handlers) getProjectID(r *http.Request) (string, error) {
	projectID, ok := r.Context().Value(projectKey{}).(string)
	if !ok || projectID == "" {
		return "", errors.New("projectID not found in context")
	}
	return projectID, nil
}

func (h *Handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var ce *core.Error
	if !errors.As(err, &ce) {
		slog.Error("unexpected error", "path", r.URL.Path, "err", err)
		ce = core.NewInternal("internal error")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ce.HTTPStatus())
	_ = json.NewEncoder(w).Encode(map[string]any{"error": ce})
}

func (h *Handlers) SendSMS(w http.ResponseWriter, r *http.Request) {
	var req SendSMSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, core.NewValidationError("invalid JSON", ""))
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
		h.writeError(w, r, core.NewValidationError("recipient required", "to"))
		return
	}

	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

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
		h.writeError(w, r, err)
		return
	}

	if resp.Message != nil {
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, resp.Message)
	} else if resp.Batch != nil {
		render.Status(r, http.StatusAccepted)
		render.JSON(w, r, resp.Batch)
	}
}

func (h *Handlers) SendEmail(w http.ResponseWriter, r *http.Request) {
	var req SendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, core.NewValidationError("invalid JSON", ""))
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

	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

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
		h.writeError(w, r, err)
		return
	}

	if resp.Message != nil {
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, resp.Message)
	} else if resp.Batch != nil {
		render.Status(r, http.StatusAccepted)
		render.JSON(w, r, resp.Batch)
	}
}

func (h *Handlers) StartVerification(w http.ResponseWriter, r *http.Request) {
	var req StartVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, core.NewValidationError("invalid JSON", ""))
		return
	}

	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

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
		h.writeError(w, r, err)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, VerificationResponse{
		Verification: convertVerification(resp.Verification),
		Message:      convertMessage(resp.Message),
	})
}

func (h *Handlers) CheckVerification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeError(w, r, core.NewValidationError("verification ID required", "id"))
		return
	}

	var req CheckVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, core.NewValidationError("invalid JSON", ""))
		return
	}

	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	resp, err := h.service.CheckVerification(r.Context(), projectID, id, core.CheckVerificationRequest{
		Code: req.Code,
	})
	if err != nil {
		h.writeError(w, r, err)
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
		h.writeError(w, r, core.NewValidationError("verification ID required", "id"))
		return
	}

	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	v, err := h.Store().GetVerification(r.Context(), projectID, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	v.Status = core.VerificationExpired
	if err := h.service.Store().UpdateVerification(r.Context(), v); err != nil {
		h.writeError(w, r, err)
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

func (h *Handlers) GetVerification(w http.ResponseWriter, r *http.Request, id string, params GetVerificationParams) {
	if id == "" {
		h.writeError(w, r, core.NewValidationError("verification ID required", "id"))
		return
	}

	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	v, err := h.Store().GetVerification(r.Context(), projectID, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	render.JSON(w, r, convertVerification(v))
}

func (h *Handlers) ListVerifications(w http.ResponseWriter, r *http.Request, params ListVerificationsParams) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		// parse limit
	}

	verifications, nextCursor, err := h.Store().ListVerifications(r.Context(), projectID, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		h.writeError(w, r, err)
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

func (h *Handlers) ListMessages(w http.ResponseWriter, r *http.Request, params ListMessagesParams) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	filter := core.MessageFilter{
		Limit:  50,
		Cursor: r.URL.Query().Get("cursor"),
	}

	if ch := r.URL.Query().Get("channel"); ch != "" {
		c := core.Channel(ch)
		filter.Channel = &c
	}
	if to := r.URL.Query().Get("to"); to != "" {
		normalized := core.NormalizePhone(to)
		filter.To = &normalized
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
		h.writeError(w, r, err)
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

func (h *Handlers) GetMessage(w http.ResponseWriter, r *http.Request, id string, params GetMessageParams) {
	if id == "" {
		h.writeError(w, r, core.NewValidationError("message ID required", "id"))
		return
	}

	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	msg, err := h.Store().GetMessage(r.Context(), projectID, id)
	if err != nil {
		h.writeError(w, r, err)
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

func (h *Handlers) GetMessageRaw(w http.ResponseWriter, r *http.Request, id string, params GetMessageRawParams) {
	if id == "" {
		h.writeError(w, r, core.NewValidationError("message ID required", "id"))
		return
	}

	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	msg, err := h.service.Store().GetMessage(r.Context(), projectID, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if msg.RawBlobID == nil {
		h.writeError(w, r, core.NewNotFound("no raw content available", ""))
		return
	}

	reader, err := h.service.BlobStore().Get(r.Context(), *msg.RawBlobID)
	if err != nil {
		h.writeError(w, r, err)
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

func (h *Handlers) DeleteMessages(w http.ResponseWriter, r *http.Request, params DeleteMessagesParams) {
	// DELETE /messages requires explicit project via Bearer token or ?project= query
	// The authMiddlewareRead already handles this - it will return 400 if no project is available
	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	// Check if project was explicitly provided via ?project= or Bearer token
	// If neither was provided, the auth middleware would have resolved "default" which we don't want for DELETE
	auth := r.Header.Get("Authorization")
	projectQuery := r.URL.Query().Get("project")
	if (auth == "" || len(auth) <= 7 || auth[:7] != "Bearer ") && projectQuery == "" {
		h.writeError(w, r, core.NewValidationError("project required", "project"))
		return
	}

	err = h.service.Store().DeleteMessages(r.Context(), projectID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) WaitForMessage(w http.ResponseWriter, r *http.Request, params WaitForMessageParams) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	to := r.URL.Query().Get("to")
	if to != "" {
		to = core.NormalizePhone(to)
	}
	channelStr := r.URL.Query().Get("channel")
	sinceStr := r.URL.Query().Get("since")
	timeoutStr := r.URL.Query().Get("timeout")

	// Parse timeout: accept seconds (e.g., "10") or Go duration (e.g., "10s"), cap at 60s
	timeout := 10 * time.Second
	if timeoutStr != "" {
		// Try parsing as seconds first
		if sec, err := time.ParseDuration(timeoutStr + "s"); err == nil {
			timeout = sec
		} else if d, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = d
		}
	}
	if timeout > 60*time.Second {
		timeout = 60 * time.Second
	}

	var channel *core.Channel
	if channelStr != "" {
		c := core.Channel(channelStr)
		channel = &c
	}

	// Default since to request arrival time
	requestTime := h.service.Clock().Now()
	var since *time.Time
	if sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = &t
		}
	} else {
		since = &requestTime
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	// Check for existing messages first
	messages, _, err := h.service.Store().ListMessages(ctx, projectID, core.MessageFilter{
		To:      &to,
		Channel: channel,
		Since:   since,
		Limit:   1,
	})
	if err == nil && len(messages) > 0 {
		render.JSON(w, r, messages[0])
		return
	}

	// Subscribe to message.created events for this project and recipient
	msgCh := make(chan *core.Message, 1)
	sub := h.service.Bus().Subscribe(string(core.EventMessageCreated), func(e core.Event) {
		if msg, ok := e.Payload.(*core.Message); ok {
			// Filter by project
			if msg.ProjectID != projectID {
				return
			}
			// Filter by recipient
			if to != "" && msg.To != to {
				return
			}
			// Filter by channel
			if channel != nil && msg.Channel != *channel {
				return
			}
			// Filter by since
			if since != nil && !msg.CreatedAt.After(*since) {
				return
			}
			select {
			case msgCh <- msg:
			default:
			}
		}
	})
	defer sub.Unsubscribe()

	select {
	case <-ctx.Done():
		h.writeError(w, r, core.NewWaitTimeout("wait timeout"))
		return
	case msg := <-msgCh:
		render.JSON(w, r, msg)
		return
	}
}

func (h *Handlers) GetLatestOTP(w http.ResponseWriter, r *http.Request, params GetLatestOTPParams) {
	_, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	to := r.URL.Query().Get("to")
	if to == "" {
		h.writeError(w, r, core.NewValidationError("to parameter required", "to"))
		return
	}

	h.writeError(w, r, core.NewValidationError("not implemented", ""))
}

func (h *Handlers) GetLatestEmail(w http.ResponseWriter, r *http.Request, params GetLatestEmailParams) {
	_, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	to := r.URL.Query().Get("to")
	if to == "" {
		h.writeError(w, r, core.NewValidationError("to parameter required", "to"))
		return
	}

	h.writeError(w, r, core.NewValidationError("not implemented", ""))
}

func (h *Handlers) SimulateInbound(w http.ResponseWriter, r *http.Request) {
	var req InboundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, core.NewValidationError("invalid JSON", ""))
		return
	}

	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	msg, err := h.service.ReceiveInbound(r.Context(), projectID, core.InboundRequest{
		From: req.From,
		To:   req.To,
		Body: req.Body,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, msg)
}

func (h *Handlers) GetAttachment(w http.ResponseWriter, r *http.Request, id string, params GetAttachmentParams) {
	if id == "" {
		h.writeError(w, r, core.NewValidationError("attachment ID required", "id"))
		return
	}

	att, err := h.Store().GetAttachment(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	reader, err := h.BlobStore().Get(r.Context(), att.BlobID)
	if err != nil {
		h.writeError(w, r, err)
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

func (h *Handlers) GetBatch(w http.ResponseWriter, r *http.Request, id string, params GetBatchParams) {
	if id == "" {
		h.writeError(w, r, core.NewValidationError("batch ID required", "id"))
		return
	}

	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	batch, err := h.Store().GetBatch(r.Context(), projectID, id)
	if err != nil {
		h.writeError(w, r, err)
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

func (h *Handlers) ListRequestLogs(w http.ResponseWriter, r *http.Request, params ListRequestLogsParams) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	logs, nextCursor, err := h.Store().ListRequestLogs(r.Context(), projectID, 50, r.URL.Query().Get("cursor"))
	if err != nil {
		h.writeError(w, r, err)
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

func (h *Handlers) GetRequestLog(w http.ResponseWriter, r *http.Request, id string, params GetRequestLogParams) {
	if id == "" {
		h.writeError(w, r, core.NewValidationError("request log ID required", "id"))
		return
	}

	log, err := h.Store().GetRequestLog(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
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

func (h *Handlers) ListWebhooks(w http.ResponseWriter, r *http.Request, params ListWebhooksParams) {
	_, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	webhooks, err := h.Store().ListPendingWebhooks(r.Context(), 50)
	if err != nil {
		h.writeError(w, r, err)
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

func (h *Handlers) ReplayWebhook(w http.ResponseWriter, r *http.Request, id string, params ReplayWebhookParams) {
	if id == "" {
		h.writeError(w, r, core.NewValidationError("webhook ID required", "id"))
		return
	}

	projectID, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	webhook, err := h.Store().GetWebhookDelivery(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Verify the webhook belongs to the project
	if webhook.ProjectID != projectID {
		h.writeError(w, r, core.NewNotFound("webhook not found", ""))
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
		h.writeError(w, r, err)
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

func (h *Handlers) ListProjects(w http.ResponseWriter, r *http.Request, params ListProjectsParams) {
	_, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	projects, nextCursor, err := h.Store().ListProjects(r.Context(), 50, r.URL.Query().Get("cursor"))
	if err != nil {
		h.writeError(w, r, err)
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

func (h *Handlers) GetProject(w http.ResponseWriter, r *http.Request, id string) {
	if id == "" {
		h.writeError(w, r, core.NewValidationError("project ID required", "id"))
		return
	}

	_, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	// For GET /projects/{id}, the project in the path should match the authenticated project
	// or the project specified via ?project=
	project, err := h.Store().GetProject(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// If a specific project was requested via ?project=, verify it matches
	projectQuery := r.URL.Query().Get("project")
	if projectQuery != "" && projectQuery != project.ID {
		h.writeError(w, r, core.NewNotFound("project not found", ""))
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
		h.writeError(w, r, core.NewValidationError("project ID required", "id"))
		return
	}

	_, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, core.NewValidationError("invalid JSON", ""))
		return
	}

	project, err := h.Store().GetProject(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Settings != nil {
		project.Settings = *req.Settings
	}

	if err := h.Store().UpdateProject(r.Context(), project); err != nil {
		h.writeError(w, r, err)
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
		h.writeError(w, r, core.NewValidationError("project ID required", "id"))
		return
	}

	_, err := h.getProjectID(r)
	if err != nil {
		h.writeError(w, r, core.NewUnauthorized("authentication required"))
		return
	}

	var req LinkCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, core.NewValidationError("invalid JSON", ""))
		return
	}

	_, err = h.projectResolver.Resolve(r.Context(), req.Provider, req.Key)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	render.JSON(w, r, map[string]string{"message": "credential linked"})
}

func (h *Handlers) SSEEvents(w http.ResponseWriter, r *http.Request, params SseEventsParams) {
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
