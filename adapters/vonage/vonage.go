// Package vonage implements a Vonage-compatible SMS and Verify API as a
// thin translator over the native core service.
package vonage

import (
	"github.com/go-chi/chi/v5"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// Adapter translates Vonage SMS/Verify API requests into core service calls.
type Adapter struct {
	service *core.Service
}

// New wires a Vonage adapter around the core service.
func New(service *core.Service) *Adapter {
	return &Adapter{service: service}
}

// Name returns the canonical adapter identifier.
func (a *Adapter) Name() string {
	return "vonage"
}

// Routes registers the SMS and Verify endpoints.
func (a *Adapter) Routes(r chi.Router) {
	r.Post("/sms/json", a.sendSMS)
	r.Post("/v2/verify", a.startVerify)
	r.Post("/v2/verify/{request_id}", a.checkVerify)
}

// Extractor resolves the caller's credential from api_key/api_secret in the
// JSON body (or query string).
func Extractor() adapterkit.CredentialExtractor {
	return adapterkit.FirstOf(
		adapterkit.JSONBodyKeyExtractor("api_key"),
		adapterkit.QueryExtractor("api_key"),
	)
}
