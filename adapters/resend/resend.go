// Package resend implements a Resend-compatible Email API as a thin
// translator over the native core service.
//
// Mounted at /resend, it accepts the same JSON requests Resend integrations
// send (via a base-URL override) and stores the provider-format message id
// in the core message's ProviderRef.
package resend

import (
	"github.com/go-chi/chi/v5"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// Adapter translates Resend Email API requests into core service calls.
type Adapter struct {
	service *core.Service
}

// New wires a Resend adapter around the core service.
func New(service *core.Service) *Adapter {
	return &Adapter{service: service}
}

// Name returns the canonical adapter identifier.
func (a *Adapter) Name() string {
	return "resend"
}

// Routes registers the Email endpoint.
func (a *Adapter) Routes(r chi.Router) {
	r.Post("/emails", a.sendEmail)
}

// Extractor resolves the caller's credential from the Authorization header
// (Bearer <api_key>).
func Extractor() adapterkit.CredentialExtractor {
	return adapterkit.BearerTokenExtractor()
}
