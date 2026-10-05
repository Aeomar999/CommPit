package core

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrCodeValidationError      ErrorCode = "validation_error"
	ErrCodeInvalidNumber        ErrorCode = "invalid_number"
	ErrCodeInvalidSender        ErrorCode = "invalid_sender"
	ErrCodeUnroutable           ErrorCode = "unroutable"
	ErrCodeNotSMSCapable        ErrorCode = "not_sms_capable"
	ErrCodeInvalidAddress       ErrorCode = "invalid_address"
	ErrCodeUnsubscribed         ErrorCode = "unsubscribed"
	ErrCodeRateLimited          ErrorCode = "rate_limited"
	ErrCodeProviderUnavailable  ErrorCode = "provider_unavailable"
	ErrCodeVerificationNotFound ErrorCode = "verification_not_found"
	ErrCodeMaxAttempts          ErrorCode = "max_attempts"
	ErrCodeUnauthorized         ErrorCode = "unauthorized"
	ErrCodeNotFound             ErrorCode = "not_found"
	ErrCodeInternal             ErrorCode = "internal"
)

type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Field   string    `json:"field,omitempty"`
}

func (e *Error) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

func (e *Error) HTTPStatus() int {
	switch e.Code {
	case ErrCodeValidationError, ErrCodeInvalidNumber, ErrCodeInvalidSender, ErrCodeUnroutable, ErrCodeNotSMSCapable, ErrCodeInvalidAddress, ErrCodeUnsubscribed:
		return 400
	case ErrCodeUnauthorized:
		return 401
	case ErrCodeRateLimited, ErrCodeMaxAttempts:
		return 429
	case ErrCodeNotFound, ErrCodeVerificationNotFound:
		return 404
	case ErrCodeProviderUnavailable:
		return 503
	case ErrCodeInternal:
		return 500
	default:
		return 500
	}
}

func NewError(code ErrorCode, message, field string) *Error {
	return &Error{Code: code, Message: message, Field: field}
}

func NewValidationError(message, field string) *Error {
	return &Error{Code: ErrCodeValidationError, Message: message, Field: field}
}

func NewInvalidNumber(message, field string) *Error {
	return &Error{Code: ErrCodeInvalidNumber, Message: message, Field: field}
}

func NewInvalidSender(message, field string) *Error {
	return &Error{Code: ErrCodeInvalidSender, Message: message, Field: field}
}

func NewUnroutable(message, field string) *Error {
	return &Error{Code: ErrCodeUnroutable, Message: message, Field: field}
}

func NewNotSMSCapable(message, field string) *Error {
	return &Error{Code: ErrCodeNotSMSCapable, Message: message, Field: field}
}

func NewInvalidAddress(message, field string) *Error {
	return &Error{Code: ErrCodeInvalidAddress, Message: message, Field: field}
}

func NewUnsubscribed(message, field string) *Error {
	return &Error{Code: ErrCodeUnsubscribed, Message: message, Field: field}
}

func NewRateLimited(message, field string) *Error {
	return &Error{Code: ErrCodeRateLimited, Message: message, Field: field}
}

func NewProviderUnavailable(message, field string) *Error {
	return &Error{Code: ErrCodeProviderUnavailable, Message: message, Field: field}
}

func NewVerificationNotFound(message, field string) *Error {
	return &Error{Code: ErrCodeVerificationNotFound, Message: message, Field: field}
}

func NewMaxAttempts(message, field string) *Error {
	return &Error{Code: ErrCodeMaxAttempts, Message: message, Field: field}
}

func NewUnauthorized(message string) *Error {
	return &Error{Code: ErrCodeUnauthorized, Message: message, Field: ""}
}

func NewNotFound(message, field string) *Error {
	return &Error{Code: ErrCodeNotFound, Message: message, Field: field}
}

func NewInternal(message string) *Error {
	return &Error{Code: ErrCodeInternal, Message: message, Field: ""}
}

func IsError(err error, code ErrorCode) bool {
	var ce *Error
	if errors.As(err, &ce) {
		if code == "" {
			return true
		}
		return ce.Code == code
	}
	return false
}
