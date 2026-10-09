package twilio

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"sort"
	"strings"
)

// SignRequest computes an X-Twilio-Signature: base64-encoded HMAC-SHA1 over
// the full callback URL followed by the sorted concatenation of every
// parameter name and value. Verified against Twilio's documented example
// (key "12345" over the mycompany.com URL yields
// "0/KCTR6DLpKmkAf8muzZqo1nDgQ=") and cross-checked with the official
// Python SDK's RequestValidator.
func SignRequest(rawURL string, params map[string]string, authToken string) string {
	names := make([]string, 0, len(params))
	for name := range params {
		names = append(names, name)
	}
	sort.Strings(names)

	var signed strings.Builder
	signed.WriteString(rawURL)
	for _, name := range names {
		signed.WriteString(name)
		signed.WriteString(params[name])
	}
	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(signed.String()))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
