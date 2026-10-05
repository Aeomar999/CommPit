package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/sim"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

// failingStore wraps a Store and fails on specific operations to simulate store failure.
type failingStore struct {
	core.Store
	failCreateMessage bool
}

func (f *failingStore) CreateMessage(ctx context.Context, m *core.Message) error {
	if f.failCreateMessage {
		return errors.New("database connection lost")
	}
	return f.Store.CreateMessage(ctx, m)
}

type customSimulator struct {
	core.Simulator
	ruleFunc func(ctx context.Context, projectID string, req core.SendRequest) (*core.Error, *core.SimResult)
}

func (c *customSimulator) Evaluate(ctx context.Context, projectID string, req core.SendRequest) (*core.Error, *core.SimResult) {
	if c.ruleFunc != nil {
		return c.ruleFunc(ctx, projectID, req)
	}
	return c.Simulator.Evaluate(ctx, projectID, req)
}

func setupTestRouter(t *testing.T, storeOverride core.Store, simOverride core.Simulator) (http.Handler, core.Simulator, core.Store) {
	t.Helper()
	store, err := sqlite.NewStore(":memory:", 1)
	if err != nil {
		t.Fatalf("sqlite.NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	var effectiveStore core.Store = store
	if storeOverride != nil {
		effectiveStore = storeOverride
	}

	clock := core.NewFakeClock()
	var simInstance core.Simulator = sim.NewSimulator()
	if simOverride != nil {
		simInstance = simOverride
	}
	eventBus := bus.NewEventBus()
	resolver := core.NewProjectResolver(store)

	svc := core.NewService(core.ServiceConfig{
		Store:     effectiveStore,
		BlobStore: store,
		Bus:       eventBus,
		Simulator: simInstance,
		Clock:     clock,
		Resolver:  resolver,
	})

	handlers := NewHandlers(svc, resolver, eventBus)
	return handlers.Routes(), simInstance, effectiveStore
}

func TestHandlers_InvalidNumber(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	body := map[string]any{
		"from": "+15555550100",
		"to":   "+15005550001",
		"body": "Hello test",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/sms", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'error' object in response, got %v", resp)
	}

	if errObj["code"] != "invalid_number" {
		t.Errorf("expected code invalid_number, got %v", errObj["code"])
	}
}

func TestHandlers_RateLimitedRule(t *testing.T) {
	customSim := &customSimulator{
		Simulator: sim.NewSimulator(),
		ruleFunc: func(_ context.Context, _ string, req core.SendRequest) (*core.Error, *core.SimResult) {
			if len(req.To) > 0 && req.To[0] == "+14155552671" {
				return core.NewRateLimited("rate limit exceeded", "to"), &core.SimResult{}
			}
			return nil, &core.SimResult{}
		},
	}
	router, _, _ := setupTestRouter(t, nil, customSim)

	body := map[string]any{
		"from": "+15555550100",
		"to":   "+14155552671",
		"body": "Hello test",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/sms", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'error' object in response, got %v", resp)
	}

	if errObj["code"] != "rate_limited" {
		t.Errorf("expected code rate_limited, got %v", errObj["code"])
	}
}

func TestHandlers_StoreFailureReturns500Internal(t *testing.T) {
	rawStore, err := sqlite.NewStore(":memory:", 1)
	if err != nil {
		t.Fatalf("sqlite.NewStore: %v", err)
	}
	defer rawStore.Close()

	wrapped := &failingStore{
		Store:             rawStore,
		failCreateMessage: true,
	}

	router, _, _ := setupTestRouter(t, wrapped, nil)

	body := map[string]any{
		"from": "+15555550100",
		"to":   "+14155552671",
		"body": "Hello test",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/sms", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'error' object in response, got %v", resp)
	}

	if errObj["code"] != "internal" {
		t.Errorf("expected code internal, got %v", errObj["code"])
	}
}

func TestHandlers_SuccessCarriesContentTypeJSON(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	body := map[string]any{
		"from": "+15555550100",
		"to":   "+14155552671",
		"body": "Hello success",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/sms", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_BatchSendCarries202AndContentTypeJSON(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	body := map[string]any{
		"from": "+15555550100",
		"to":   []string{"+14155552671", "+14155552672"},
		"body": "Hello batch",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/sms", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_StartVerificationCarries201AndContentTypeJSON(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	body := map[string]any{
		"to":      "+14155552671",
		"channel": "sms",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/verifications", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_CheckVerificationNotFoundReturns404(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	body := map[string]any{
		"code": "123456",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/verifications/vrf_nonexistent1234567890/check", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'error' object in response, got %v", resp)
	}

	if errObj["code"] != "verification_not_found" {
		t.Errorf("expected code verification_not_found, got %v", errObj["code"])
	}
}

func TestHandlers_GetMessageNotFoundReturns404(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/messages/msg_nonexistent1234567890", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'error' object in response, got %v", resp)
	}

	if errObj["code"] != "not_found" {
		t.Errorf("expected code not_found, got %v", errObj["code"])
	}
}

func TestHandlers_SimulateInboundCarries201AndContentTypeJSON(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	body := map[string]any{
		"from": "+14155552671",
		"to":   "+15555550100",
		"body": "Hello inbound",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inbound", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_ValidationErrorReturns400(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/sms", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'error' object in response, got %v", resp)
	}

	if errObj["code"] != "validation_error" {
		t.Errorf("expected code validation_error, got %v", errObj["code"])
	}
}

func TestHandlers_FormattedRecipientResolvesToConversation(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	// Send message to formatted recipient
	sendBody := map[string]any{
		"from": "+15555550100",
		"to":   "+1 (500) 555-0006",
		"body": "Test message",
	}
	sendPayload, _ := json.Marshal(sendBody)
	sendReq := httptest.NewRequest(http.MethodPost, "/sms", bytes.NewReader(sendPayload))
	sendReq.Header.Set("Content-Type", "application/json")
	sendRec := httptest.NewRecorder()
	router.ServeHTTP(sendRec, sendReq)

	if sendRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", sendRec.Code, sendRec.Body.String())
	}

	// Query with unformatted E.164
	listReq1 := httptest.NewRequest(http.MethodGet, "/messages?to=%2B15005550006", nil)
	listRec1 := httptest.NewRecorder()
	router.ServeHTTP(listRec1, listReq1)

	if listRec1.Code != http.StatusOK {
		t.Fatalf("list with normalized to failed: %d", listRec1.Code)
	}
	var resp1 map[string]any
	json.Unmarshal(listRec1.Body.Bytes(), &resp1)
	items1, _ := resp1["messages"].([]any)
	if len(items1) != 1 {
		t.Fatalf("expected 1 message for normalized query, got %d (body: %s)", len(items1), listRec1.Body.String())
	}

	// Query with formatted number (with spaces and parentheses URL-encoded)
	listReq2 := httptest.NewRequest(http.MethodGet, "/messages?to=%2B1%20(500)%20555-0006", nil)
	listRec2 := httptest.NewRecorder()
	router.ServeHTTP(listRec2, listReq2)

	if listRec2.Code != http.StatusOK {
		t.Fatalf("list with formatted to failed: %d", listRec2.Code)
	}
	var resp2 map[string]any
	json.Unmarshal(listRec2.Body.Bytes(), &resp2)
	items2, _ := resp2["messages"].([]any)
	if len(items2) != 1 {
		t.Fatalf("expected 1 message for formatted query, got %d (body: %s)", len(items2), listRec2.Body.String())
	}
}

func TestHandlers_BatchWithRejectedRecipients(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	batchBody := map[string]any{
		"from": "+15555550100",
		"to":   []string{"+1 (500) 555-0006", "invalid-phone", "+15005550001", "+15005550007"},
		"body": "Batch announcement",
	}
	payload, _ := json.Marshal(batchBody)
	req := httptest.NewRequest(http.MethodPost, "/sms", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var batchResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &batchResp); err != nil {
		t.Fatalf("failed to decode batch response: %v", err)
	}

	total, _ := batchResp["total"].(float64)
	if int(total) != 2 {
		t.Errorf("expected total 2, got %v", total)
	}

	rejected, ok := batchResp["rejected"].([]any)
	if !ok || len(rejected) != 2 {
		t.Fatalf("expected 2 rejected entries, got %v", rejected)
	}
}

func TestHandlers_CallbackURLOmittedWhenEmpty(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	body := map[string]any{
		"from":         "+15555550100",
		"to":           "+15005550006",
		"body":         "No callback test",
		"callback_url": "",
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/sms", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var msgResp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &msgResp)
	if cb, ok := msgResp["callback_url"]; ok && cb != nil && cb != "" {
		t.Errorf("expected callback_url to be omitted or nil, got %v", cb)
	}
}
