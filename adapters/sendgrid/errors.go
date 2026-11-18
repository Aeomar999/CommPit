package sendgrid

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Aeomar999/CommPit/core"
)

// sendGridError is one entry in SendGrid's error array.
type sendGridError struct {
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
	Help    string `json:"help,omitempty"`
}

// sendGridErrorResponse wraps the errors array SendGrid returns on failure.
type sendGridErrorResponse struct {
	Errors []sendGridError `json:"errors"`
}

// WriteError translates a canonical core error into SendGrid's error format.
func (a *Adapter) WriteError(w http.ResponseWriter, err *core.Error) {
	if err == nil {
		err = core.NewInternal("internal error")
	}
	status := err.HTTPStatus()
	if status < 400 {
		status = 400
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(sendGridErrorResponse{
		Errors: []sendGridError{{Message: err.Message, Field: err.Field}},
	})
}

// writeServiceError converts a service-layer error (canonical or unexpected)
// into SendGrid's error format.
func (a *Adapter) writeServiceError(w http.ResponseWriter, err error) {
	var ce *core.Error
	if !errors.As(err, &ce) {
		ce = core.NewInternal("internal error")
	}
	a.WriteError(w, ce)
}
