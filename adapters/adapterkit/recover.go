package adapterkit

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/Aeomar999/CommPit/core"
)

// Recoverer returns a middleware that catches panics originating within adapter
// handlers, logs the occurrence and stack trace, and writes a canonical internal error
// formatted according to the provider adapter's error specification.
func Recoverer(adapter Adapter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			defer func() {
				if recoveredVal := recover(); recoveredVal != nil {
					if err, ok := recoveredVal.(error); ok && errors.Is(err, http.ErrAbortHandler) {
						panic(recoveredVal)
					}

					slog.Error("panic recovered in adapter handler",
						"adapter", adapter.Name(),
						"panic", recoveredVal,
						"path", req.URL.Path,
						"stack", string(debug.Stack()),
					)

					adapter.WriteError(writer, core.NewInternal("internal server error"))
				}
			}()

			next.ServeHTTP(writer, req)
		})
	}
}
