package vonage

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Aeomar999/CommPit/core"
)

// WriteError translates a canonical core error into Vonage's error format.
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
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": status,
		"error":  err.Message,
		"code":   err.Code,
	})
}

// writeServiceError converts a service-layer error into Vonage's format.
func (a *Adapter) writeServiceError(w http.ResponseWriter, err error) {
	var ce *core.Error
	if !errors.As(err, &ce) {
		ce = core.NewInternal("internal error")
	}
	a.WriteError(w, ce)
}
