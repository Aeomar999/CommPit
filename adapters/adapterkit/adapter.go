package adapterkit

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Aeomar999/CommPit/core"
)

// Adapter defines the contract that each provider adapter (Twilio, Termii, etc.)
// must satisfy. Adapters translate external provider protocols and payloads
// into core domain actions without directly accessing underlying stores.
type Adapter interface {
	// Name returns the canonical identifier of the adapter (e.g. "twilio", "termii").
	Name() string

	// Routes registers the provider-specific HTTP endpoints onto the given router.
	Routes(router chi.Router)

	// WriteError translates a canonical core error into the provider's specific error format
	// and writes the corresponding HTTP response.
	WriteError(writer http.ResponseWriter, coreErr *core.Error)
}
