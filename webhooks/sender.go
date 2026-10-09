package webhooks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

// maxResponseBody caps stored response bodies, matching the 64 KB logging
// convention used across the codebase.
const maxResponseBody = 64 * 1024

// Sender delivers one webhook HTTP request. Implementations must honor ctx
// cancellation; the worker additionally bounds each attempt by timeout.
type Sender interface {
	Send(ctx context.Context, url string, payload []byte, headers map[string]string) (status int, body []byte, err error)
}

// SenderFunc adapts a function to a Sender.
type SenderFunc func(ctx context.Context, url string, payload []byte, headers map[string]string) (int, []byte, error)

// Send implements Sender.
func (f SenderFunc) Send(ctx context.Context, url string, payload []byte, headers map[string]string) (int, []byte, error) {
	return f(ctx, url, payload, headers)
}

// HTTPSender is the production Sender over net/http.
type HTTPSender struct {
	client *http.Client
}

// NewHTTPSender builds a production Sender sharing one client.
func NewHTTPSender() *HTTPSender {
	return &HTTPSender{client: &http.Client{}}
}

// Send posts the payload and returns the response status and capped body.
func (s *HTTPSender) Send(ctx context.Context, url string, payload []byte, headers map[string]string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, fmt.Errorf("webhooks: build request: %w", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("webhooks: read response: %w", err)
	}
	return resp.StatusCode, body, nil
}
