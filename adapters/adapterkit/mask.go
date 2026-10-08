package adapterkit

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
)

// MaskKey redacts a sensitive token or key string. Short keys (<= 8 characters)
// are replaced with "****". Longer keys preserve prefix and suffix segments.
func MaskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// MaskHeaders creates a copy of the request header map with sensitive values redacted.
func MaskHeaders(headers map[string]string) map[string]string {
	result := make(map[string]string, len(headers))
	for headerKey, headerVal := range headers {
		if isSensitiveHeader(headerKey) {
			result[headerKey] = maskHeaderValue(headerKey, headerVal)
		} else {
			result[headerKey] = headerVal
		}
	}
	return result
}

func isSensitiveHeader(name string) bool {
	lower := strings.ToLower(name)
	switch lower {
	case "authorization", "proxy-authorization", "cookie", "set-cookie", "x-api-key", "api-key", "apikey", "x-auth-token":
		return true
	default:
		return strings.Contains(lower, "token") ||
			strings.Contains(lower, "secret") ||
			strings.Contains(lower, "password") ||
			strings.Contains(lower, "credential") ||
			strings.Contains(lower, "api-key")
	}
}

func maskHeaderValue(headerKey, headerVal string) string {
	lowerKey := strings.ToLower(headerKey)
	if lowerKey == "authorization" {
		if strings.HasPrefix(headerVal, "Basic ") {
			encodedPart := strings.TrimPrefix(headerVal, "Basic ")
			decodedBytes, err := base64.StdEncoding.DecodeString(encodedPart)
			if err == nil {
				parts := strings.SplitN(string(decodedBytes), ":", 2)
				if len(parts) == 2 {
					return "Basic " + MaskKey(parts[0]) + ":****"
				}
			}
			return "Basic ****"
		}
		if strings.HasPrefix(headerVal, "Bearer ") {
			tokenPart := strings.TrimPrefix(headerVal, "Bearer ")
			return "Bearer " + MaskKey(tokenPart)
		}
	}

	if lowerKey == "cookie" || lowerKey == "set-cookie" {
		return "[REDACTED]"
	}

	return MaskKey(headerVal)
}

// MaskBody sanitizes credentials found within JSON or form-urlencoded request/response bodies.
func MaskBody(contentType string, body []byte) []byte {
	if len(body) == 0 {
		return body
	}

	// Try JSON first
	var jsonVal any
	if err := json.Unmarshal(body, &jsonVal); err == nil {
		maskedVal := maskJSONValue(jsonVal)
		maskedBytes, err := json.Marshal(maskedVal)
		if err == nil {
			return maskedBytes
		}
	}

	// Try form urlencoded
	if strings.Contains(contentType, "application/x-www-form-urlencoded") || strings.Contains(string(body), "=") {
		values, err := url.ParseQuery(string(body))
		if err == nil && len(values) > 0 {
			for formKey, formVals := range values {
				if isSensitiveKey(formKey) {
					for i, val := range formVals {
						values[formKey][i] = MaskKey(val)
					}
				}
			}
			return []byte(values.Encode())
		}
	}

	return body
}

func maskJSONValue(val any) any {
	switch typed := val.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			if isSensitiveKey(key) {
				if strVal, ok := child.(string); ok {
					result[key] = MaskKey(strVal)
				} else {
					result[key] = "****"
				}
			} else {
				result[key] = maskJSONValue(child)
			}
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for i, child := range typed {
			result[i] = maskJSONValue(child)
		}
		return result
	default:
		return val
	}
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	switch lower {
	case "api_key", "apikey", "api-key", "token", "auth_token", "authtoken", "auth-token",
		"secret", "client_secret", "password", "credential", "private_key", "access_token":
		return true
	default:
		return strings.Contains(lower, "secret") ||
			strings.Contains(lower, "password") ||
			strings.Contains(lower, "api_key") ||
			strings.Contains(lower, "apikey") ||
			strings.HasSuffix(lower, "token")
	}
}

// MaskURL redacts sensitive query parameters embedded within a raw request URI.
func MaskURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	queryParams := parsedURL.Query()
	modified := false
	for paramKey, paramVals := range queryParams {
		if isSensitiveKey(paramKey) {
			for i, val := range paramVals {
				queryParams[paramKey][i] = MaskKey(val)
			}
			modified = true
		}
	}

	if modified {
		parsedURL.RawQuery = queryParams.Encode()
	}
	return parsedURL.String()
}
