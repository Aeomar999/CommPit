package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Aeomar999/CommPit/config"
)

func SecurityMiddleware(cfg *config.SecurityConfig) func(http.Handler) http.Handler {
	allowedHosts := make(map[string]bool)
	for _, h := range cfg.AllowedHosts {
		allowedHosts[strings.ToLower(h)] = true
	}

	var uiAuthUser, uiAuthPass string
	if cfg.UIAuth != "" {
		parts := strings.SplitN(cfg.UIAuth, ":", 2)
		if len(parts) == 2 {
			uiAuthUser = parts[0]
			uiAuthPass = parts[1]
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Host header allow-list (DNS rebinding protection)
			if len(allowedHosts) > 0 {
				host := strings.ToLower(r.Host)
				if idx := strings.Index(host, ":"); idx != -1 {
					host = host[:idx]
				}
				if !allowedHosts[host] {
					writeJSONError(w, 421, "misdirected_request", "Host not allowed")
					return
				}
			}

			// Require X-Mocksms header for API routes (skip for healthz and UI)
			if cfg.RequireXMocksms && !isExemptPath(r.URL.Path) {
				if r.Header.Get("X-Mocksms") == "" {
					writeJSONError(w, 400, "missing_header", "X-Mocksms header required")
					return
				}
			}

			// UI basic auth
			if uiAuthUser != "" && isUIPath(r.URL.Path) {
				auth := r.Header.Get("Authorization")
				if !checkBasicAuth(auth, uiAuthUser, uiAuthPass) {
					w.Header().Set("WWW-Authenticate", `Basic realm="mocksms"`)
					writeJSONError(w, 401, "unauthorized", "Authentication required")
					return
				}
			}

			// Non-loopback warning (logged, not enforced)
			if isNonLoopback(r) {
				// Could log a warning here
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), securityConfigKey{}, cfg)))
		})
	}
}

func isExemptPath(path string) bool {
	return path == "/healthz" || strings.HasPrefix(path, "/api/v1/events") || strings.HasPrefix(path, "/api/v1/messages/wait")
}

func isUIPath(path string) bool {
	return path == "/" || strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/favicon")
}

func isNonLoopback(r *http.Request) bool {
	host := r.Host
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host != "127.0.0.1" && host != "localhost" && host != "::1"
}

func checkBasicAuth(auth, user, pass string) bool {
	if auth == "" || len(auth) < 6 || auth[:6] != "Basic " {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		return false
	}
	creds := string(decoded)
	parts := strings.SplitN(creds, ":", 2)
	if len(parts) != 2 {
		return false
	}
	return parts[0] == user && parts[1] == pass
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

type securityConfigKey struct{}
