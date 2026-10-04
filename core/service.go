package core

import (
	"context"
	"errors"
	"time"

	"github.com/Aeomar999/CommPit/extract"
	"github.com/Aeomar999/CommPit/phone"
)

type Service struct {
	store        Store
	blobStore    BlobStore
	bus          Bus
	simulator    Simulator
	clock        Clock
	resolver     ProjectResolver
	lifecycle    *LifecycleRunner
	stepDelay    time.Duration
	otpFixedCode *string
	phoneMode    phone.Mode
}

type ServiceConfig struct {
	Store        Store
	BlobStore    BlobStore
	Bus          Bus
	Simulator    Simulator
	Clock        Clock
	Resolver     ProjectResolver
	StepDelay    time.Duration
	OTPFixedCode *string
	PhoneMode    phone.Mode
}

func NewService(cfg ServiceConfig) *Service {
	if cfg.StepDelay == 0 {
		cfg.StepDelay = 300 * time.Millisecond
	}
	if cfg.PhoneMode == 0 {
		cfg.PhoneMode = phone.ModeValid
	}
	s := &Service{
		store:        cfg.Store,
		blobStore:    cfg.BlobStore,
		bus:          cfg.Bus,
		simulator:    cfg.Simulator,
		clock:        cfg.Clock,
		resolver:     cfg.Resolver,
		stepDelay:    cfg.StepDelay,
		otpFixedCode: cfg.OTPFixedCode,
		phoneMode:    cfg.PhoneMode,
	}
	s.lifecycle = NewLifecycleRunner(s)
	return s
}

func (s *Service) LifecycleRunner() *LifecycleRunner {
	return s.lifecycle
}

func (s *Service) Clock() Clock {
	return s.clock
}

func (s *Service) SendMessage(ctx context.Context, projectID string, req SendRequest) (*SendResponse, error) {
	if err := s.validateSendRequest(req); err != nil {
		return nil, err
	}

	if len(req.To) > 1 {
		return s.sendBatch(ctx, projectID, req)
	}

	to := req.To[0]
	if err := s.validateRecipient(ctx, projectID, to); err != nil {
		return nil, err
	}

	simErr, simResult := s.simulator.Evaluate(ctx, projectID, req)
	if simErr != nil {
		return nil, simErr
	}

	msg := &Message{
		ID:          NewMessageID(),
		ProjectID:   projectID,
		Channel:     req.Channel,
		Direction:   DirectionOutbound,
		Provider:    req.Provider,
		ProviderRef: req.ProviderRef,
		From:        req.From,
		To:          to,
		CC:          req.CC,
		BCC:         req.BCC,
		Subject:     req.Subject,
		BodyText:    req.BodyText,
		BodyHTML:    req.BodyHTML,
		Encoding:    "",
		Segments:    0,
		Status:      StatusQueued,
		CallbackURL: &req.CallbackURL,
		CreatedAt:   s.clock.Now(),
		UpdatedAt:   s.clock.Now(),
	}

	if req.Channel == ChannelSMS {
		enc, segs := phone.Analyze(req.BodyText)
		msg.Encoding = enc.String()
		msg.Segments = segs
	}

	if req.RawBlobID != nil {
		msg.RawBlobID = req.RawBlobID
	}

	if err := s.extractAndSet(msg); err != nil {
		return nil, err
	}

	if err := s.store.CreateMessage(ctx, msg); err != nil {
		return nil, NewInternal("failed to create message: " + err.Error())
	}

	s.bus.Publish(ctx, Event{
		Type:      EventMessageCreated,
		Payload:   msg,
		ProjectID: projectID,
		Timestamp: s.clock.Now(),
	})

	s.lifecycle.Schedule(msg, simResult)

	return &SendResponse{Message: msg}, nil
}

func (s *Service) sendBatch(ctx context.Context, projectID string, req SendRequest) (*SendResponse, error) {
	batch := &Batch{
		ID:        NewBatchID(),
		ProjectID: projectID,
		Provider:  req.Provider,
		Channel:   req.Channel,
		Total:     len(req.To),
		Counts: map[string]int{
			string(StatusQueued): len(req.To),
		},
		CreatedAt: s.clock.Now(),
	}

	if err := s.store.CreateBatch(ctx, batch); err != nil {
		return nil, NewInternal("failed to create batch: " + err.Error())
	}

	var messages []*Message
	for _, to := range req.To {
		if err := s.validateRecipient(ctx, projectID, to); err != nil {
			continue
		}

		simErr, _ := s.simulator.Evaluate(ctx, projectID, req)
		if simErr != nil {
			continue
		}

		msg := &Message{
			ID:          NewMessageID(),
			ProjectID:   projectID,
			BatchID:     &batch.ID,
			Channel:     req.Channel,
			Direction:   DirectionOutbound,
			Provider:    req.Provider,
			ProviderRef: req.ProviderRef,
			From:        req.From,
			To:          to,
			CC:          req.CC,
			BCC:         req.BCC,
			Subject:     req.Subject,
			BodyText:    req.BodyText,
			BodyHTML:    req.BodyHTML,
			Encoding:    "",
			Segments:    0,
			Status:      StatusQueued,
			CallbackURL: &req.CallbackURL,
			CreatedAt:   s.clock.Now(),
			UpdatedAt:   s.clock.Now(),
		}

		if req.Channel == ChannelSMS {
			enc, segs := phone.Analyze(req.BodyText)
			msg.Encoding = enc.String()
			msg.Segments = segs
		}

		if err := s.extractAndSet(msg); err != nil {
			continue
		}

		messages = append(messages, msg)
	}

	for _, msg := range messages {
		if err := s.store.CreateMessage(ctx, msg); err != nil {
			continue
		}
		s.bus.Publish(ctx, Event{
			Type:      EventMessageCreated,
			Payload:   msg,
			ProjectID: projectID,
			Timestamp: s.clock.Now(),
		})
		s.lifecycle.Schedule(msg, &SimResult{})
	}

	batch.Counts[string(StatusQueued)] = len(messages)
	batch.Total = len(messages)
	if err := s.store.UpdateBatch(ctx, batch); err != nil {
		return nil, NewInternal("failed to update batch: " + err.Error())
	}

	s.bus.Publish(ctx, Event{
		Type:      EventBatchUpdated,
		Payload:   batch,
		ProjectID: projectID,
		Timestamp: s.clock.Now(),
	})

	return &SendResponse{Batch: batch}, nil
}

func (s *Service) SendBatch(ctx context.Context, projectID string, req SendRequest) (*SendResponse, error) {
	return s.sendBatch(ctx, projectID, req)
}

func (s *Service) StartVerification(ctx context.Context, projectID string, req VerificationRequest) (*VerificationResponse, error) {
	if req.To == "" {
		return nil, NewValidationError("recipient is required", "to")
	}
	if req.Channel != ChannelSMS && req.Channel != ChannelEmail {
		return nil, NewValidationError("channel must be sms or email", "channel")
	}

	codeLength := req.CodeLength
	if codeLength == 0 {
		codeLength = 6
	}
	if codeLength < 4 || codeLength > 8 {
		return nil, NewValidationError("code_length must be between 4 and 8", "code_length")
	}

	ttl := req.TTLSeconds
	if ttl == 0 {
		ttl = 600
	}

	maxAttempts := req.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 5
	}

	var code string
	if s.otpFixedCode != nil {
		code = *s.otpFixedCode
	} else {
		code = generateCode(codeLength)
	}

	now := s.clock.Now()
	verification := &Verification{
		ID:          NewVerificationID(),
		ProjectID:   projectID,
		Provider:    req.Provider,
		ProviderRef: req.ProviderRef,
		ServiceRef:  req.ServiceRef,
		To:          req.To,
		Channel:     req.Channel,
		Code:        code,
		Status:      VerificationPending,
		Attempts:    0,
		MaxAttempts: maxAttempts,
		ExpiresAt:   now.Add(time.Duration(ttl) * time.Second),
		CreatedAt:   now,
	}

	sendReq := SendRequest{
		Channel:     req.Channel,
		From:        defaultVerificationSender(req.Channel),
		To:          []string{req.To},
		BodyText:    s.defaultVerificationText(req.Channel, code, req.ServiceRef),
		BodyHTML:    "",
		CallbackURL: "",
		Provider:    req.Provider,
		ProviderRef: req.ProviderRef,
	}

	sendResp, err := s.SendMessage(ctx, projectID, sendReq)
	if err != nil {
		return nil, err
	}

	verification.MessageID = sendResp.Message.ID
	if err := s.store.CreateVerification(ctx, verification); err != nil {
		return nil, NewInternal("failed to create verification: " + err.Error())
	}

	s.bus.Publish(ctx, Event{
		Type:      EventVerificationUpdated,
		Payload:   verification,
		ProjectID: projectID,
		Timestamp: s.clock.Now(),
	})

	return &VerificationResponse{Verification: verification, Message: sendResp.Message}, nil
}

func (s *Service) CheckVerification(ctx context.Context, projectID, verificationID string, req CheckVerificationRequest) (*CheckVerificationResponse, error) {
	v, err := s.store.GetVerification(ctx, projectID, verificationID)
	if err != nil {
		return nil, NewVerificationNotFound("verification not found", "id")
	}

	if v.Status != VerificationPending {
		return &CheckVerificationResponse{Valid: false, Status: v.Status}, nil
	}

	if s.clock.Now().After(v.ExpiresAt) {
		v.Status = VerificationExpired
		if err := s.store.UpdateVerification(ctx, v); err != nil {
			return nil, NewInternal("failed to update verification: " + err.Error())
		}
		s.bus.Publish(ctx, Event{
			Type:      EventVerificationUpdated,
			Payload:   v,
			ProjectID: projectID,
			Timestamp: s.clock.Now(),
		})
		return &CheckVerificationResponse{Valid: false, Status: VerificationExpired}, nil
	}

	v.Attempts++
	if req.Code == v.Code {
		v.Status = VerificationApproved
	} else if v.Attempts >= v.MaxAttempts {
		v.Status = VerificationMaxAttempts
	}

	if err := s.store.UpdateVerification(ctx, v); err != nil {
		return nil, NewInternal("failed to update verification: " + err.Error())
	}

	s.bus.Publish(ctx, Event{
		Type:      EventVerificationUpdated,
		Payload:   v,
		ProjectID: projectID,
		Timestamp: s.clock.Now(),
	})

	return &CheckVerificationResponse{Valid: req.Code == v.Code, Status: v.Status}, nil
}

func (s *Service) ReceiveInbound(ctx context.Context, projectID string, req InboundRequest) (*Message, error) {
	if req.From == "" || req.To == "" {
		return nil, NewValidationError("from and to are required", "")
	}

	if unsubscribed, err := s.store.IsUnsubscribed(ctx, projectID, req.From); err != nil {
		return nil, NewInternal("failed to check unsubscribe: " + err.Error())
	} else if unsubscribed {
		return nil, NewUnsubscribed("recipient has unsubscribed", "from")
	}

	bodyLower := req.Body
	if isSTOPKeyword(bodyLower) {
		if err := s.store.CreateUnsubscribe(ctx, &Unsubscribe{ProjectID: projectID, Number: req.From, At: s.clock.Now()}); err != nil {
			return nil, NewInternal("failed to create unsubscribe: " + err.Error())
		}
	}
	if isSTARTKeyword(bodyLower) {
		if err := s.store.DeleteUnsubscribe(ctx, projectID, req.From); err != nil {
			return nil, NewInternal("failed to delete unsubscribe: " + err.Error())
		}
	}

	msg := &Message{
		ID:        NewMessageID(),
		ProjectID: projectID,
		Channel:   ChannelSMS,
		Direction: DirectionInbound,
		Provider:  "native",
		From:      req.From,
		To:        req.To,
		BodyText:  req.Body,
		Status:    StatusReceived,
		CreatedAt: s.clock.Now(),
		UpdatedAt: s.clock.Now(),
	}

	if err := s.extractAndSet(msg); err != nil {
		return nil, err
	}

	if err := s.store.CreateMessage(ctx, msg); err != nil {
		return nil, NewInternal("failed to create inbound message: " + err.Error())
	}

	s.bus.Publish(ctx, Event{
		Type:      EventMessageCreated,
		Payload:   msg,
		ProjectID: projectID,
		Timestamp: s.clock.Now(),
	})

	return msg, nil
}

func (s *Service) validateSendRequest(req SendRequest) error {
	if req.From == "" {
		return NewValidationError("sender is required", "from")
	}
	if len(req.To) == 0 {
		return NewValidationError("recipient is required", "to")
	}
	if req.Channel == ChannelSMS {
		for _, to := range req.To {
			parsed, err := phone.Parse(to, s.phoneMode)
			if err != nil {
				return err
			}
			_ = parsed
		}
	}
	if req.Channel == ChannelEmail {
		if req.BodyText == "" && req.BodyHTML == "" {
			return NewValidationError("body_text or body_html is required", "body")
		}
	}
	return nil
}

func (s *Service) validateRecipient(ctx context.Context, projectID, to string) error {
	if unsubscribed, err := s.store.IsUnsubscribed(ctx, projectID, to); err != nil {
		return NewInternal("failed to check unsubscribe: " + err.Error())
	} else if unsubscribed {
		return NewUnsubscribed("recipient has unsubscribed", "to")
	}
	return nil
}

func (s *Service) extractAndSet(msg *Message) error {
	extracted := extract.ExtractAll(msg.BodyText)
	if msg.BodyHTML != "" {
		htmlExtracted := extract.ExtractAll(msg.BodyHTML)
		extracted.Codes = append(extracted.Codes, htmlExtracted.Codes...)
		extracted.Links = append(extracted.Links, htmlExtracted.Links...)
		if extracted.PrimaryLink == "" {
			extracted.PrimaryLink = htmlExtracted.PrimaryLink
		}
	}

	seenCodes := make(map[string]bool)
	uniqueCodes := make([]string, 0)
	for _, c := range extracted.Codes {
		if !seenCodes[c] {
			seenCodes[c] = true
			uniqueCodes = append(uniqueCodes, c)
		}
	}
	msg.ExtractedCodes = uniqueCodes

	seenLinks := make(map[string]bool)
	uniqueLinks := make([]string, 0)
	for _, l := range extracted.Links {
		if !seenLinks[l] {
			seenLinks[l] = true
			uniqueLinks = append(uniqueLinks, l)
		}
	}
	msg.ExtractedLinks = uniqueLinks

	if extracted.PrimaryLink != "" {
		msg.PrimaryLink = &extracted.PrimaryLink
	}
	return nil
}

func (s *Service) defaultVerificationText(channel Channel, code string, serviceRef *string) string {
	if channel == ChannelEmail {
		return "Your verification code is " + code
	}
	if serviceRef != nil && *serviceRef != "" {
		return "Your " + *serviceRef + " verification code is: " + code
	}
	return "Your verification code is " + code
}

func defaultVerificationSender(channel Channel) string {
	if channel == ChannelEmail {
		return "verify@example.com"
	}
	return "Verify"
}

func generateCode(length int) string {
	const digits = "0123456789"
	code := make([]byte, length)
	for i := range code {
		code[i] = digits[time.Now().UnixNano()%10]
		time.Sleep(1)
	}
	return string(code)
}

func isSTOPKeyword(body string) bool {
	stopKeywords := []string{"stop", "stopall", "unsubscribe", "cancel", "end", "quit"}
	bodyLower := ""
	for _, r := range body {
		if r >= 'A' && r <= 'Z' {
			bodyLower += string(r + 32)
		} else {
			bodyLower += string(r)
		}
	}
	bodyLower = trimSpace(bodyLower)
	for _, kw := range stopKeywords {
		if bodyLower == kw {
			return true
		}
	}
	return false
}

func isSTARTKeyword(body string) bool {
	startKeywords := []string{"start", "yes", "unstop"}
	bodyLower := ""
	for _, r := range body {
		if r >= 'A' && r <= 'Z' {
			bodyLower += string(r + 32)
		} else {
			bodyLower += string(r)
		}
	}
	bodyLower = trimSpace(bodyLower)
	for _, kw := range startKeywords {
		if bodyLower == kw {
			return true
		}
	}
	return false
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

var (
	ErrMessageNotFound = errors.New("message not found")
	ErrBatchNotFound   = errors.New("batch not found")
)

func (s *Service) Shutdown() {
	s.lifecycle.Stop()
}
