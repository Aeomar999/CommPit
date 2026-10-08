package termii

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Aeomar999/CommPit/core"
)

// termiiError is the provider error payload. Termii publishes no error
// code catalog, so the canonical code travels as a string; see
// docs/fidelity.md.
type termiiError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// WriteError translates a canonical core error into Termii's error format.
func (a *Adapter) WriteError(w http.ResponseWriter, err *core.Error) {
	if err == nil {
		err = core.NewInternal("internal error")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus())
	_ = json.NewEncoder(w).Encode(termiiError{
		Message: err.Message,
		Code:    string(err.Code),
	})
}

// writeServiceError converts a service-layer error (canonical or
// unexpected) into Termii's error format.
func (a *Adapter) writeServiceError(w http.ResponseWriter, err error) {
	var ce *core.Error
	if !errors.As(err, &ce) {
		ce = core.NewInternal("internal error")
	}
	a.WriteError(w, ce)
}
