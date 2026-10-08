package adapterkit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Aeomar999/CommPit/core"
)

// CredentialExtractor inspects an incoming HTTP request to extract a provider credential key.
// Returns (key, true, nil) when a credential is found, ("", false, nil) when none is present,
// or an error if extraction failed.
type CredentialExtractor func(req *http.Request) (key string, ok bool, err error)

// BasicAuthExtractor extracts the username portion of an HTTP Basic Auth header.
func BasicAuthExtractor() CredentialExtractor {
	return func(req *http.Request) (string, bool, error) {
		username, _, hasAuth := req.BasicAuth()
		if hasAuth && username != "" {
			return username, true, nil
		}
		return "", false, nil
	}
}

// HeaderExtractor extracts a credential key from the specified HTTP header.
func HeaderExtractor(headerName string) CredentialExtractor {
	return func(req *http.Request) (string, bool, error) {
		val := req.Header.Get(headerName)
		if val != "" {
			return val, true, nil
		}
		return "", false, nil
	}
}

// BearerTokenExtractor extracts the token part of an Authorization: Bearer <token> header.
func BearerTokenExtractor() CredentialExtractor {
	return func(req *http.Request) (string, bool, error) {
		authHeader := req.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if token != "" {
				return token, true, nil
			}
		}
		return "", false, nil
	}
}

// QueryExtractor extracts a credential key from the request's URL query string.
func QueryExtractor(paramName string) CredentialExtractor {
	return func(req *http.Request) (string, bool, error) {
		val := req.URL.Query().Get(paramName)
		if val != "" {
			return val, true, nil
		}
		return "", false, nil
	}
}

// PathExtractor extracts a credential key from a Chi route path parameter.
func PathExtractor(paramName string) CredentialExtractor {
	return func(req *http.Request) (string, bool, error) {
		val := chi.URLParam(req, paramName)
		if val != "" {
			return val, true, nil
		}
		return "", false, nil
	}
}

// JSONBodyKeyExtractor peeks at a top-level field in a JSON request body
// and restores the request body for downstream handlers. It reads at most
// 64 KB; larger bodies fail closed (no credential found).
func JSONBodyKeyExtractor(fieldName string) CredentialExtractor {
	return JSONBodyKeyExtractorWithLimit(fieldName, 64*1024)
}

// JSONBodyKeyExtractorWithLimit behaves like JSONBodyKeyExtractor with a
// caller-chosen read cap. Bulk endpoints (e.g. Termii's 10,000-recipient
// batch) legitimately exceed 64 KB, so those adapters pass a larger cap.
// Bodies beyond the cap fail closed.
func JSONBodyKeyExtractorWithLimit(fieldName string, maxBytes int) CredentialExtractor {
	return func(req *http.Request) (string, bool, error) {
		if req.Body == nil {
			return "", false, nil
		}

		bodyBytes, err := io.ReadAll(io.LimitReader(req.Body, int64(maxBytes)))
		if err != nil {
			return "", false, err
		}

		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		if len(bodyBytes) == 0 {
			return "", false, nil
		}

		var payload map[string]any
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			return "", false, nil
		}

		if val, exists := payload[fieldName]; exists {
			if strVal, isString := val.(string); isString && strVal != "" {
				return strVal, true, nil
			}
		}

		return "", false, nil
	}
}

// FirstOf attempts a slice of extractors in sequence, returning the first successful match.
func FirstOf(extractors ...CredentialExtractor) CredentialExtractor {
	return func(req *http.Request) (string, bool, error) {
		for _, extractor := range extractors {
			key, ok, err := extractor(req)
			if err != nil {
				return "", false, err
			}
			if ok && key != "" {
				return key, true, nil
			}
		}
		return "", false, nil
	}
}

type resolverConfig struct {
	defaultProject string
	required       bool
}

// ResolverOption configures the credential resolution middleware.
type ResolverOption func(*resolverConfig)

// WithDefaultProject specifies a fallback project ID when no credentials are provided.
func WithDefaultProject(projectID string) ResolverOption {
	return func(cfg *resolverConfig) {
		cfg.defaultProject = projectID
	}
}

// WithRequired enforces that requests must supply credentials.
func WithRequired(required bool) ResolverOption {
	return func(cfg *resolverConfig) {
		cfg.required = required
	}
}

// ResolveProject returns a middleware that extracts caller credentials, resolves the
// corresponding project via core.ProjectResolver, and annotates the context.
func ResolveProject(
	adapter Adapter,
	resolver core.ProjectResolver,
	extractor CredentialExtractor,
	opts ...ResolverOption,
) func(http.Handler) http.Handler {
	cfg := resolverConfig{
		defaultProject: "default",
		required:       false,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			ctx := EnsureCarrier(req.Context())
			req = req.WithContext(ctx)

			key, ok, err := extractor(req)
			if err != nil {
				adapter.WriteError(writer, core.NewValidationError("failed to extract credentials: "+err.Error(), "credential"))
				return
			}

			if ok && key != "" {
				projectID, resolveErr := resolver.Resolve(req.Context(), adapter.Name(), key)
				if resolveErr != nil {
					adapter.WriteError(writer, core.NewInternal("failed to resolve project: "+resolveErr.Error()))
					return
				}
				WithProjectID(ctx, projectID)
				WithCredential(ctx, key)
			} else {
				if cfg.required {
					adapter.WriteError(writer, core.NewUnauthorized("credentials required"))
					return
				}
				if cfg.defaultProject != "" {
					WithProjectID(ctx, cfg.defaultProject)
				}
			}

			next.ServeHTTP(writer, req)
		})
	}
}
