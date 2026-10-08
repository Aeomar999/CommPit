// Package termii implements Termii-compatible SMS APIs as thin translators
// over the native core service (spec §7.4).
//
// Mounted at /termii, it accepts the same JSON requests Termii integrations
// send (via a TERMII_BASE_URL override) and stores numeric message IDs in
// the core message's ProviderRef. Token endpoints arrive in M2-08.
package termii

import (
	cryptoRand "crypto/rand"
	"fmt"
	"math/big"

	"github.com/go-chi/chi/v5"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// senderAllowListKey is the project settings key holding the permitted
// sender IDs. An absent or empty list accepts every sender.
const senderAllowListKey = "termii.sender_allowlist"

// Recipient caps per endpoint.
const (
	maxSingleRecipients = 100
	maxBulkRecipients   = 10000
)

// maxCredentialBodyBytes bounds credential extraction. Bulk batches carry
// up to 10,000 recipients (~200 KB of JSON), far beyond the 64 KB default,
// while request logging still caps stored bodies at 64 KB.
const maxCredentialBodyBytes = 1024 * 1024

// Adapter translates Termii SMS API requests into core service calls.
type Adapter struct {
	service *core.Service
}

// New wires a Termii adapter around the core service. The service must be
// non-nil for request handling; WriteError works without one.
func New(service *core.Service) *Adapter {
	return &Adapter{service: service}
}

// Name returns the canonical adapter identifier.
func (a *Adapter) Name() string {
	return "termii"
}

// Routes registers the SMS endpoints. The router is mounted at /termii by
// the composition root, so paths here are Termii-relative.
func (a *Adapter) Routes(r chi.Router) {
	r.Post("/api/sms/send", a.sendHandler(maxSingleRecipients, false))
	r.Post("/api/sms/send/bulk", a.sendHandler(maxBulkRecipients, true))
	r.Post("/api/sms/number/send", a.sendHandler(maxSingleRecipients, false))
}

// Extractor resolves the caller's credential from the api_key in the JSON
// request body.
func Extractor() adapterkit.CredentialExtractor {
	return adapterkit.JSONBodyKeyExtractorWithLimit("api_key", maxCredentialBodyBytes)
}

// newTermiiID generates a numeric Termii-format message ID.
func newTermiiID() (int64, error) {
	n, err := cryptoRand.Int(cryptoRand.Reader, big.NewInt(10000000000))
	if err != nil {
		return 0, fmt.Errorf("termii: generate id: %w", err)
	}
	return n.Int64() + 1, nil
}
