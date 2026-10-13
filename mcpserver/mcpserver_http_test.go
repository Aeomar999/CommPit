package mcpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func mcpPost(t *testing.T, url, session string, payload map[string]any) (int, http.Header, []byte) {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if session != "" {
		req.Header.Set("Mcp-Session-Id", session)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, resp.Header, body
}

// decodeFrame extracts the JSON payload from a streamable-HTTP response,
// whether plain JSON or SSE-framed.
func decodeFrame(t *testing.T, body []byte, out any) {
	t.Helper()
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		if err := json.Unmarshal(trimmed, out); err != nil {
			t.Fatalf("decode JSON: %v: %s", err, body)
		}
		return
	}
	var payload strings.Builder
	for _, line := range strings.Split(string(body), "\n") {
		if data, ok := strings.CutPrefix(line, "data:"); ok {
			payload.WriteString(strings.TrimSpace(data))
		}
	}
	if payload.Len() == 0 {
		t.Fatalf("no data frame in: %s", body)
	}
	if err := json.Unmarshal([]byte(payload.String()), out); err != nil {
		t.Fatalf("decode SSE frame: %v: %s", err, body)
	}
}

func TestMCP_HTTPWire(t *testing.T) {
	server, _, _ := setupTestMCP(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	status, headers, body := mcpPost(t, httpServer.URL, "", map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "0"},
		},
	})
	if status != http.StatusOK {
		t.Fatalf("initialize: status %d: %s", status, body)
	}
	session := headers.Get("Mcp-Session-Id")
	if session == "" {
		t.Fatalf("initialize: missing session id: %s", body)
	}

	status, _, body = mcpPost(t, httpServer.URL, session, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})
	if status != http.StatusAccepted && status != http.StatusOK {
		t.Fatalf("initialized: status %d: %s", status, body)
	}

	status, _, body = mcpPost(t, httpServer.URL, session, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	if status != http.StatusOK {
		t.Fatalf("tools/list: status %d: %s", status, body)
	}
	var listed struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	decodeFrame(t, body, &listed)
	names := map[string]bool{}
	for _, tool := range listed.Result.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"list_messages", "wait_for_message", "get_latest_otp", "get_latest_email", "simulate_inbound", "list_requests", "reset_project"} {
		if !names[want] {
			t.Errorf("missing tool %s, got %v", want, names)
		}
	}

	status, _, body = mcpPost(t, httpServer.URL, session, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "simulate_inbound",
			"arguments": map[string]any{"from": "+15005550006", "to": "+15555550100", "body": "wire check"},
		},
	})
	if status != http.StatusOK {
		t.Fatalf("tools/call: status %d: %s", status, body)
	}
	var called struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	decodeFrame(t, body, &called)
	if len(called.Result.Content) != 1 || called.Result.Content[0].Type != "text" {
		t.Fatalf("unexpected call content: %s", body)
	}
	var msg map[string]any
	if err := json.Unmarshal([]byte(called.Result.Content[0].Text), &msg); err != nil {
		t.Fatalf("call text is not JSON: %v", err)
	}
	if msg["direction"] != "inbound" || msg["body_text"] != "wire check" {
		t.Fatalf("unexpected message: %v", msg)
	}
}
