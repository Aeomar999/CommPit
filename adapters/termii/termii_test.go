package termii_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/adapters/termii"
	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/sim"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

const (
	testAPIKey = "tl_test_key_12345"
	testFrom   = "+15555550100"
	testSender = "TestSender"
	testToNG   = "+2348031234567"
	testToNG2  = "+2348051234567"
	testToNG3  = "+2348071234567"
)

func setupTestTermii(t *testing.T) (http.Handler, core.Store) {
	t.Helper()
	store, err := sqlite.NewStore(":memory:", 1)
	if err != nil {
		t.Fatalf("sqlite.NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	eventBus := bus.NewEventBus()
	resolver := core.NewProjectResolver(store)
	svc := core.NewService(core.ServiceConfig{
		Store:     store,
		BlobStore: store,
		Bus:       eventBus,
		Simulator: sim.NewSimulator(),
		Clock:     core.NewFakeClock(),
		Resolver:  resolver,
	})

	sink := adapterkit.RequestLogSinkFunc(func(ctx context.Context, entry *core.RequestLog) error {
		return store.CreateRequestLog(ctx, entry)
	})
	kit := adapterkit.New(resolver, adapterkit.WithBus(eventBus), adapterkit.WithSink(sink))
	handler := kit.Wrap(termii.New(svc), termii.Extractor(), adapterkit.WithRequired(true))
	return handler, store
}

func termiiRequest(t *testing.T, target string, payload map[string]any, withKey bool) *http.Request {
	t.Helper()
	if withKey {
		payload["api_key"] = testAPIKey
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func decodeTermii(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v (%s)", err, rec.Body.String())
	}
	return resp
}

func checkContentType(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

func TestTermii_SendSingle(t *testing.T) {
	handler, _ := setupTestTermii(t)

	req := termiiRequest(t, "/api/sms/send", map[string]any{
		"to":   testToNG,
		"from": testSender,
		"sms":  "Hello from mocksms",
	}, true)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	checkContentType(t, rec)
	resp := decodeTermii(t, rec)
	if _, ok := resp["message_id"].(float64); !ok {
		t.Errorf("expected numeric message_id, got %v", resp["message_id"])
	}
	for _, key := range []string{"message", "balance", "user"} {
		if _, ok := resp[key]; !ok {
			t.Errorf("expected key %q, got %v", key, resp)
		}
	}
	if _, ok := resp["code"]; ok {
		t.Errorf("single send must not carry bulk code, got %v", resp)
	}
}

func TestTermii_SendArray(t *testing.T) {
	handler, store := setupTestTermii(t)

	req := termiiRequest(t, "/api/sms/send", map[string]any{
		"to":   []string{testToNG, testToNG2, testToNG3},
		"from": testSender,
		"sms":  "Batch announcement",
	}, true)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := decodeTermii(t, rec)
	if resp["code"] != "ok" {
		t.Errorf("expected bulk code ok, got %v", resp)
	}

	projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "termii", testAPIKey)
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	msgs, _, err := store.ListMessages(context.Background(), projectID, core.MessageFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 stored messages, got %d", len(msgs))
	}
	for _, m := range msgs {
		if m.Provider != "termii" {
			t.Errorf("expected provider termii, got %q", m.Provider)
		}
	}
}

func TestTermii_SendLimits(t *testing.T) {
	newList := func(n int) []string {
		list := make([]string, 0, n)
		for i := 0; i < n; i++ {
			list = append(list, testToNG)
		}
		return list
	}

	t.Run("send rejects more than 100 recipients", func(t *testing.T) {
		handler, _ := setupTestTermii(t)
		req := termiiRequest(t, "/api/sms/send", map[string]any{
			"to":   newList(101),
			"from": testSender,
			"sms":  "Too many",
		}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
		if resp := decodeTermii(t, rec); resp["code"] != "validation_error" {
			t.Errorf("expected code validation_error, got %v", resp["code"])
		}
	})

	t.Run("bulk accepts hundreds of recipients", func(t *testing.T) {
		handler, store := setupTestTermii(t)
		req := termiiRequest(t, "/api/sms/send/bulk", map[string]any{
			"to":   newList(150),
			"from": testSender,
			"sms":  "Bulk hundreds",
		}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "termii", testAPIKey)
		if err != nil {
			t.Fatalf("resolve project: %v", err)
		}
		msgs, _, err := store.ListMessages(context.Background(), projectID, core.MessageFilter{Limit: 200})
		if err != nil {
			t.Fatalf("ListMessages: %v", err)
		}
		if len(msgs) != 150 {
			t.Fatalf("expected 150 stored messages, got %d", len(msgs))
		}
	})

	t.Run("bulk rejects more than 10000 recipients", func(t *testing.T) {
		handler, _ := setupTestTermii(t)
		req := termiiRequest(t, "/api/sms/send/bulk", map[string]any{
			"to":   newList(10001),
			"from": testSender,
			"sms":  "Too many",
		}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestTermii_SendErrors(t *testing.T) {
	cases := []struct {
		name       string
		payload    map[string]any
		auth       bool
		wantStatus int
		wantCode   string
	}{
		{name: "no api key", payload: map[string]any{"to": testToNG, "from": testSender, "sms": "hi"}, auth: false, wantStatus: http.StatusUnauthorized, wantCode: "unauthorized"},
		{name: "missing to", payload: map[string]any{"from": testSender, "sms": "hi"}, auth: true, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "missing from", payload: map[string]any{"to": testToNG, "sms": "hi"}, auth: true, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "missing sms", payload: map[string]any{"to": testToNG, "from": testSender}, auth: true, wantStatus: http.StatusBadRequest, wantCode: "validation_error"},
		{name: "invalid to", payload: map[string]any{"to": "+15005550001", "from": testSender, "sms": "hi"}, auth: true, wantStatus: http.StatusBadRequest, wantCode: "invalid_number"},
		{name: "unroutable to", payload: map[string]any{"to": "+15005550002", "from": testSender, "sms": "hi"}, auth: true, wantStatus: http.StatusBadRequest, wantCode: "unroutable"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler, _ := setupTestTermii(t)
			req := termiiRequest(t, "/api/sms/send", tc.payload, tc.auth)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
			checkContentType(t, rec)
			if resp := decodeTermii(t, rec); resp["code"] != tc.wantCode {
				t.Errorf("expected code %q, got %v", tc.wantCode, resp["code"])
			}
		})
	}
}

func TestTermii_SenderAllowList(t *testing.T) {
	allowListed := func(t *testing.T, store core.Store, senders ...string) {
		t.Helper()
		projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "termii", testAPIKey)
		if err != nil {
			t.Fatalf("resolve project: %v", err)
		}
		project, err := store.GetProject(context.Background(), projectID)
		if err != nil {
			t.Fatalf("GetProject: %v", err)
		}
		list := make([]interface{}, 0, len(senders))
		for _, s := range senders {
			list = append(list, s)
		}
		if project.Settings == nil {
			project.Settings = map[string]interface{}{}
		}
		project.Settings["termii.sender_allowlist"] = list
		if err := store.UpdateProject(context.Background(), project); err != nil {
			t.Fatalf("UpdateProject: %v", err)
		}
	}

	t.Run("unlisted sender rejected when allow-list set", func(t *testing.T) {
		handler, store := setupTestTermii(t)
		allowListed(t, store, "AllowedSender")
		req := termiiRequest(t, "/api/sms/send", map[string]any{
			"to": testToNG, "from": "BlockedSender", "sms": "hi",
		}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
		if resp := decodeTermii(t, rec); resp["code"] != "invalid_sender" {
			t.Errorf("expected code invalid_sender, got %v", resp["code"])
		}
	})

	t.Run("listed sender accepted", func(t *testing.T) {
		handler, store := setupTestTermii(t)
		allowListed(t, store, "AllowedSender")
		req := termiiRequest(t, "/api/sms/send", map[string]any{
			"to": testToNG, "from": "AllowedSender", "sms": "hi",
		}, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestTermii_NumberSend(t *testing.T) {
	handler, _ := setupTestTermii(t)
	req := termiiRequest(t, "/api/sms/number/send", map[string]any{
		"to": testToNG, "from": "+2348012345678", "sms": "From a number",
	}, true)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := decodeTermii(t, rec)
	if _, ok := resp["message_id"].(float64); !ok {
		t.Errorf("expected numeric message_id, got %v", resp["message_id"])
	}
}

func TestTermii_RequestLoggedWithMaskedKey(t *testing.T) {
	handler, store := setupTestTermii(t)
	req := termiiRequest(t, "/api/sms/send", map[string]any{
		"to": testToNG, "from": testSender, "sms": "logged",
	}, true)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "termii", testAPIKey)
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	logs, _, err := store.ListRequestLogs(context.Background(), projectID, 10, "")
	if err != nil {
		t.Fatalf("ListRequestLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 request log, got %d", len(logs))
	}
	if logs[0].Adapter != "termii" {
		t.Errorf("expected adapter termii, got %q", logs[0].Adapter)
	}
	if strings.Contains(string(logs[0].RequestBody), testAPIKey) {
		t.Errorf("stored request body leaks api_key: %s", logs[0].RequestBody)
	}
}

func TestTermii_WriteErrorMapping(t *testing.T) {
	adapter := termii.New(nil)
	cases := []struct {
		name       string
		err        *core.Error
		wantStatus int
		wantCode   string
	}{
		{"invalid number", core.NewInvalidNumber("bad", "to"), http.StatusBadRequest, "invalid_number"},
		{"invalid sender", core.NewInvalidSender("bad", "from"), http.StatusBadRequest, "invalid_sender"},
		{"unsubscribed", core.NewUnsubscribed("stop", "to"), http.StatusBadRequest, "unsubscribed"},
		{"rate limited", core.NewRateLimited("slow", "to"), http.StatusTooManyRequests, "rate_limited"},
		{"not found", core.NewNotFound("missing", "id"), http.StatusNotFound, "not_found"},
		{"unauthorized", core.NewUnauthorized("auth"), http.StatusUnauthorized, "unauthorized"},
		{"internal", core.NewInternal("boom"), http.StatusInternalServerError, "internal"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			adapter.WriteError(rec, tc.err)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
			checkContentType(t, rec)
			resp := decodeTermii(t, rec)
			if resp["code"] != tc.wantCode {
				t.Errorf("expected code %q, got %v", tc.wantCode, resp["code"])
			}
			if _, ok := resp["message"]; !ok {
				t.Errorf("expected message field, got %v", resp)
			}
		})
	}
}
