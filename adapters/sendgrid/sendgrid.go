// Package sendgrid implements a SendGrid-compatible Mail Send API as a thin
// translator over the native core service (spec §7.4).
//
// Mounted at /sendgrid, it accepts the same JSON requests SendGrid integrations
// send (via a base-URL override) and stores the provider-format message id in
// the core message's ProviderRef.
package sendgrid

import (
	"github.com/go-chi/chi/v5"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// Adapter translates SendGrid Mail Send API requests into core service calls.
type Adapter struct {
	service *core.Service
}

// New wires a SendGrid adapter around the core service. The service must be
// non-nil for request handling; WriteError works without one.
func New(service *core.Service) *Adapter {
	return &Adapter{service: service}
}

// Name returns the canonical adapter identifier.
func (a *Adapter) Name() string {
	return "sendgrid"
}

// Routes registers the Mail Send endpoint. The router is mounted at
// /sendgrid by the composition root, so paths here are SendGrid-relative.
func (a *Adapter) Routes(r chi.Router) {
	r.Post("/v3/mail/send", a.sendMail)
}

// Extractor resolves the caller's credential from the Authorization header
// (Bearer <api_key>) or the X-Api-Key header.
func Extractor() adapterkit.CredentialExtractor {
	return adapterkit.FirstOf(
		adapterkit.BearerTokenExtractor(),
		adapterkit.HeaderExtractor("X-Api-Key"),
	)
}
