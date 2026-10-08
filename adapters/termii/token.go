package termii

import (
	cryptoRand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// Default OTP settings when the caller omits them.
const (
	defaultPinLength      = 6
	defaultPinPlaceholder = "< 1234 >"
)

// otpSendRequest mirrors Termii's otp/send JSON body.
type otpSendRequest struct {
	To             json.RawMessage `json:"to"`
	From           string          `json:"from"`
	MessageText    string          `json:"message_text"`
	MessageType    string          `json:"message_type"`
	Channel        string          `json:"channel"`
	PinAttempts    int             `json:"pin_attempts"`
	PinTimeToLive  int             `json:"pin_time_to_live"`
	PinLength      int             `json:"pin_length"`
	PinPlaceholder string          `json:"pin_placeholder"`
}

// otpSendResponse mirrors Termii's otp/send response.
type otpSendResponse struct {
	PinID     string `json:"pinId"`
	To        string `json:"to"`
	SmsStatus string `json:"smsStatus"`
}

// otpVerifyRequest mirrors Termii's otp/verify JSON body.
type otpVerifyRequest struct {
	PinID string `json:"pin_id"`
	Pin   string `json:"pin"`
}

// otpVerifyResponse mirrors Termii's otp/verify response.
type otpVerifyResponse struct {
	PinID    string `json:"pinId"`
	Verified bool   `json:"verified"`
	Msisdn   string `json:"msisdn"`
}

// otpGenerateRequest mirrors Termii's otp/generate JSON body.
type otpGenerateRequest struct {
	PinType       string `json:"pin_type"`
	PinAttempts   int    `json:"pin_attempts"`
	PinTimeToLive int    `json:"pin_time_to_live"`
	PinLength     int    `json:"pin_length"`
}

// otpGenerateResponse carries an in-app OTP back to the caller. Nothing is
// sent and no verification entity is stored.
type otpGenerateResponse struct {
	Pin string `json:"pin"`
}

// emailOtpRequest mirrors Termii's email/otp/send JSON body. The caller
// supplies the code; the adapter stores it verbatim on the verification.
type emailOtpRequest struct {
	EmailAddress         string `json:"email_address"`
	Code                 string `json:"code"`
	EmailConfigurationID string `json:"email_configuration_id"`
}

// emailOtpResponse acknowledges an email OTP send.
type emailOtpResponse struct {
	MessageID int64  `json:"message_id"`
	Message   string `json:"message"`
	Email     string `json:"email"`
}

// newPinID generates a UUIDv4 pin identifier without a new dependency.
func newPinID() (string, error) {
	var raw [16]byte
	if _, err := cryptoRand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("termii: generate pin id: %w", err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	hexed := hex.EncodeToString(raw[:])
	return hexed[0:8] + "-" + hexed[8:12] + "-" + hexed[12:16] + "-" + hexed[16:20] + "-" + hexed[20:32], nil
}

// generatePin creates a numeric OTP of the requested length.
func generatePin(length int) (string, error) {
	digits := make([]byte, length)
	maxDigit := big.NewInt(10)
	for i := range digits {
		n, err := cryptoRand.Int(cryptoRand.Reader, maxDigit)
		if err != nil {
			return "", fmt.Errorf("termii: generate pin: %w", err)
		}
		digits[i] = byte('0' + n.Int64())
	}
	return string(digits), nil
}

// decodeBody decodes a JSON request body or reports a validation error.
func (a *Adapter) decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		a.WriteError(w, core.NewValidationError("invalid JSON", ""))
		return false
	}
	return true
}

// otpSend handles POST /api/sms/otp/send.
func (a *Adapter) otpSend(w http.ResponseWriter, r *http.Request) {
	var req otpSendRequest
	if !a.decodeBody(w, r, &req) {
		return
	}

	recipients, verr := parseRecipients(req.To)
	if verr != nil {
		a.WriteError(w, verr)
		return
	}
	if len(recipients) != 1 {
		a.WriteError(w, core.NewValidationError("exactly one recipient is required", "to"))
		return
	}
	if strings.TrimSpace(req.From) == "" {
		a.WriteError(w, core.NewValidationError("sender is required", "from"))
		return
	}
	pinLength := req.PinLength
	if pinLength == 0 {
		pinLength = defaultPinLength
	}
	if pinLength < 4 || pinLength > 10 {
		a.WriteError(w, core.NewValidationError("pin_length must be between 4 and 10", "pin_length"))
		return
	}

	projectID := adapterkit.ProjectID(r)
	if verr := a.checkSender(r.Context(), projectID, req.From); verr != nil {
		a.WriteError(w, verr)
		return
	}

	// The adapter mints the PIN up front so a message_text template can be
	// rendered in a single write; the PIN travels as an explicit custom
	// code, which also keeps --otp-code out of provider flows.
	pin, err := generatePin(pinLength)
	if err != nil {
		a.WriteError(w, core.NewInternal("failed to generate pin"))
		return
	}
	var bodyText *string
	if strings.TrimSpace(req.MessageText) != "" {
		placeholder := req.PinPlaceholder
		if placeholder == "" {
			placeholder = defaultPinPlaceholder
		}
		text := strings.ReplaceAll(req.MessageText, placeholder, pin)
		if !strings.Contains(req.MessageText, placeholder) {
			text += " " + pin
		}
		bodyText = &text
	}

	pinID, err := newPinID()
	if err != nil {
		a.WriteError(w, core.NewInternal("failed to allocate pin id"))
		return
	}

	ttlSeconds := 0
	if req.PinTimeToLive > 0 {
		ttlSeconds = req.PinTimeToLive * 60
	}
	resp, err := a.service.StartVerification(r.Context(), projectID, core.VerificationRequest{
		To:          recipients[0],
		Channel:     core.ChannelSMS,
		TTLSeconds:  ttlSeconds,
		MaxAttempts: req.PinAttempts,
		Provider:    a.Name(),
		ProviderRef: pinID,
		CustomCode:  &pin,
		BodyText:    bodyText,
	})
	if err != nil {
		a.writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(otpSendResponse{
		PinID:     pinID,
		To:        resp.Verification.To,
		SmsStatus: "Message Sent",
	})
}

// otpVerify handles POST /api/sms/otp/verify.
func (a *Adapter) otpVerify(w http.ResponseWriter, r *http.Request) {
	var req otpVerifyRequest
	if !a.decodeBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.PinID) == "" {
		a.WriteError(w, core.NewValidationError("pin_id is required", "pin_id"))
		return
	}
	if strings.TrimSpace(req.Pin) == "" {
		a.WriteError(w, core.NewValidationError("pin is required", "pin"))
		return
	}

	projectID := adapterkit.ProjectID(r)
	v, err := a.service.Store().GetVerificationByProviderRef(r.Context(), projectID, req.PinID)
	if err != nil {
		a.WriteError(w, core.NewVerificationNotFound("pin not found", "pin_id"))
		return
	}

	res, err := a.service.CheckVerification(r.Context(), projectID, v.ID, core.CheckVerificationRequest{Code: req.Pin})
	if err != nil {
		a.writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(otpVerifyResponse{
		PinID:    req.PinID,
		Verified: res.Valid,
		Msisdn:   v.To,
	})
}

// otpGenerate handles POST /api/sms/otp/generate. The PIN is returned for
// in-app verification; nothing is sent and nothing is stored.
func (a *Adapter) otpGenerate(w http.ResponseWriter, r *http.Request) {
	var req otpGenerateRequest
	if !a.decodeBody(w, r, &req) {
		return
	}

	pinType := strings.ToUpper(strings.TrimSpace(req.PinType))
	if pinType == "" {
		pinType = "NUMERIC"
	}
	if pinType != "NUMERIC" {
		a.WriteError(w, core.NewValidationError("only NUMERIC pins are supported", "pin_type"))
		return
	}
	pinLength := req.PinLength
	if pinLength == 0 {
		pinLength = defaultPinLength
	}
	if pinLength < 4 || pinLength > 10 {
		a.WriteError(w, core.NewValidationError("pin_length must be between 4 and 10", "pin_length"))
		return
	}

	pin, err := generatePin(pinLength)
	if err != nil {
		a.WriteError(w, core.NewInternal("failed to generate pin"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(otpGenerateResponse{Pin: pin})
}

// emailOtpSend handles POST /api/email/otp/send. The caller supplies the
// code, which is stored verbatim on an email-channel verification.
func (a *Adapter) emailOtpSend(w http.ResponseWriter, r *http.Request) {
	var req emailOtpRequest
	if !a.decodeBody(w, r, &req) {
		return
	}

	email := strings.TrimSpace(req.EmailAddress)
	if _, err := mail.ParseAddress(email); err != nil || email == "" {
		a.WriteError(w, core.NewValidationError("valid email_address is required", "email_address"))
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		a.WriteError(w, core.NewValidationError("code is required", "code"))
		return
	}

	id, err := newTermiiID()
	if err != nil {
		a.WriteError(w, core.NewInternal("failed to allocate message id"))
		return
	}

	projectID := adapterkit.ProjectID(r)
	custom := strings.TrimSpace(req.Code)
	if _, err := a.service.StartVerification(r.Context(), projectID, core.VerificationRequest{
		To:          email,
		Channel:     core.ChannelEmail,
		Provider:    a.Name(),
		ProviderRef: strconv.FormatInt(id, 10),
		CustomCode:  &custom,
	}); err != nil {
		a.writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(emailOtpResponse{
		MessageID: id,
		Message:   "Successfully Sent",
		Email:     email,
	})
}
