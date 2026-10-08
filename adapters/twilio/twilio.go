// Package twilio implements a Twilio-compatible Messages API as a thin
// translator over the native core service (spec §7.4).
//
// Mounted at /twilio, it accepts the same form-encoded requests the
// official Twilio SDKs send (rewritten to the local server via the
// redirect snippets in examples/) and stores the provider-format message
// SID in the core message's ProviderRef. Status callbacks are accepted
// and stored; delivery happens in M3 via the webhooks worker.
package twilio

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/go-chi/chi/v5"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// APIVersion is the Twilio API version this adapter emulates.
const APIVersion = "2010-04-01"

// Adapter translates Twilio Messages API requests into core service calls.
type Adapter struct {
	service *core.Service
}

// New wires a Twilio adapter around the core service. The service must be
// non-nil for request handling; WriteError works without one.
func New(service *core.Service) *Adapter {
	return &Adapter{service: service}
}

// Name returns the canonical adapter identifier.
func (a *Adapter) Name() string {
	return "twilio"
}

// Routes registers the Messages endpoints. The router is mounted at
// /twilio by the composition root, so paths here are Twilio-relative.
func (a *Adapter) Routes(r chi.Router) {
	r.Route("/2010-04-01/Accounts/{AccountSid}", func(r chi.Router) {
		r.Post("/Messages.json", a.createMessage)
		r.Get("/Messages.json", a.listMessages)
		r.Get("/Messages/{MessageSid}.json", a.fetchMessage)
	})
}

// Extractor resolves the caller's credential from the Account SID in the
// path, falling back to the Basic-auth username the SDK always sends.
func Extractor() adapterkit.CredentialExtractor {
	return adapterkit.FirstOf(
		adapterkit.PathExtractor("AccountSid"),
		adapterkit.BasicAuthExtractor(),
	)
}

// newMessageSID generates a Twilio-format message SID: SM + 32 hex chars.
func newMessageSID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("twilio: generate sid: %w", err)
	}
	return "SM" + hex.EncodeToString(buf[:]), nil
}
