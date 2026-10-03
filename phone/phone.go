package phone

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nyaruka/phonenumbers"
)

type Mode int

const (
	ModeValid Mode = iota
	ModePossible
	ModeOff
)

var (
	ErrValidation    = errors.New("validation error")
	ErrInvalidNumber = errors.New("invalid number")
)

type Error struct {
	Code    string
	Message string
	Field   string
}

func (e *Error) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

func newError(code, message, field string) *Error {
	return &Error{Code: code, Message: message, Field: field}
}

type Parsed struct {
	E164     string
	Country  string
	National string
	Valid    bool
	Possible bool
}

func Parse(input string, mode Mode) (*Parsed, error) {
	input = strings.TrimSpace(input)

	switch mode {
	case ModeOff:
		return &Parsed{
			E164:     input,
			Country:  "",
			National: "",
			Valid:    false,
			Possible: false,
		}, nil
	case ModeValid, ModePossible:
		if input == "" {
			return nil, newError("validation_error", "phone number is required", "to")
		}
		if !strings.HasPrefix(input, "+") {
			return nil, newError("invalid_number", "phone number must be in E.164 format (starting with +)", "to")
		}
	}

	num, err := phonenumbers.Parse(input, "")
	if err != nil {
		return nil, newError("invalid_number", "failed to parse phone number", "to")
	}

	possible := phonenumbers.IsPossibleNumber(num)
	valid := phonenumbers.IsValidNumber(num)

	if mode == ModeValid && !valid {
		return nil, newError("invalid_number", "phone number is not valid", "to")
	}
	if mode == ModePossible && !possible {
		return nil, newError("invalid_number", "phone number is not possible", "to")
	}

	e164 := phonenumbers.Format(num, phonenumbers.E164)
	country := phonenumbers.GetRegionCodeForNumber(num)
	national := phonenumbers.Format(num, phonenumbers.NATIONAL)
	national = strings.ReplaceAll(national, " ", "")
	national = strings.ReplaceAll(national, "-", "")
	national = strings.ReplaceAll(national, "(", "")
	national = strings.ReplaceAll(national, ")", "")

	return &Parsed{
		E164:     e164,
		Country:  country,
		National: national,
		Valid:    valid,
		Possible: possible,
	}, nil
}
