package adapterkit_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/core"
)

// mockAdapter implements adapterkit.Adapter for testing.
type mockAdapter struct {
	name          string
	routesFunc    func(r chi.Router)
	lastErrorCode core.ErrorCode
	lastErrorMsg  string
}

func (m *mockAdapter) Name() string {
	if m.name != "" {
		return m.name
	}
	return "testadapter"
}

func (m *mockAdapter) Routes(r chi.Router) {
	if m.routesFunc != nil {
		m.routesFunc(r)
	}
}

func (m *mockAdapter) WriteError(w http.ResponseWriter, err *core.Error) {
	m.lastErrorCode = err.Code
	m.lastErrorMsg = err.Message
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus())
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error_code":    err.Code,
		"error_message": err.Message,
	})
}

// mockResolver implements core.ProjectResolver for testing.
type mockResolver struct {
	resolveFunc func(ctx context.Context, provider, key string) (string, error)
}

func (m *mockResolver) Resolve(ctx context.Context, provider, key string) (string, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(ctx, provider, key)
	}
	return "prj_" + key, nil
}

// mockBus records published events.
type mockBus struct {
	mu     sync.Mutex
	events []core.Event
}

func (b *mockBus) Publish(ctx context.Context, event core.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
}

func (b *mockBus) Subscribe(eventType string, handler core.EventHandler) core.Subscription {
	return nil
}

func (b *mockBus) Events() []core.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	copied := make([]core.Event, len(b.events))
	copy(copied, b.events)
	return copied
}

func TestRecoverer(t *testing.T) {
	t.Run("recovers from panic and writes 500 internal error", func(t *testing.T) {
		adapter := &mockAdapter{name: "twilio"}
		middleware := adapterkit.Recoverer(adapter)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("unexpected boom!")
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}
		if adapter.lastErrorCode != core.ErrCodeInternal {
			t.Fatalf("expected ErrCodeInternal, got %s", adapter.lastErrorCode)
		}

		var resp map[string]string
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["error_code"] != string(core.ErrCodeInternal) {
			t.Errorf("expected error_code %q, got %q", core.ErrCodeInternal, resp["error_code"])
		}
	})

	t.Run("passes through http.ErrAbortHandler", func(t *testing.T) {
		adapter := &mockAdapter{name: "twilio"}
		middleware := adapterkit.Recoverer(adapter)

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic(http.ErrAbortHandler)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		defer func() {
			r := recover()
			if r == nil {
				t.Fatalf("expected panic http.ErrAbortHandler to propagate")
			}
			if !errors.Is(r.(error), http.ErrAbortHandler) {
				t.Fatalf("expected http.ErrAbortHandler, got %v", r)
			}
		}()

		handler.ServeHTTP(rec, req)
	})

	t.Run("calls next normally when no panic", func(t *testing.T) {
		adapter := &mockAdapter{name: "twilio"}
		middleware := adapterkit.Recoverer(adapter)

		called := false
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if !called {
			t.Fatal("expected handler to be called")
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})
}

func TestCredentialResolution(t *testing.T) {
	t.Run("basic auth extractor", func(t *testing.T) {
		adapter := &mockAdapter{name: "twilio"}
		resolver := &mockResolver{
			resolveFunc: func(ctx context.Context, provider, key string) (string, error) {
				if provider != "twilio" || key != "AC12345" {
					return "", errors.New("mismatch")
				}
				return "prj_twilio_123", nil
			},
		}

		extractor := adapterkit.BasicAuthExtractor()
		mw := adapterkit.ResolveProject(adapter, resolver, extractor)

		var capturedProjectID string
		var capturedCred string

		next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedProjectID = adapterkit.ProjectID(r)
			capturedCred = adapterkit.Credential(r)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.SetBasicAuth("AC12345", "auth_secret_token")
		rec := httptest.NewRecorder()

		next.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if capturedProjectID != "prj_twilio_123" {
			t.Errorf("expected project prj_twilio_123, got %q", capturedProjectID)
		}
		if capturedCred != "AC12345" {
			t.Errorf("expected credential AC12345, got %q", capturedCred)
		}
	})

	t.Run("json body extractor preserves request body", func(t *testing.T) {
		adapter := &mockAdapter{name: "termii"}
		resolver := &mockResolver{
			resolveFunc: func(ctx context.Context, provider, key string) (string, error) {
				if key != "tl_secret_key" {
					return "", errors.New("bad key")
				}
				return "prj_termii_999", nil
			},
		}

		extractor := adapterkit.JSONBodyKeyExtractor("api_key")
		mw := adapterkit.ResolveProject(adapter, resolver, extractor)

		bodyJSON := `{"api_key":"tl_secret_key","to":"+2341234567","sms":"Hello"}`
		var handlerReadBody string

		next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read body in handler: %v", err)
			}
			handlerReadBody = string(bodyBytes)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodPost, "/sms/send", strings.NewReader(bodyJSON))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		next.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if handlerReadBody != bodyJSON {
			t.Errorf("expected handler to read full body %q, got %q", bodyJSON, handlerReadBody)
		}
	})

	t.Run("first of extractor falls back in order", func(t *testing.T) {
		adapter := &mockAdapter{name: "twilio"}
		resolver := &mockResolver{}

		extractor := adapterkit.FirstOf(
			adapterkit.BasicAuthExtractor(),
			adapterkit.HeaderExtractor("X-Account-Sid"),
			adapterkit.QueryExtractor("account_sid"),
		)
		mw := adapterkit.ResolveProject(adapter, resolver, extractor)

		var capturedCred string
		next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedCred = adapterkit.Credential(r)
			w.WriteHeader(http.StatusOK)
		}))

		// Query param fallback
		req := httptest.NewRequest(http.MethodGet, "/test?account_sid=AC_QUERY_99", nil)
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, req)

		if capturedCred != "AC_QUERY_99" {
			t.Errorf("expected AC_QUERY_99, got %q", capturedCred)
		}

		// Header precedes query
		req2 := httptest.NewRequest(http.MethodGet, "/test?account_sid=AC_QUERY_99", nil)
		req2.Header.Set("X-Account-Sid", "AC_HEADER_88")
		rec2 := httptest.NewRecorder()
		next.ServeHTTP(rec2, req2)

		if capturedCred != "AC_HEADER_88" {
			t.Errorf("expected AC_HEADER_88, got %q", capturedCred)
		}
	})

	t.Run("missing credential with WithDefaultProject falls back", func(t *testing.T) {
		adapter := &mockAdapter{name: "test"}
		resolver := &mockResolver{}

		mw := adapterkit.ResolveProject(
			adapter,
			resolver,
			adapterkit.BasicAuthExtractor(),
			adapterkit.WithDefaultProject("default_project"),
		)

		var capturedProjectID string
		next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedProjectID = adapterkit.ProjectID(r)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if capturedProjectID != "default_project" {
			t.Errorf("expected default_project, got %q", capturedProjectID)
		}
	})

	t.Run("missing credential returns 401 when required", func(t *testing.T) {
		adapter := &mockAdapter{name: "test"}
		resolver := &mockResolver{}

		mw := adapterkit.ResolveProject(adapter, resolver, adapterkit.BasicAuthExtractor(), adapterkit.WithRequired(true))

		next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
		if adapter.lastErrorCode != core.ErrCodeUnauthorized {
			t.Errorf("expected ErrCodeUnauthorized, got %s", adapter.lastErrorCode)
		}
	})

	t.Run("resolver error returns 500 internal error", func(t *testing.T) {
		adapter := &mockAdapter{name: "test"}
		resolver := &mockResolver{
			resolveFunc: func(ctx context.Context, provider, key string) (string, error) {
				return "", errors.New("database locked")
			},
		}

		mw := adapterkit.ResolveProject(adapter, resolver, adapterkit.HeaderExtractor("X-Key"))

		next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Key", "mykey")
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rec.Code)
		}
		if adapter.lastErrorCode != core.ErrCodeInternal {
			t.Errorf("expected ErrCodeInternal, got %s", adapter.lastErrorCode)
		}
	})
}

func TestMasking(t *testing.T) {
	t.Run("masks sensitive headers", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte("AC12345678:my_auth_secret_password")),
			"X-Api-Key":     "secret_api_key_12345",
			"Cookie":        "session=abc123xyz",
			"Content-Type":  "application/json",
			"User-Agent":    "curl/7.68.0",
		}

		masked := adapterkit.MaskHeaders(headers)

		if !strings.HasPrefix(masked["Authorization"], "Basic ") || strings.Contains(masked["Authorization"], "my_auth_secret_password") {
			t.Errorf("authorization header not masked properly: %q", masked["Authorization"])
		}
		if masked["X-Api-Key"] == "secret_api_key_12345" || !strings.Contains(masked["X-Api-Key"], "*") && !strings.Contains(masked["X-Api-Key"], "...") {
			t.Errorf("api key header not masked: %q", masked["X-Api-Key"])
		}
		if masked["Cookie"] == "session=abc123xyz" {
			t.Errorf("cookie header not masked: %q", masked["Cookie"])
		}
		if masked["Content-Type"] != "application/json" {
			t.Errorf("non-sensitive header altered: %q", masked["Content-Type"])
		}
		if masked["User-Agent"] != "curl/7.68.0" {
			t.Errorf("non-sensitive header altered: %q", masked["User-Agent"])
		}
	})

	t.Run("masks json body credentials", func(t *testing.T) {
		rawJSON := []byte(`{"api_key":"TL_SECRET_TOKEN_9999","to":"+233241234567","from":"Company","message":"Hello world"}`)
		masked := adapterkit.MaskBody("application/json", rawJSON)

		var parsed map[string]any
		if err := json.Unmarshal(masked, &parsed); err != nil {
			t.Fatalf("masked body is invalid json: %v", err)
		}

		apiKeyVal, ok := parsed["api_key"].(string)
		if !ok || apiKeyVal == "TL_SECRET_TOKEN_9999" {
			t.Errorf("api_key not masked: %v", apiKeyVal)
		}
		if parsed["to"] != "+233241234567" || parsed["from"] != "Company" || parsed["message"] != "Hello world" {
			t.Errorf("unexpected field changes: %v", parsed)
		}
	})

	t.Run("masks form urlencoded credentials", func(t *testing.T) {
		form := url.Values{}
		form.Set("api_key", "super_secret_form_token")
		form.Set("To", "+15551234567")
		form.Set("Body", "Hi there")

		masked := adapterkit.MaskBody("application/x-www-form-urlencoded", []byte(form.Encode()))
		parsed, err := url.ParseQuery(string(masked))
		if err != nil {
			t.Fatalf("masked body is invalid query: %v", err)
		}

		if parsed.Get("api_key") == "super_secret_form_token" {
			t.Errorf("form api_key not masked: %q", parsed.Get("api_key"))
		}
		if parsed.Get("To") != "+15551234567" || parsed.Get("Body") != "Hi there" {
			t.Errorf("form non-sensitive values changed: %v", parsed)
		}
	})

	t.Run("masks url query parameters in path", func(t *testing.T) {
		rawURL := "/2010-04-01/Messages.json?api_key=secret_12345&To=%2B1555123"
		masked := adapterkit.MaskURL(rawURL)

		if strings.Contains(masked, "secret_12345") {
			t.Errorf("expected secret_12345 to be masked in URL, got %q", masked)
		}
		if !strings.Contains(masked, "To=%2B1555123") {
			t.Errorf("expected To parameter to remain intact, got %q", masked)
		}
	})
}

func TestRequestLogging(t *testing.T) {
	t.Run("records request log with 64 KB cap and masking", func(t *testing.T) {
		adapter := &mockAdapter{name: "twilio"}
		bus := &mockBus{}
		var recordedLog *core.RequestLog

		sink := adapterkit.RequestLogSinkFunc(func(ctx context.Context, entry *core.RequestLog) error {
			recordedLog = entry
			return nil
		})

		cfg := adapterkit.LoggingConfig{
			AdapterName: adapter.Name(),
			Sink:        sink,
			Bus:         bus,
			Clock:       core.RealClock{},
		}

		mw := adapterkit.RequestLogger(adapter, cfg)

		// Create 70 KB request body (> 64 KB cap)
		largeReqBody := strings.Repeat("A", 70*1024)
		// Handler responds with 70 KB response body (> 64 KB cap)
		largeRespBody := strings.Repeat("B", 70*1024)

		var handlerReadLen int

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("handler failed to read body: %v", err)
			}
			handlerReadLen = len(b)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(largeRespBody))
		}))

		req := httptest.NewRequest(http.MethodPost, "/2010-04-01/Accounts/AC123/Messages.json", strings.NewReader(largeReqBody))
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("AC123:secret_token_12345")))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// Verify client received entire response body (70 KB)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d", rec.Code)
		}
		if rec.Body.Len() != 70*1024 {
			t.Fatalf("expected client to receive 70 KB response, got %d", rec.Body.Len())
		}
		// Verify handler was able to read entire request body (70 KB)
		if handlerReadLen != 70*1024 {
			t.Fatalf("expected handler to read full 70 KB, got %d", handlerReadLen)
		}

		// Verify sink received the log entry
		if recordedLog == nil {
			t.Fatal("expected sink to receive RequestLog")
		}
		if !strings.HasPrefix(recordedLog.ID, "req_") {
			t.Errorf("expected ID prefix req_, got %q", recordedLog.ID)
		}
		if recordedLog.Adapter != "twilio" {
			t.Errorf("expected adapter twilio, got %q", recordedLog.Adapter)
		}
		if recordedLog.Method != http.MethodPost {
			t.Errorf("expected method POST, got %q", recordedLog.Method)
		}
		if recordedLog.ResponseStatus != http.StatusCreated {
			t.Errorf("expected response status 201, got %d", recordedLog.ResponseStatus)
		}

		// Capped at 64 KB (65536 bytes)
		const maxCap = 64 * 1024
		if len(recordedLog.RequestBody) != maxCap {
			t.Errorf("expected RequestBody capped at 64 KB (%d), got %d", maxCap, len(recordedLog.RequestBody))
		}
		if len(recordedLog.ResponseBody) != maxCap {
			t.Errorf("expected ResponseBody capped at 64 KB (%d), got %d", maxCap, len(recordedLog.ResponseBody))
		}

		// Headers masked
		authHeader := recordedLog.RequestHeaders["Authorization"]
		if strings.Contains(authHeader, "secret_token_12345") {
			t.Errorf("recorded auth header not masked: %q", authHeader)
		}

		// Event published to bus
		events := bus.Events()
		if len(events) != 1 {
			t.Fatalf("expected 1 event published, got %d", len(events))
		}
		if events[0].Type != core.EventRequestLogged {
			t.Errorf("expected event type %s, got %s", core.EventRequestLogged, events[0].Type)
		}
	})
}

func TestKitWrap(t *testing.T) {
	t.Run("end to end wrap executes full stack", func(t *testing.T) {
		bus := &mockBus{}
		var recordedLogs []*core.RequestLog
		sink := adapterkit.RequestLogSinkFunc(func(ctx context.Context, entry *core.RequestLog) error {
			recordedLogs = append(recordedLogs, entry)
			return nil
		})

		resolver := &mockResolver{
			resolveFunc: func(ctx context.Context, provider, key string) (string, error) {
				return "prj_e2e_" + key, nil
			},
		}

		kit := adapterkit.New(resolver,
			adapterkit.WithBus(bus),
			adapterkit.WithSink(sink),
		)

		adapter := &mockAdapter{
			name: "twilio",
			routesFunc: func(r chi.Router) {
				r.Post("/test", func(w http.ResponseWriter, r *http.Request) {
					projectID := adapterkit.ProjectID(r)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"project_id": projectID,
					})
				})
				r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
					panic("boom in route")
				})
			},
		}

		handler := kit.Wrap(adapter, adapterkit.BasicAuthExtractor())

		// Test success route
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"hello":"world"}`))
		req.SetBasicAuth("AC_TEST", "my_secret")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]string
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp["project_id"] != "prj_e2e_AC_TEST" {
			t.Errorf("expected prj_e2e_AC_TEST, got %q", resp["project_id"])
		}

		// Test panic route is captured in request log as 500
		reqPanic := httptest.NewRequest(http.MethodGet, "/panic", nil)
		reqPanic.SetBasicAuth("AC_TEST", "my_secret")
		recPanic := httptest.NewRecorder()

		handler.ServeHTTP(recPanic, reqPanic)

		if recPanic.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 from panic, got %d", recPanic.Code)
		}

		// We should have 2 request logs recorded in sink
		if len(recordedLogs) != 2 {
			t.Fatalf("expected 2 recorded logs, got %d", len(recordedLogs))
		}
		if recordedLogs[0].ResponseStatus != http.StatusOK {
			t.Errorf("log 0 expected status 200, got %d", recordedLogs[0].ResponseStatus)
		}
		if recordedLogs[0].ProjectID != "prj_e2e_AC_TEST" {
			t.Errorf("log 0 expected project prj_e2e_AC_TEST, got %q", recordedLogs[0].ProjectID)
		}
		if recordedLogs[1].ResponseStatus != http.StatusInternalServerError {
			t.Errorf("log 1 expected status 500, got %d", recordedLogs[1].ResponseStatus)
		}
	})
}
