package twilio_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/adapters/twilio"
	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/sim"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

const (
	testAccountSID = "AC1234567890abcdef1234567890abcd"
	testAuthToken  = "testauthtoken1234567890abcdef"
	testFrom       = "+15555550100"
	testTo         = "+15005550006"
)

var sidPattern = regexp.MustCompile(`^SM[0-9a-f]{32}$`)

func setupTestTwilio(t *testing.T) (http.Handler, core.Store) {
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
	handler := kit.Wrap(twilio.New(svc), twilio.Extractor(), adapterkit.WithRequired(true))
	return handler, store
}

func twilioRequest(t *testing.T, method, target string, form url.Values, withAuth bool) *http.Request {
	t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, target, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if withAuth {
		req.SetBasicAuth(testAccountSID, testAuthToken)
	}
	return req
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v (%s)", err, rec.Body.String())
	}
	return resp
}

func createTwilioMessage(t *testing.T, handler http.Handler, to, from, body string) map[string]any {
	t.Helper()
	form := url.Values{"To": {to}, "From": {from}, "Body": {body}}
	req := twilioRequest(t, http.MethodPost, "/2010-04-01/Accounts/"+testAccountSID+"/Messages.json", form, true)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	return decodeJSON(t, rec)
}

func TestTwilio_CreateMessage(t *testing.T) {
	handler, _ := setupTestTwilio(t)

	form := url.Values{"To": {testTo}, "From": {testFrom}, "Body": {"Hello from mocksms"}}
	req := twilioRequest(t, http.MethodPost, "/2010-04-01/Accounts/"+testAccountSID+"/Messages.json", form, true)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	resp := decodeJSON(t, rec)
	sid, _ := resp["sid"].(string)
	if !sidPattern.MatchString(sid) {
		t.Errorf("expected sid SM+32 hex, got %q", sid)
	}
	if resp["account_sid"] != testAccountSID {
		t.Errorf("expected account_sid %s, got %v", testAccountSID, resp["account_sid"])
	}
	if resp["to"] != testTo {
		t.Errorf("expected to %s, got %v", testTo, resp["to"])
	}
	if resp["from"] != testFrom {
		t.Errorf("expected from %s, got %v", testFrom, resp["from"])
	}
	if resp["body"] != "Hello from mocksms" {
		t.Errorf("expected body echoed, got %v", resp["body"])
	}
	if resp["status"] != "queued" {
		t.Errorf("expected status queued, got %v", resp["status"])
	}
	if resp["num_segments"] != "1" {
		t.Errorf("expected num_segments 1, got %v", resp["num_segments"])
	}
	if resp["num_media"] != "0" {
		t.Errorf("expected num_media 0, got %v", resp["num_media"])
	}
	if resp["direction"] != "outbound-api" {
		t.Errorf("expected direction outbound-api, got %v", resp["direction"])
	}
	if resp["api_version"] != "2010-04-01" {
		t.Errorf("expected api_version 2010-04-01, got %v", resp["api_version"])
	}
	if _, ok := resp["date_created"]; !ok {
		t.Errorf("expected date_created, got %v", resp)
	}
	if dateSent, ok := resp["date_sent"]; !ok || dateSent != nil {
		t.Errorf("expected date_sent null while queued, got %v", resp["date_sent"])
	}
	uri, _ := resp["uri"].(string)
	if !strings.Contains(uri, sid) || !strings.Contains(uri, testAccountSID) {
		t.Errorf("expected uri to contain account and message sid, got %q", uri)
	}
}

func TestTwilio_CreateStoresStatusCallback(t *testing.T) {
	handler, store := setupTestTwilio(t)

	form := url.Values{
		"To":             {testTo},
		"From":           {testFrom},
		"Body":           {"Callback me"},
		"StatusCallback": {"https://example.com/twilio-status"},
	}
	req := twilioRequest(t, http.MethodPost, "/2010-04-01/Accounts/"+testAccountSID+"/Messages.json", form, true)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := decodeJSON(t, rec)
	sid, _ := resp["sid"].(string)

	projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "twilio", testAccountSID)
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	msgs, _, err := store.ListMessages(context.Background(), projectID, core.MessageFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	var found *core.Message
	for _, m := range msgs {
		if m.ProviderRef == sid {
			found = m
		}
	}
	if found == nil {
		t.Fatalf("message with provider_ref %s not stored", sid)
	}
	if found.CallbackURL == nil || *found.CallbackURL != "https://example.com/twilio-status" {
		t.Errorf("expected StatusCallback stored, got %v", found.CallbackURL)
	}
}

func TestTwilio_CreateErrors(t *testing.T) {
	cases := []struct {
		name       string
		to         string
		from       string
		body       string
		auth       bool
		wantStatus int
		wantCode   float64
	}{
		{name: "missing To", from: testFrom, body: "hi", auth: true, wantStatus: http.StatusBadRequest, wantCode: 21604},
		{name: "missing From", to: testTo, body: "hi", auth: true, wantStatus: http.StatusBadRequest, wantCode: 21606},
		{name: "missing Body", to: testTo, from: testFrom, auth: true, wantStatus: http.StatusBadRequest, wantCode: 21602},
		{name: "invalid To", to: "+15005550001", from: testFrom, body: "hi", auth: true, wantStatus: http.StatusBadRequest, wantCode: 21211},
		{name: "unroutable To", to: "+15005550002", from: testFrom, body: "hi", auth: true, wantStatus: http.StatusBadRequest, wantCode: 21612},
		{name: "not sms capable", to: "+15005550009", from: testFrom, body: "hi", auth: true, wantStatus: http.StatusBadRequest, wantCode: 21614},
		{name: "no credentials", to: testTo, from: testFrom, body: "hi", auth: false, wantStatus: http.StatusUnauthorized, wantCode: 20003},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler, _ := setupTestTwilio(t)
			form := url.Values{}
			if tc.to != "" {
				form.Set("To", tc.to)
			}
			if tc.from != "" {
				form.Set("From", tc.from)
			}
			if tc.body != "" {
				form.Set("Body", tc.body)
			}
			req := twilioRequest(t, http.MethodPost, "/2010-04-01/Accounts/"+testAccountSID+"/Messages.json", form, tc.auth)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
			resp := decodeJSON(t, rec)
			if resp["code"] != tc.wantCode {
				t.Errorf("expected Twilio code %v, got %v (%s)", tc.wantCode, resp["code"], rec.Body.String())
			}
			if _, ok := resp["more_info"]; !ok {
				t.Errorf("expected more_info in error, got %v", resp)
			}
			if resp["status"] != float64(tc.wantStatus) {
				t.Errorf("expected status field %d, got %v", tc.wantStatus, resp["status"])
			}
		})
	}
}

func TestTwilio_CreateUnsubscribed(t *testing.T) {
	handler, store := setupTestTwilio(t)

	resolver := core.NewProjectResolver(store)
	projectID, err := resolver.Resolve(context.Background(), "twilio", testAccountSID)
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	if err := store.CreateUnsubscribe(context.Background(), &core.Unsubscribe{
		ProjectID: projectID,
		Number:    testTo,
		At:        time.Now(),
	}); err != nil {
		t.Fatalf("CreateUnsubscribe: %v", err)
	}

	form := url.Values{"To": {testTo}, "From": {testFrom}, "Body": {"hello again"}}
	req := twilioRequest(t, http.MethodPost, "/2010-04-01/Accounts/"+testAccountSID+"/Messages.json", form, true)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if resp := decodeJSON(t, rec); resp["code"] != float64(21610) {
		t.Errorf("expected Twilio code 21610, got %v", resp["code"])
	}
}

func TestTwilio_FetchMessage(t *testing.T) {
	handler, _ := setupTestTwilio(t)
	created := createTwilioMessage(t, handler, testTo, testFrom, "fetch me")
	sid := created["sid"].(string)

	t.Run("fetch returns the message", func(t *testing.T) {
		req := twilioRequest(t, http.MethodGet, "/2010-04-01/Accounts/"+testAccountSID+"/Messages/"+sid+".json", nil, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if resp := decodeJSON(t, rec); resp["sid"] != sid {
			t.Errorf("expected sid %s, got %v", sid, resp["sid"])
		}
	})

	t.Run("unknown sid returns 20404", func(t *testing.T) {
		req := twilioRequest(t, http.MethodGet, "/2010-04-01/Accounts/"+testAccountSID+"/Messages/SM00000000000000000000000000000000.json", nil, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
		}
		if resp := decodeJSON(t, rec); resp["code"] != float64(20404) {
			t.Errorf("expected Twilio code 20404, got %v", resp["code"])
		}
	})
}

func TestTwilio_ListMessages(t *testing.T) {
	handler, _ := setupTestTwilio(t)
	createTwilioMessage(t, handler, testTo, testFrom, "first")
	createTwilioMessage(t, handler, "+15005550007", testFrom, "second")
	createTwilioMessage(t, handler, testTo, testFrom, "third")

	list := func(t *testing.T, query string) map[string]any {
		t.Helper()
		req := twilioRequest(t, http.MethodGet, "/2010-04-01/Accounts/"+testAccountSID+"/Messages.json"+query, nil, true)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		return decodeJSON(t, rec)
	}

	t.Run("lists all with paging envelope", func(t *testing.T) {
		resp := list(t, "")
		if resp["total"] != float64(3) {
			t.Errorf("expected total 3, got %v", resp["total"])
		}
		msgs, ok := resp["messages"].([]any)
		if !ok || len(msgs) != 3 {
			t.Fatalf("expected 3 messages, got %v", resp["messages"])
		}
		for _, key := range []string{"page", "num_pages", "page_size", "start", "end", "first_page_uri", "uri"} {
			if _, ok := resp[key]; !ok {
				t.Errorf("expected paging key %q, got %v", key, resp)
			}
		}
		if resp["next_page_uri"] != nil {
			t.Errorf("expected null next_page_uri on single page, got %v", resp["next_page_uri"])
		}
	})

	t.Run("filters by To", func(t *testing.T) {
		resp := list(t, "?To="+url.QueryEscape(testTo))
		if resp["total"] != float64(2) {
			t.Errorf("expected total 2 for To filter, got %v", resp["total"])
		}
	})

	t.Run("paginates with PageSize and Page", func(t *testing.T) {
		first := list(t, "?PageSize=2&Page=0")
		msgs, _ := first["messages"].([]any)
		if len(msgs) != 2 {
			t.Fatalf("expected 2 messages on page 0, got %v", first["messages"])
		}
		if first["next_page_uri"] == nil {
			t.Fatalf("expected next_page_uri on page 0, got %v", first)
		}
		second := list(t, "?PageSize=2&Page=1")
		msgs2, _ := second["messages"].([]any)
		if len(msgs2) != 1 {
			t.Fatalf("expected 1 message on page 1, got %v", second["messages"])
		}
		if second["next_page_uri"] != nil {
			t.Errorf("expected null next_page_uri on last page, got %v", second["next_page_uri"])
		}
	})

	t.Run("filters by DateSent", func(t *testing.T) {
		today := time.Now().UTC().Format("2006-01-02")
		resp := list(t, "?DateSent="+today)
		if resp["total"] != float64(3) {
			t.Errorf("expected total 3 for today, got %v", resp["total"])
		}
		resp = list(t, "?DateSent=2000-01-01")
		if resp["total"] != float64(0) {
			t.Errorf("expected total 0 for 2000-01-01, got %v", resp["total"])
		}
	})
}

func TestTwilio_RequestLogged(t *testing.T) {
	handler, store := setupTestTwilio(t)
	createTwilioMessage(t, handler, testTo, testFrom, "logged")

	projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "twilio", testAccountSID)
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
	if logs[0].Adapter != "twilio" {
		t.Errorf("expected adapter twilio, got %q", logs[0].Adapter)
	}
	if !strings.Contains(logs[0].Path, "Messages.json") {
		t.Errorf("expected path with Messages.json, got %q", logs[0].Path)
	}
	if logs[0].ResponseStatus != http.StatusCreated {
		t.Errorf("expected response status 201, got %d", logs[0].ResponseStatus)
	}
}

func TestTwilio_WriteErrorMapping(t *testing.T) {
	adapter := twilio.New(nil)
	cases := []struct {
		name       string
		err        *core.Error
		wantStatus int
		wantCode   float64
	}{
		{"invalid number", core.NewInvalidNumber("bad", "to"), http.StatusBadRequest, 21211},
		{"invalid sender", core.NewInvalidSender("bad", "from"), http.StatusBadRequest, 21212},
		{"unroutable", core.NewUnroutable("bad", "to"), http.StatusBadRequest, 21612},
		{"not sms capable", core.NewNotSMSCapable("bad", "to"), http.StatusBadRequest, 21614},
		{"unsubscribed", core.NewUnsubscribed("stop", "to"), http.StatusBadRequest, 21610},
		{"rate limited", core.NewRateLimited("slow", "to"), http.StatusTooManyRequests, 20429},
		{"provider unavailable", core.NewProviderUnavailable("down", ""), http.StatusServiceUnavailable, 20503},
		{"not found", core.NewNotFound("missing", "id"), http.StatusNotFound, 20404},
		{"unauthorized", core.NewUnauthorized("auth"), http.StatusUnauthorized, 20003},
		{"internal", core.NewInternal("boom"), http.StatusInternalServerError, 20500},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			adapter.WriteError(rec, tc.err)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
			resp := decodeJSON(t, rec)
			if resp["code"] != tc.wantCode {
				t.Errorf("expected code %v, got %v", tc.wantCode, resp["code"])
			}
		})
	}
}
