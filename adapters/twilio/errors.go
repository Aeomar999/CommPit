package twilio

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Aeomar999/CommPit/core"
)

// twilioError is the provider error payload. Codes are integers and every
// response carries a link to the canonical error documentation.
type twilioError struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	MoreInfo string `json:"more_info"`
	Status   int    `json:"status"`
}

// twilioCodeFor maps a canonical core error to Twilio's numeric error code
// (spec §7.2). Field-specific validation codes are the most likely mapping;
// see docs/fidelity.md for what is still unverified against Twilio's docs.
func twilioCodeFor(err *core.Error) int {
	switch err.Code {
	case core.ErrCodeInvalidNumber:
		return 21211
	case core.ErrCodeInvalidSender:
		return 21212
	case core.ErrCodeUnroutable:
		return 21612
	case core.ErrCodeNotSMSCapable:
		return 21614
	case core.ErrCodeUnsubscribed:
		return 21610
	case core.ErrCodeRateLimited:
		return 20429
	case core.ErrCodeProviderUnavailable:
		return 20503
	case core.ErrCodeVerificationNotFound:
		return 20404
	case core.ErrCodeMaxAttempts:
		return 60202
	case core.ErrCodeUnauthorized:
		return 20003
	case core.ErrCodeNotFound:
		return 20404
	case core.ErrCodeValidationError:
		return validationTwilioCode(err.Field)
	default:
		return 20500
	}
}

// validationTwilioCode picks the field-specific Twilio code for a
// validation_error. The 2160x assignments are best-effort and unverified.
func validationTwilioCode(field string) int {
	switch strings.ToLower(field) {
	case "from":
		return 21606
	case "body":
		return 21602
	default:
		return 21604
	}
}

// WriteError translates a canonical core error into Twilio's error format.
func (a *Adapter) WriteError(w http.ResponseWriter, err *core.Error) {
	if err == nil {
		err = core.NewInternal("internal error")
	}
	code := twilioCodeFor(err)
	status := err.HTTPStatus()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(twilioError{
		Code:     code,
		Message:  err.Message,
		MoreInfo: "https://www.twilio.com/docs/errors/" + strconv.Itoa(code),
		Status:   status,
	})
}

// writeServiceError converts a service-layer error (canonical or
// unexpected) into Twilio's error format.
func (a *Adapter) writeServiceError(w http.ResponseWriter, err error) {
	var ce *core.Error
	if !errors.As(err, &ce) {
		ce = core.NewInternal("internal error")
	}
	a.WriteError(w, ce)
}
