package core

import (
	"context"
	cryptoRand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
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
	if cfg.Simulator == nil {
		cfg.Simulator = NoopSimulator{}
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
	if len(req.To) > 1 {
		return s.sendBatch(ctx, projectID, req)
	}

	if err := s.validateSendRequest(req); err != nil {
		return nil, err
	}

	to := strings.TrimSpace(req.To[0])
	if req.Channel == ChannelSMS {
		parsed, err := phone.Parse(to, s.phoneMode)
		if err != nil {
			var pe *phone.Error
			if errors.As(err, &pe) {
				return nil, NewError(ErrorCode(pe.Code), pe.Message, pe.Field)
			}
			return nil, NewInvalidNumber("invalid phone number", "to")
		}
		to = parsed.E164
	}

	if err := s.validateRecipient(ctx, projectID, to); err != nil {
		return nil, err
	}

	simReq := req
	simReq.To = []string{to}
	simErr, simResult := s.simulator.Evaluate(ctx, projectID, simReq)
	if simErr != nil {
		return nil, simErr
	}

	var callbackURL *string
	if cb := strings.TrimSpace(req.CallbackURL); cb != "" {
		callbackURL = &cb
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
		CallbackURL: callbackURL,
		CreatedAt:   s.clock.Now(),
		UpdatedAt:   s.clock.Now(),
	}

	if req.Channel == ChannelSMS {
		enc, segs := phone.Analyze(req.BodyText)
		if segs > phone.MaxSegments(enc) {
			return nil, NewValidationError("message too long", "body")
		}
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

	s.lifecycle.Schedule(msg.ProjectID, msg.ID, simResult)

	return &SendResponse{Message: msg}, nil
}

func (s *Service) sendBatch(ctx context.Context, projectID string, req SendRequest) (*SendResponse, error) {
	if req.From == "" {
		return nil, NewValidationError("sender is required", "from")
	}
	if len(req.To) == 0 {
		return nil, NewValidationError("recipient is required", "to")
	}
	if req.Channel == ChannelEmail && req.BodyText == "" && req.BodyHTML == "" {
		return nil, NewValidationError("body_text or body_html is required", "body")
	}

	var callbackURL *string
	if cb := strings.TrimSpace(req.CallbackURL); cb != "" {
		callbackURL = &cb
	}

	batch := &Batch{
		ID:        NewBatchID(),
		ProjectID: projectID,
		Provider:  req.Provider,
		Channel:   req.Channel,
		Total:     0,
		Counts:    map[string]int{},
		CreatedAt: s.clock.Now(),
	}

	var messages []*Message
	var simResults []*SimResult
	var rejected []BatchRejectedRecipient

	for _, rawTo := range req.To {
		to := strings.TrimSpace(rawTo)
		if to == "" {
			rejected = append(rejected, BatchRejectedRecipient{
				To:      rawTo,
				Code:    "validation_error",
				Message: "recipient is required",
			})
			continue
		}

		if req.Channel == ChannelSMS {
			parsed, err := phone.Parse(to, s.phoneMode)
			if err != nil {
				var pe *phone.Error
				code := "invalid_number"
				msg := "invalid phone number"
				if errors.As(err, &pe) {
					code = pe.Code
					msg = pe.Message
				}
				rejected = append(rejected, BatchRejectedRecipient{
					To:      rawTo,
					Code:    code,
					Message: msg,
				})
				continue
			}
			to = parsed.E164
		}

		if unsubscribed, err := s.store.IsUnsubscribed(ctx, projectID, to); err != nil {
			return nil, NewInternal("failed to check unsubscribe: " + err.Error())
		} else if unsubscribed {
			rejected = append(rejected, BatchRejectedRecipient{
				To:      rawTo,
				Code:    "unsubscribed",
				Message: "recipient has unsubscribed",
			})
			continue
		}

		singleReq := req
		singleReq.To = []string{to}
		simErr, simResult := s.simulator.Evaluate(ctx, projectID, singleReq)
		if simErr != nil {
			rejected = append(rejected, BatchRejectedRecipient{
				To:      rawTo,
				Code:    string(simErr.Code),
				Message: simErr.Message,
			})
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
			CallbackURL: callbackURL,
			CreatedAt:   s.clock.Now(),
			UpdatedAt:   s.clock.Now(),
		}

		if req.Channel == ChannelSMS {
			enc, segs := phone.Analyze(req.BodyText)
			if segs > phone.MaxSegments(enc) {
				rejected = append(rejected, BatchRejectedRecipient{
					To:      to,
					Code:    "validation_error",
					Message: "message too long",
				})
				continue
			}
			msg.Encoding = enc.String()
			msg.Segments = segs
		}

		if err := s.extractAndSet(msg); err != nil {
			continue
		}

		messages = append(messages, msg)
		simResults = append(simResults, simResult)
	}

	batch.Total = len(messages)
	batch.Counts[string(StatusQueued)] = len(messages)
	batch.Rejected = rejected

	err := s.store.Transaction(ctx, func(txStore Store) error {
		if err := txStore.CreateBatch(ctx, batch); err != nil {
			return err
		}
		for _, msg := range messages {
			if err := txStore.CreateMessage(ctx, msg); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, NewInternal("failed to create batch: " + err.Error())
	}

	for i, msg := range messages {
		s.bus.Publish(ctx, Event{
			Type:      EventMessageCreated,
			Payload:   msg,
			ProjectID: projectID,
			Timestamp: s.clock.Now(),
		})
		s.lifecycle.Schedule(msg.ProjectID, msg.ID, simResults[i])
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

	to := strings.TrimSpace(req.To)
	if req.Channel == ChannelSMS {
		parsed, err := phone.Parse(to, s.phoneMode)
		if err != nil {
			var pe *phone.Error
			if errors.As(err, &pe) {
				return nil, NewError(ErrorCode(pe.Code), pe.Message, pe.Field)
			}
			return nil, NewInvalidNumber("invalid phone number", "to")
		}
		to = parsed.E164
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
	switch {
	case req.CustomCode != nil:
		// An explicitly supplied code (e.g. Termii email OTP) always wins,
		// including over the otpFixedCode test override.
		code = *req.CustomCode
		if len(code) < 4 || len(code) > 10 {
			return nil, NewValidationError("custom code must be between 4 and 10 characters", "code")
		}
	case s.otpFixedCode != nil:
		code = *s.otpFixedCode
	default:
		code = generateCode(codeLength)
	}

	now := s.clock.Now()
	verification := &Verification{
		ID:          NewVerificationID(),
		ProjectID:   projectID,
		Provider:    req.Provider,
		ProviderRef: req.ProviderRef,
		ServiceRef:  req.ServiceRef,
		To:          to,
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
		To:          []string{to},
		BodyText:    s.defaultVerificationText(req.Channel, code, req.ServiceRef, req.ServiceLabel),
		BodyHTML:    "",
		CallbackURL: "",
		Provider:    req.Provider,
		ProviderRef: req.ProviderRef,
	}
	if req.BodyText != nil && *req.BodyText != "" {
		sendReq.BodyText = *req.BodyText
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
	var (
		res         *CheckVerificationResponse
		errToReturn error
		eventToSend *Verification
	)

	err := s.store.Transaction(ctx, func(txStore Store) error {
		v, err := txStore.GetVerification(ctx, projectID, verificationID)
		if err != nil {
			if IsError(err, ErrCodeVerificationNotFound) {
				errToReturn = err
				return nil
			}
			return err
		}

		// Spec §7.2: Unknown, expired, approved or canceled verification -> 404 verification_not_found
		if v.Status == VerificationApproved || v.Status == VerificationCanceled || v.Status == VerificationExpired {
			errToReturn = NewVerificationNotFound(fmt.Sprintf("verification is %s", v.Status), "id")
			return nil
		}

		// Spec §7.2: attempts exhausted -> 429 max_attempts
		if v.Status == VerificationMaxAttempts || v.Attempts >= v.MaxAttempts {
			errToReturn = NewMaxAttempts("maximum attempts reached", "code")
			return nil
		}

		// Expired verification -> 404 verification_not_found
		if s.clock.Now().After(v.ExpiresAt) {
			v.Status = VerificationExpired
			if err := txStore.UpdateVerification(ctx, v); err != nil {
				return err
			}
			eventToSend = v
			errToReturn = NewVerificationNotFound("verification expired", "id")
			return nil
		}

		// Atomically increment attempts
		v.Attempts++

		if req.Code == v.Code {
			v.Status = VerificationApproved
			if err := txStore.UpdateVerification(ctx, v); err != nil {
				return err
			}
			eventToSend = v
			res = &CheckVerificationResponse{
				Valid:  true,
				Status: VerificationApproved,
			}
			return nil
		}

		// Wrong code:
		if v.Attempts >= v.MaxAttempts {
			v.Status = VerificationMaxAttempts
			if err := txStore.UpdateVerification(ctx, v); err != nil {
				return err
			}
			eventToSend = v
			errToReturn = NewMaxAttempts("maximum attempts reached", "code")
			return nil
		}

		v.Status = VerificationPending
		if err := txStore.UpdateVerification(ctx, v); err != nil {
			return err
		}
		eventToSend = v
		res = &CheckVerificationResponse{
			Valid:  false,
			Status: VerificationPending,
		}
		return nil
	})

	if err != nil {
		return nil, NewInternal("failed to check verification: " + err.Error())
	}

	if eventToSend != nil {
		s.bus.Publish(ctx, Event{
			Type:      EventVerificationUpdated,
			Payload:   eventToSend,
			ProjectID: projectID,
			Timestamp: s.clock.Now(),
		})
	}

	if errToReturn != nil {
		return nil, errToReturn
	}

	return res, nil
}

func (s *Service) ReceiveInbound(ctx context.Context, projectID string, req InboundRequest) (*Message, error) {
	if req.From == "" || req.To == "" {
		return nil, NewValidationError("from and to are required", "")
	}

	from := strings.TrimSpace(req.From)
	to := strings.TrimSpace(req.To)
	if parsed, err := phone.Parse(from, s.phoneMode); err == nil {
		from = parsed.E164
	}
	if parsed, err := phone.Parse(to, s.phoneMode); err == nil {
		to = parsed.E164
	}

	bodyClean := strings.TrimSpace(strings.ToLower(req.Body))
	if isSTOPKeyword(bodyClean) {
		if err := s.store.CreateUnsubscribe(ctx, &Unsubscribe{
			ProjectID: projectID,
			Number:    from,
			At:        s.clock.Now(),
		}); err != nil {
			return nil, NewInternal("failed to create unsubscribe: " + err.Error())
		}
	} else if isSTARTKeyword(bodyClean) {
		if err := s.store.DeleteUnsubscribe(ctx, projectID, from); err != nil {
			return nil, NewInternal("failed to delete unsubscribe: " + err.Error())
		}
	}

	msg := &Message{
		ID:        NewMessageID(),
		ProjectID: projectID,
		Channel:   ChannelSMS,
		Direction: DirectionInbound,
		Provider:  "native",
		From:      from,
		To:        to,
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
	if len(req.To) == 1 && req.Channel == ChannelSMS {
		parsed, err := phone.Parse(req.To[0], s.phoneMode)
		if err != nil {
			var pe *phone.Error
			if errors.As(err, &pe) {
				return NewError(ErrorCode(pe.Code), pe.Message, pe.Field)
			}
			return NewInvalidNumber("invalid phone number", "to")
		}
		_ = parsed
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

func (s *Service) defaultVerificationText(channel Channel, code string, serviceRef, serviceLabel *string) string {
	if channel == ChannelEmail {
		return "Your verification code is " + code
	}
	if serviceLabel != nil && *serviceLabel != "" {
		return "Your " + *serviceLabel + " verification code is: " + code
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

// NormalizePhone returns the E.164 normalized form of a phone number,
// or the trimmed input if it cannot be parsed.
func NormalizePhone(number string) string {
	return phone.Normalize(number)
}

func generateCode(length int) string {
	const digits = "0123456789"
	code := make([]byte, length)
	maxDigit := big.NewInt(10)
	for i := range code {
		n, err := cryptoRand.Int(cryptoRand.Reader, maxDigit)
		if err != nil {
			code[i] = digits[0]
			continue
		}
		code[i] = digits[n.Int64()]
	}
	return string(code)
}

func isSTOPKeyword(body string) bool {
	switch strings.TrimSpace(strings.ToLower(body)) {
	case "stop", "stopall", "unsubscribe", "cancel", "end", "quit":
		return true
	}
	return false
}

func isSTARTKeyword(body string) bool {
	switch strings.TrimSpace(strings.ToLower(body)) {
	case "start", "yes", "unstop":
		return true
	}
	return false
}

var (
	ErrMessageNotFound = errors.New("message not found")
	ErrBatchNotFound   = errors.New("batch not found")
)

func (s *Service) Shutdown() {
	s.lifecycle.Stop()
}

func (s *Service) Store() Store {
	return s.store
}

func (s *Service) BlobStore() BlobStore {
	return s.blobStore
}

func (s *Service) Bus() Bus {
	return s.bus
}

func (s *Service) Resolver() ProjectResolver {
	return s.resolver
}
