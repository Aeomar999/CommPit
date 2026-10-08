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
	"time"

	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/config"
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

	handlers := NewHandlers(svc, resolver, eventBus, "test", &config.SecurityConfig{
		AllowedHosts:    []string{}, // Empty = allow all for tests
		RequireXMocksms: false,      // Disable for tests
		UIAuth:          "",
	})
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

func TestHandlers_CheckVerification_WrongCodePending(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	startBody := map[string]any{
		"channel":      "sms",
		"to":           "+15005550006",
		"max_attempts": 3,
	}
	payload, _ := json.Marshal(startBody)
	req := httptest.NewRequest(http.MethodPost, "/verifications", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var startResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &startResp)
	verObj := startResp["verification"].(map[string]any)
	verID := verObj["id"].(string)

	checkPayload, _ := json.Marshal(map[string]string{"code": "000000"})
	checkReq := httptest.NewRequest(http.MethodPost, "/verifications/"+verID+"/check", bytes.NewReader(checkPayload))
	checkReq.Header.Set("Content-Type", "application/json")
	checkRec := httptest.NewRecorder()
	router.ServeHTTP(checkRec, checkReq)

	if checkRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", checkRec.Code, checkRec.Body.String())
	}
	var checkResp map[string]any
	_ = json.Unmarshal(checkRec.Body.Bytes(), &checkResp)
	if checkResp["valid"] != false {
		t.Errorf("expected valid=false, got %v", checkResp["valid"])
	}
	if checkResp["status"] != "pending" {
		t.Errorf("expected status=pending, got %v", checkResp["status"])
	}
}

func TestHandlers_CheckVerification_SuccessApproved(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	startBody := map[string]any{
		"channel": "sms",
		"to":      "+15005550006",
	}
	payload, _ := json.Marshal(startBody)
	req := httptest.NewRequest(http.MethodPost, "/verifications", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var startResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &startResp)
	verObj := startResp["verification"].(map[string]any)
	verID := verObj["id"].(string)
	verCode := verObj["code"].(string)

	checkPayload, _ := json.Marshal(map[string]string{"code": verCode})
	checkReq := httptest.NewRequest(http.MethodPost, "/verifications/"+verID+"/check", bytes.NewReader(checkPayload))
	checkReq.Header.Set("Content-Type", "application/json")
	checkRec := httptest.NewRecorder()
	router.ServeHTTP(checkRec, checkReq)

	if checkRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", checkRec.Code, checkRec.Body.String())
	}
	var checkResp map[string]any
	_ = json.Unmarshal(checkRec.Body.Bytes(), &checkResp)
	if checkResp["valid"] != true {
		t.Errorf("expected valid=true, got %v", checkResp["valid"])
	}
	if checkResp["status"] != "approved" {
		t.Errorf("expected status=approved, got %v", checkResp["status"])
	}
}

func TestHandlers_CheckVerification_MaxAttemptsReturns429(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	startBody := map[string]any{
		"channel":      "sms",
		"to":           "+15005550006",
		"max_attempts": 2,
	}
	payload, _ := json.Marshal(startBody)
	req := httptest.NewRequest(http.MethodPost, "/verifications", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var startResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &startResp)
	verObj := startResp["verification"].(map[string]any)
	verID := verObj["id"].(string)

	// Attempt 1: wrong code -> 200 pending
	checkPayload, _ := json.Marshal(map[string]string{"code": "000000"})
	checkReq1 := httptest.NewRequest(http.MethodPost, "/verifications/"+verID+"/check", bytes.NewReader(checkPayload))
	checkReq1.Header.Set("Content-Type", "application/json")
	checkRec1 := httptest.NewRecorder()
	router.ServeHTTP(checkRec1, checkReq1)
	if checkRec1.Code != http.StatusOK {
		t.Fatalf("expected attempt 1 to return 200, got %d", checkRec1.Code)
	}

	// Attempt 2: hits max attempts -> 429 max_attempts
	checkReq2 := httptest.NewRequest(http.MethodPost, "/verifications/"+verID+"/check", bytes.NewReader(checkPayload))
	checkReq2.Header.Set("Content-Type", "application/json")
	checkRec2 := httptest.NewRecorder()
	router.ServeHTTP(checkRec2, checkReq2)
	if checkRec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected attempt 2 to return 429, got %d: %s", checkRec2.Code, checkRec2.Body.String())
	}
	var errResp map[string]any
	_ = json.Unmarshal(checkRec2.Body.Bytes(), &errResp)
	errObj := errResp["error"].(map[string]any)
	if errObj["code"] != "max_attempts" {
		t.Errorf("expected code max_attempts, got %v", errObj["code"])
	}

	// Attempt 3: exhausted -> 429 max_attempts
	checkReq3 := httptest.NewRequest(http.MethodPost, "/verifications/"+verID+"/check", bytes.NewReader(checkPayload))
	checkReq3.Header.Set("Content-Type", "application/json")
	checkRec3 := httptest.NewRecorder()
	router.ServeHTTP(checkRec3, checkReq3)
	if checkRec3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected attempt 3 to return 429, got %d: %s", checkRec3.Code, checkRec3.Body.String())
	}
}

func TestHandlers_CheckVerification_AlreadyApprovedReturns404(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	startBody := map[string]any{
		"channel": "sms",
		"to":      "+15005550006",
	}
	payload, _ := json.Marshal(startBody)
	req := httptest.NewRequest(http.MethodPost, "/verifications", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var startResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &startResp)
	verObj := startResp["verification"].(map[string]any)
	verID := verObj["id"].(string)
	verCode := verObj["code"].(string)

	// First check: valid code -> 200 approved
	checkPayload, _ := json.Marshal(map[string]string{"code": verCode})
	checkReq1 := httptest.NewRequest(http.MethodPost, "/verifications/"+verID+"/check", bytes.NewReader(checkPayload))
	checkReq1.Header.Set("Content-Type", "application/json")
	checkRec1 := httptest.NewRecorder()
	router.ServeHTTP(checkRec1, checkReq1)
	if checkRec1.Code != http.StatusOK {
		t.Fatalf("expected 200 approved, got %d", checkRec1.Code)
	}

	// Second check: checking already approved verification -> 404 verification_not_found per spec §7.2
	checkReq2 := httptest.NewRequest(http.MethodPost, "/verifications/"+verID+"/check", bytes.NewReader(checkPayload))
	checkReq2.Header.Set("Content-Type", "application/json")
	checkRec2 := httptest.NewRecorder()
	router.ServeHTTP(checkRec2, checkReq2)
	if checkRec2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for already approved, got %d: %s", checkRec2.Code, checkRec2.Body.String())
	}
	var errResp map[string]any
	_ = json.Unmarshal(checkRec2.Body.Bytes(), &errResp)
	errObj := errResp["error"].(map[string]any)
	if errObj["code"] != "verification_not_found" {
		t.Errorf("expected code verification_not_found, got %v", errObj["code"])
	}
}

func TestHandlers_CheckVerification_ExpiredReturns404(t *testing.T) {
	router, _, _ := setupTestRouter(t, nil, nil)

	startBody := map[string]any{
		"channel": "sms",
		"to":      "+15005550006",
	}
	payload, _ := json.Marshal(startBody)
	req := httptest.NewRequest(http.MethodPost, "/verifications", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var startResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &startResp)
	verObj := startResp["verification"].(map[string]any)
	verID := verObj["id"].(string)

	// Expire it via endpoint
	expireReq := httptest.NewRequest(http.MethodPost, "/verifications/"+verID+"/expire", nil)
	expireRec := httptest.NewRecorder()
	router.ServeHTTP(expireRec, expireReq)
	if expireRec.Code != http.StatusOK {
		t.Fatalf("failed to expire verification: %d", expireRec.Code)
	}

	// Check expired verification -> 404 verification_not_found per spec §7.2
	checkPayload, _ := json.Marshal(map[string]string{"code": "123456"})
	checkReq := httptest.NewRequest(http.MethodPost, "/verifications/"+verID+"/check", bytes.NewReader(checkPayload))
	checkReq.Header.Set("Content-Type", "application/json")
	checkRec := httptest.NewRecorder()
	router.ServeHTTP(checkRec, checkReq)

	if checkRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for expired verification, got %d: %s", checkRec.Code, checkRec.Body.String())
	}
	var errResp map[string]any
	_ = json.Unmarshal(checkRec.Body.Bytes(), &errResp)
	errObj := errResp["error"].(map[string]any)
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

func seedRequestLogs(t *testing.T, store core.Store, projectID string, n int) []*core.RequestLog {
	t.Helper()
	ctx := context.Background()
	logs := make([]*core.RequestLog, 0, n)
	for i := 0; i < n; i++ {
		l := &core.RequestLog{
			ID:             core.NewRequestLogID(),
			ProjectID:      projectID,
			Adapter:        "twilio",
			Method:         http.MethodPost,
			Path:           "/2010-04-01/Accounts/AC123/Messages.json",
			RequestHeaders: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			RequestBody:    []byte("To=%2B15551234567&Body=Hello"),
			ResponseStatus: http.StatusCreated,
			ResponseBody:   []byte(`{"sid":"SM123"}`),
			DurationMS:     int64(i + 1),
			CreatedAt:      time.Now(),
		}
		if err := store.CreateRequestLog(ctx, l); err != nil {
			t.Fatalf("CreateRequestLog: %v", err)
		}
		logs = append(logs, l)
	}
	return logs
}

func TestHandlers_RequestLogs(t *testing.T) {
	router, _, store := setupTestRouter(t, nil, nil)
	ctx := context.Background()

	prjA := &core.Project{ID: core.NewProjectID(), Name: "reqlog-a", Settings: map[string]interface{}{}, CreatedAt: time.Now()}
	if err := store.CreateProject(ctx, prjA); err != nil {
		t.Fatalf("CreateProject A: %v", err)
	}
	prjB := &core.Project{ID: core.NewProjectID(), Name: "reqlog-b", Settings: map[string]interface{}{}, CreatedAt: time.Now()}
	if err := store.CreateProject(ctx, prjB); err != nil {
		t.Fatalf("CreateProject B: %v", err)
	}
	logs := seedRequestLogs(t, store, prjA.ID, 3)

	t.Run("list returns all logs with JSON content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/requests?project="+prjA.ID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		items, ok := resp["logs"].([]any)
		if !ok || len(items) != 3 {
			t.Fatalf("expected 3 logs, got %v (%s)", resp["logs"], rec.Body.String())
		}
		first, _ := items[0].(map[string]any)
		for _, key := range []string{"id", "project_id", "adapter", "method", "path", "request_body", "response_status", "response_body", "duration_ms", "created_at"} {
			if _, ok := first[key]; !ok {
				t.Errorf("expected key %q in log, got %v", key, first)
			}
		}
		if first["project_id"] != prjA.ID {
			t.Errorf("expected project_id %s, got %v", prjA.ID, first["project_id"])
		}
	})

	t.Run("list respects limit and cursor", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/requests?project="+prjA.ID+"&limit=2", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		items, ok := resp["logs"].([]any)
		if !ok || len(items) != 2 {
			t.Fatalf("expected 2 logs with limit=2, got %v (%s)", resp["logs"], rec.Body.String())
		}
		cursor, _ := resp["next_cursor"].(string)
		if cursor == "" {
			t.Fatalf("expected non-empty next_cursor, got %v", resp)
		}

		req2 := httptest.NewRequest(http.MethodGet, "/requests?project="+prjA.ID+"&limit=2&cursor="+cursor, nil)
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Fatalf("expected status 200 on second page, got %d: %s", rec2.Code, rec2.Body.String())
		}
		var resp2 map[string]any
		if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
			t.Fatalf("failed to decode second page: %v", err)
		}
		items2, ok := resp2["logs"].([]any)
		if !ok || len(items2) != 1 {
			t.Fatalf("expected 1 log on second page, got %v (%s)", resp2["logs"], rec2.Body.String())
		}
	})

	t.Run("get by id returns the log", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/requests/"+logs[0].ID+"?project="+prjA.ID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["id"] != logs[0].ID {
			t.Errorf("expected id %s, got %v", logs[0].ID, resp["id"])
		}
		if resp["adapter"] != "twilio" {
			t.Errorf("expected adapter twilio, got %v", resp["adapter"])
		}
	})

	t.Run("get nonexistent returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/requests/req_00000000000000000000000000?project="+prjA.ID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		errObj, ok := resp["error"].(map[string]any)
		if !ok {
			t.Fatalf("expected 'error' object, got %v", resp)
		}
		if errObj["code"] != "not_found" {
			t.Errorf("expected code not_found, got %v", errObj["code"])
		}
	})

	t.Run("get from another project returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/requests/"+logs[0].ID+"?project="+prjB.ID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404 for cross-project access, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func sendTestSMS(t *testing.T, router http.Handler, from, to, body string) map[string]any {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{"from": from, "to": to, "body": body})
	req := httptest.NewRequest(http.MethodPost, "/sms", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send SMS: expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("send SMS: failed to decode response: %v", err)
	}
	return resp
}

func startTestVerification(t *testing.T, router http.Handler, to, channel string) (id, code string) {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{"to": to, "channel": channel})
	req := httptest.NewRequest(http.MethodPost, "/verifications", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("start verification: expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("start verification: failed to decode response: %v", err)
	}
	verObj, ok := resp["verification"].(map[string]any)
	if !ok {
		t.Fatalf("start verification: no verification in response: %v", resp)
	}
	return verObj["id"].(string), verObj["code"].(string)
}

func TestHandlers_GetLatestOTP(t *testing.T) {
	t.Run("extracted code from plain SMS", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		sendTestSMS(t, router, "+15555550100", "+15005550006", "Your code is 445566")

		req := httptest.NewRequest(http.MethodGet, "/otp/latest?to=%2B15005550006", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["code"] != "445566" {
			t.Errorf("expected code 445566, got %v", resp["code"])
		}
		if resp["source"] != "extracted" {
			t.Errorf("expected source extracted, got %v", resp["source"])
		}
		if _, ok := resp["message_id"].(string); !ok {
			t.Errorf("expected message_id, got %v", resp)
		}
		if _, ok := resp["verification_id"]; ok {
			t.Errorf("expected no verification_id, got %v", resp)
		}
	})

	t.Run("verification code with linkage", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		verID, verCode := startTestVerification(t, router, "+15005550006", "sms")

		req := httptest.NewRequest(http.MethodGet, "/otp/latest?to=%2B15005550006", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["code"] != verCode {
			t.Errorf("expected code %s, got %v", verCode, resp["code"])
		}
		if resp["source"] != "verification" {
			t.Errorf("expected source verification, got %v", resp["source"])
		}
		if resp["verification_id"] != verID {
			t.Errorf("expected verification_id %s, got %v", verID, resp["verification_id"])
		}
	})

	t.Run("newer verification beats older SMS", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		sendTestSMS(t, router, "+15555550100", "+15005550006", "Your code is 445566")
		_, verCode := startTestVerification(t, router, "+15005550006", "sms")

		req := httptest.NewRequest(http.MethodGet, "/otp/latest?to=%2B15005550006", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["code"] != verCode || resp["source"] != "verification" {
			t.Errorf("expected verification code %s, got %v", verCode, resp)
		}
	})

	t.Run("newer SMS beats older verification", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		startTestVerification(t, router, "+15005550006", "sms")
		sendTestSMS(t, router, "+15555550100", "+15005550006", "Your code is 445566")

		req := httptest.NewRequest(http.MethodGet, "/otp/latest?to=%2B15005550006", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["code"] != "445566" || resp["source"] != "extracted" {
			t.Errorf("expected extracted code 445566, got %v", resp)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/otp/latest?to=%2B15005550010", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing to returns 400", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/otp/latest", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("future since returns 404", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		sendTestSMS(t, router, "+15555550100", "+15005550006", "Your code is 445566")
		req := httptest.NewRequest(http.MethodGet, "/otp/latest?to=%2B15005550006&since=2999-01-01T00%3A00%3A00Z", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlers_GetLatestEmail(t *testing.T) {
	sendTestEmail := func(t *testing.T, router http.Handler, to, text string) {
		t.Helper()
		payload, _ := json.Marshal(map[string]any{
			"from":    "app@example.com",
			"to":      []string{to},
			"subject": "Verify",
			"text":    text,
		})
		req := httptest.NewRequest(http.MethodPost, "/email", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("send email: expected status 201, got %d: %s", rec.Code, rec.Body.String())
		}
	}

	t.Run("latest email with codes and links", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		sendTestEmail(t, router, "user@example.com", "Your code is 778899, verify at https://example.com/verify")

		req := httptest.NewRequest(http.MethodGet, "/emails/latest?to=user%40example.com", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		codes, _ := resp["codes"].([]any)
		if len(codes) != 1 || codes[0] != "778899" {
			t.Errorf("expected codes [778899], got %v", resp["codes"])
		}
		links, _ := resp["links"].([]any)
		if len(links) != 1 || links[0] != "https://example.com/verify" {
			t.Errorf("expected verify link, got %v", resp["links"])
		}
		if resp["primary_link"] != "https://example.com/verify" {
			t.Errorf("expected primary_link, got %v", resp["primary_link"])
		}
		if _, ok := resp["message"].(map[string]any); !ok {
			t.Errorf("expected message object, got %v", resp)
		}
	})

	t.Run("newest email wins", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		sendTestEmail(t, router, "user@example.com", "Your code is 111111")
		sendTestEmail(t, router, "user@example.com", "Your code is 222222")

		req := httptest.NewRequest(http.MethodGet, "/emails/latest?to=user%40example.com", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		codes, _ := resp["codes"].([]any)
		if len(codes) != 1 || codes[0] != "222222" {
			t.Errorf("expected codes [222222], got %v", resp["codes"])
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/emails/latest?to=nobody%40example.com", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing to returns 400", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/emails/latest", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlers_WaitForMessage(t *testing.T) {
	t.Run("returns existing match immediately", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		sendTestSMS(t, router, "+15555550100", "+15005550006", "Wait for me")

		req := httptest.NewRequest(http.MethodGet, "/messages/wait?to=%2B15005550006&since=2020-01-01T00%3A00%3A00Z&timeout=1", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("timeout returns 408 wait_timeout", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/messages/wait?to=%2B15005550010&since=2999-01-01T00%3A00%3A00Z&timeout=1", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusRequestTimeout {
			t.Fatalf("expected status 408, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		errObj, ok := resp["error"].(map[string]any)
		if !ok {
			t.Fatalf("expected 'error' object, got %v", resp)
		}
		if errObj["code"] != "wait_timeout" {
			t.Errorf("expected code wait_timeout, got %v", errObj["code"])
		}
	})
}

func TestHandlers_ExpireVerification(t *testing.T) {
	t.Run("expire moves to expired", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		verID, _ := startTestVerification(t, router, "+15005550006", "sms")

		req := httptest.NewRequest(http.MethodPost, "/verifications/"+verID+"/expire", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["status"] != "expired" {
			t.Errorf("expected status expired, got %v", resp["status"])
		}
	})

	t.Run("expire unknown returns 404", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		req := httptest.NewRequest(http.MethodPost, "/verifications/vrf_00000000000000000000000000/expire", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlers_DeleteMessages(t *testing.T) {
	t.Run("requires explicit project", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		req := httptest.NewRequest(http.MethodDelete, "/messages", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("deletes project messages", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		sendTestSMS(t, router, "+15555550100", "+15005550006", "Delete me")

		listReq := httptest.NewRequest(http.MethodGet, "/projects", nil)
		listRec := httptest.NewRecorder()
		router.ServeHTTP(listRec, listReq)
		var listResp map[string]any
		if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
			t.Fatalf("list projects: %v", err)
		}
		projects, _ := listResp["projects"].([]any)
		if len(projects) == 0 {
			t.Fatalf("expected a project, got %v", listResp)
		}
		projectID := projects[0].(map[string]any)["id"].(string)

		delReq := httptest.NewRequest(http.MethodDelete, "/messages?project="+projectID, nil)
		delRec := httptest.NewRecorder()
		router.ServeHTTP(delRec, delReq)
		if delRec.Code != http.StatusNoContent {
			t.Fatalf("expected status 204, got %d: %s", delRec.Code, delRec.Body.String())
		}

		getReq := httptest.NewRequest(http.MethodGet, "/messages?project="+projectID, nil)
		getRec := httptest.NewRecorder()
		router.ServeHTTP(getRec, getReq)
		var getResp map[string]any
		_ = json.Unmarshal(getRec.Body.Bytes(), &getResp)
		if msgs, _ := getResp["messages"].([]any); len(msgs) != 0 {
			t.Errorf("expected no messages after delete, got %v", msgs)
		}
	})
}

func projectIDs(t *testing.T, router http.Handler) []string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list projects: %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("list projects: %v", err)
	}
	projects, _ := resp["projects"].([]any)
	ids := make([]string, 0, len(projects))
	for _, p := range projects {
		ids = append(ids, p.(map[string]any)["id"].(string))
	}
	return ids
}

func sendTestSMSAs(t *testing.T, router http.Handler, bearer, from, to, body string) {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{"from": from, "to": to, "body": body})
	req := httptest.NewRequest(http.MethodPost, "/sms", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send SMS as %q: expected status 201, got %d: %s", bearer, rec.Code, rec.Body.String())
	}
}

func messageCount(t *testing.T, router http.Handler, projectID string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/messages?project="+projectID, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list messages: %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("list messages: %v", err)
	}
	msgs, _ := resp["messages"].([]any)
	return len(msgs)
}

func TestHandlers_LinkCredential(t *testing.T) {
	link := func(t *testing.T, router http.Handler, projectID, provider, key string) *httptest.ResponseRecorder {
		t.Helper()
		payload, _ := json.Marshal(map[string]any{"provider": provider, "key": key})
		req := httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/credentials", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	t.Run("links new credential and routes traffic", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		sendTestSMSAs(t, router, "", "+15555550100", "+15005550006", "First")
		ids := projectIDs(t, router)
		if len(ids) != 1 {
			t.Fatalf("expected 1 project, got %v", ids)
		}

		rec := link(t, router, ids[0], "native", "LINKKEY123")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		sendTestSMSAs(t, router, "LINKKEY123", "+15555550100", "+15005550006", "Second")
		if got := projectIDs(t, router); len(got) != 1 {
			t.Fatalf("linking must not create a project, got %v", got)
		}
		if n := messageCount(t, router, ids[0]); n != 2 {
			t.Errorf("expected 2 messages in linked project, got %d", n)
		}
	})

	t.Run("moves an existing credential", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		sendTestSMSAs(t, router, "", "+15555550100", "+15005550006", "Default project")
		sendTestSMSAs(t, router, "MOVEKEY456", "+15555550100", "+15005550006", "Own project")
		ids := projectIDs(t, router)
		if len(ids) != 2 {
			t.Fatalf("expected 2 projects, got %v", ids)
		}

		var target, other string
		if messageCount(t, router, ids[0]) == 1 && messageCount(t, router, ids[1]) == 1 {
			target, other = ids[0], ids[1]
		} else {
			t.Fatalf("expected one message per project, got %v", ids)
		}

		rec := link(t, router, target, "native", "MOVEKEY456")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		sendTestSMSAs(t, router, "MOVEKEY456", "+15555550100", "+15005550006", "After move")
		if n := messageCount(t, router, target); n != 2 {
			t.Errorf("expected 2 messages in target project, got %d", n)
		}
		if n := messageCount(t, router, other); n != 1 {
			t.Errorf("expected 1 message left in old project, got %d", n)
		}
	})

	t.Run("unknown project returns 404", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		rec := link(t, router, "prj_00000000000000000000000000", "native", "K")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing provider and key returns 400", func(t *testing.T) {
		router, _, _ := setupTestRouter(t, nil, nil)
		sendTestSMSAs(t, router, "", "+15555550100", "+15005550006", "First")
		target := projectIDs(t, router)[0]
		for name, payload := range map[string]map[string]any{
			"missing provider": {"key": "K"},
			"missing key":      {"provider": "native"},
		} {
			raw, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/projects/"+target+"/credentials", bytes.NewReader(raw))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s: expected status 400, got %d: %s", name, rec.Code, rec.Body.String())
			}
		}
	})
}
