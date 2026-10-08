package twilio_test

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/adapters/twilio"
	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/sim"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

var goldenUpdate = flag.Bool("update", false, "regenerate golden files")

// volatilePatterns normalizes changing fields (SIDs, dates) so goldens are
// stable across runs. Clocks are fixed (see setupGoldenTwilio); SIDs and
// service timestamps are normalized here.
var volatilePatterns = []struct {
	re   *regexp.Regexp
	repl string
}{
	{regexp.MustCompile(`SM[0-9a-f]{32}`), "SM00000000000000000000000000000000"},
	{regexp.MustCompile(`VA[0-9a-f]{32}`), "VA00000000000000000000000000000000"},
	{regexp.MustCompile(`VE[0-9a-f]{32}`), "VE00000000000000000000000000000000"},
	// The account SID is normalized to Twilio's docs-conventional fake so
	// committed fixtures never contain a secret-like AC+32 literal.
	{regexp.MustCompile(`AC[0-9a-fA-F]{32}`), "ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"},
	{regexp.MustCompile(`(Mon|Tue|Wed|Thu|Fri|Sat|Sun), \d{2} (Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d{4} \d{2}:\d{2}:\d{2} \+\d{4}`), "DATE"},
	{regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`), "DATE"},
}

func normalizeGolden(body string) string {
	for _, p := range volatilePatterns {
		body = p.re.ReplaceAllString(body, p.repl)
	}
	return body
}

func setupGoldenTwilio(t *testing.T) (http.Handler, core.Store) {
	t.Helper()
	store, err := sqlite.NewStore(":memory:", 1)
	if err != nil {
		t.Fatalf("sqlite.NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	clock := core.NewFakeClock()
	clock.Set(time.Date(2026, time.October, 8, 12, 0, 0, 0, time.UTC))

	eventBus := bus.NewEventBus()
	resolver := core.NewProjectResolver(store)
	svc := core.NewService(core.ServiceConfig{
		Store:     store,
		BlobStore: store,
		Bus:       eventBus,
		Simulator: sim.NewSimulator(),
		Clock:     clock,
		Resolver:  resolver,
	})

	sink := adapterkit.RequestLogSinkFunc(func(ctx context.Context, entry *core.RequestLog) error {
		return store.CreateRequestLog(ctx, entry)
	})
	kit := adapterkit.New(resolver, adapterkit.WithBus(eventBus), adapterkit.WithSink(sink))
	return kit.Wrap(twilio.New(svc), twilio.Extractor(), adapterkit.WithRequired(true)), store
}

// goldenCall performs one adapter request and returns status + raw body.
func goldenCall(handler http.Handler, method, target string, form url.Values) (int, string) {
	var body *strings.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.SetBasicAuth(testAccountSID, testAuthToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code, strings.TrimSpace(rec.Body.String())
}

// goldenScenario runs the request sequence for one fixture and reports the
// final request shown in the golden file.
type goldenScenario struct {
	name   string
	method string
	path   string
	form   url.Values
	run    func(t *testing.T, handler http.Handler, store core.Store) (status int, body string)
}

func goldenScenarios() []goldenScenario {
	msgPath := "/2010-04-01/Accounts/" + testAccountSID + "/Messages.json"
	scenarios := []goldenScenario{
		{
			name:   "messages_create_success",
			method: http.MethodPost, path: msgPath,
			form: url.Values{"To": {testTo}, "From": {testFrom}, "Body": {"Hello golden"}},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				return goldenCall(handler, http.MethodPost, msgPath, url.Values{"To": {testTo}, "From": {testFrom}, "Body": {"Hello golden"}})
			},
		},
		{
			name:   "messages_create_with_callback",
			method: http.MethodPost, path: msgPath,
			form: url.Values{"To": {testTo}, "From": {testFrom}, "Body": {"With callback"}, "StatusCallback": {"https://example.com/cb"}},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				return goldenCall(handler, http.MethodPost, msgPath, url.Values{"To": {testTo}, "From": {testFrom}, "Body": {"With callback"}, "StatusCallback": {"https://example.com/cb"}})
			},
		},
		{
			name:   "messages_create_missing_to",
			method: http.MethodPost, path: msgPath,
			form: url.Values{"From": {testFrom}, "Body": {"No to"}},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				return goldenCall(handler, http.MethodPost, msgPath, url.Values{"From": {testFrom}, "Body": {"No to"}})
			},
		},
		{
			name:   "messages_create_invalid_to",
			method: http.MethodPost, path: msgPath,
			form: url.Values{"To": {"+15005550001"}, "From": {testFrom}, "Body": {"Bad to"}},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				return goldenCall(handler, http.MethodPost, msgPath, url.Values{"To": {"+15005550001"}, "From": {testFrom}, "Body": {"Bad to"}})
			},
		},
		{
			name:   "messages_fetch_success",
			method: http.MethodGet, path: msgPath + "/{sid}",
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				_, created := goldenCall(handler, http.MethodPost, msgPath, url.Values{"To": {testTo}, "From": {testFrom}, "Body": {"Fetch me"}})
				var resp map[string]any
				if err := json.Unmarshal([]byte(created), &resp); err != nil {
					t.Fatalf("decode created: %v", err)
				}
				sid, _ := resp["sid"].(string)
				return goldenCall(handler, http.MethodGet, "/2010-04-01/Accounts/"+testAccountSID+"/Messages/"+sid+".json", nil)
			},
		},
		{
			name:   "messages_fetch_not_found",
			method: http.MethodGet, path: msgPath + "/{sid}",
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				return goldenCall(handler, http.MethodGet, "/2010-04-01/Accounts/"+testAccountSID+"/Messages/SM00000000000000000000000000000000.json", nil)
			},
		},
		{
			name:   "messages_list_all",
			method: http.MethodGet, path: msgPath,
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				for _, body := range []string{"one", "two", "three"} {
					goldenCall(handler, http.MethodPost, msgPath, url.Values{"To": {testTo}, "From": {testFrom}, "Body": {body}})
				}
				return goldenCall(handler, http.MethodGet, msgPath, nil)
			},
		},
		{
			name:   "messages_list_filter_to",
			method: http.MethodGet, path: msgPath,
			form: url.Values{"To": {testTo}},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				goldenCall(handler, http.MethodPost, msgPath, url.Values{"To": {testTo}, "From": {testFrom}, "Body": {"match"}})
				goldenCall(handler, http.MethodPost, msgPath, url.Values{"To": {"+15005550007"}, "From": {testFrom}, "Body": {"other"}})
				return goldenCall(handler, http.MethodGet, msgPath+"?To="+url.QueryEscape(testTo), nil)
			},
		},
		{
			name:   "messages_list_paged",
			method: http.MethodGet, path: msgPath,
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				for _, body := range []string{"one", "two", "three"} {
					goldenCall(handler, http.MethodPost, msgPath, url.Values{"To": {testTo}, "From": {testFrom}, "Body": {body}})
				}
				return goldenCall(handler, http.MethodGet, msgPath+"?PageSize=2&Page=0", nil)
			},
		},
	}

	svcPath := "/v2/Services"
	scenarios = append(scenarios,
		goldenScenario{
			name:   "verify_service_create",
			method: http.MethodPost, path: svcPath,
			form: url.Values{"FriendlyName": {"golden-svc"}, "CodeLength": {"6"}},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				return goldenCall(handler, http.MethodPost, svcPath, url.Values{"FriendlyName": {"golden-svc"}, "CodeLength": {"6"}})
			},
		},
		goldenScenario{
			name:   "verify_service_fetch",
			method: http.MethodGet, path: svcPath + "/{sid}",
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				_, created := goldenCall(handler, http.MethodPost, svcPath, url.Values{"FriendlyName": {"golden-svc"}})
				var resp map[string]any
				if err := json.Unmarshal([]byte(created), &resp); err != nil {
					t.Fatalf("decode created: %v", err)
				}
				sid, _ := resp["sid"].(string)
				return goldenCall(handler, http.MethodGet, svcPath+"/"+sid, nil)
			},
		},
		goldenScenario{
			name:   "verify_verification_create",
			method: http.MethodPost, path: svcPath + "/{sid}/Verifications",
			form: url.Values{"To": {testTo}, "Channel": {"sms"}},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				_, created := goldenCall(handler, http.MethodPost, svcPath, url.Values{"FriendlyName": {"golden-svc"}})
				var resp map[string]any
				if err := json.Unmarshal([]byte(created), &resp); err != nil {
					t.Fatalf("decode created: %v", err)
				}
				sid, _ := resp["sid"].(string)
				return goldenCall(handler, http.MethodPost, svcPath+"/"+sid+"/Verifications", url.Values{"To": {testTo}, "Channel": {"sms"}})
			},
		},
		goldenScenario{
			name:   "verify_check_wrong_code",
			method: http.MethodPost, path: svcPath + "/{sid}/VerificationCheck",
			form: url.Values{"To": {testTo}, "Code": {"000000"}},
			run: func(t *testing.T, handler http.Handler, store core.Store) (int, string) {
				_, created := goldenCall(handler, http.MethodPost, svcPath, url.Values{})
				var svcResp map[string]any
				if err := json.Unmarshal([]byte(created), &svcResp); err != nil {
					t.Fatalf("decode created: %v", err)
				}
				sid, _ := svcResp["sid"].(string)
				_, verCreated := goldenCall(handler, http.MethodPost, svcPath+"/"+sid+"/Verifications", url.Values{"To": {testTo}, "Channel": {"sms"}})
				var verResp map[string]any
				if err := json.Unmarshal([]byte(verCreated), &verResp); err != nil {
					t.Fatalf("decode verification: %v", err)
				}
				projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "twilio", testAccountSID)
				if err != nil {
					t.Fatalf("resolve project: %v", err)
				}
				v, err := store.GetVerificationByProviderRef(context.Background(), projectID, verResp["sid"].(string))
				if err != nil {
					t.Fatalf("GetVerificationByProviderRef: %v", err)
				}
				wrong := "000000"
				if wrong == v.Code {
					wrong = "111111"
				}
				return goldenCall(handler, http.MethodPost, svcPath+"/"+sid+"/VerificationCheck", url.Values{"To": {testTo}, "Code": {wrong}})
			},
		},
		goldenScenario{
			name:   "verify_check_approved",
			method: http.MethodPost, path: svcPath + "/{sid}/VerificationCheck",
			run: func(t *testing.T, handler http.Handler, store core.Store) (int, string) {
				_, created := goldenCall(handler, http.MethodPost, svcPath, url.Values{})
				var svcResp map[string]any
				if err := json.Unmarshal([]byte(created), &svcResp); err != nil {
					t.Fatalf("decode created: %v", err)
				}
				sid, _ := svcResp["sid"].(string)
				_, verCreated := goldenCall(handler, http.MethodPost, svcPath+"/"+sid+"/Verifications", url.Values{"To": {testTo}, "Channel": {"sms"}})
				var verResp map[string]any
				if err := json.Unmarshal([]byte(verCreated), &verResp); err != nil {
					t.Fatalf("decode verification: %v", err)
				}
				projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "twilio", testAccountSID)
				if err != nil {
					t.Fatalf("resolve project: %v", err)
				}
				v, err := store.GetVerificationByProviderRef(context.Background(), projectID, verResp["sid"].(string))
				if err != nil {
					t.Fatalf("GetVerificationByProviderRef: %v", err)
				}
				return goldenCall(handler, http.MethodPost, svcPath+"/"+sid+"/VerificationCheck", url.Values{"To": {testTo}, "Code": {v.Code}})
			},
		},
		goldenScenario{
			name:   "verify_check_max_attempts",
			method: http.MethodPost, path: svcPath + "/{sid}/VerificationCheck",
			run: func(t *testing.T, handler http.Handler, store core.Store) (int, string) {
				_, created := goldenCall(handler, http.MethodPost, svcPath, url.Values{})
				var svcResp map[string]any
				if err := json.Unmarshal([]byte(created), &svcResp); err != nil {
					t.Fatalf("decode created: %v", err)
				}
				sid, _ := svcResp["sid"].(string)
				_, verCreated := goldenCall(handler, http.MethodPost, svcPath+"/"+sid+"/Verifications", url.Values{"To": {testTo}, "Channel": {"sms"}})
				var verResp map[string]any
				if err := json.Unmarshal([]byte(verCreated), &verResp); err != nil {
					t.Fatalf("decode verification: %v", err)
				}
				projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "twilio", testAccountSID)
				if err != nil {
					t.Fatalf("resolve project: %v", err)
				}
				v, err := store.GetVerificationByProviderRef(context.Background(), projectID, verResp["sid"].(string))
				if err != nil {
					t.Fatalf("GetVerificationByProviderRef: %v", err)
				}
				wrong := "000000"
				if wrong == v.Code {
					wrong = "111111"
				}
				var status int
				var body string
				for i := 0; i < 5; i++ {
					status, body = goldenCall(handler, http.MethodPost, svcPath+"/"+sid+"/VerificationCheck", url.Values{"To": {testTo}, "Code": {wrong}})
				}
				return status, body
			},
		},
		goldenScenario{
			name:   "verify_verification_cancel",
			method: http.MethodPost, path: svcPath + "/{sid}/Verifications/{sid}",
			form: url.Values{"Status": {"canceled"}},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				_, created := goldenCall(handler, http.MethodPost, svcPath, url.Values{})
				var svcResp map[string]any
				if err := json.Unmarshal([]byte(created), &svcResp); err != nil {
					t.Fatalf("decode created: %v", err)
				}
				sid, _ := svcResp["sid"].(string)
				_, verCreated := goldenCall(handler, http.MethodPost, svcPath+"/"+sid+"/Verifications", url.Values{"To": {testTo}, "Channel": {"sms"}})
				var verResp map[string]any
				if err := json.Unmarshal([]byte(verCreated), &verResp); err != nil {
					t.Fatalf("decode verification: %v", err)
				}
				verSid, _ := verResp["sid"].(string)
				return goldenCall(handler, http.MethodPost, svcPath+"/"+sid+"/Verifications/"+verSid, url.Values{"Status": {"canceled"}})
			},
		},
		goldenScenario{
			name:   "verify_verification_fetch",
			method: http.MethodGet, path: svcPath + "/{sid}/Verifications/{sid}",
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, string) {
				_, created := goldenCall(handler, http.MethodPost, svcPath, url.Values{})
				var svcResp map[string]any
				if err := json.Unmarshal([]byte(created), &svcResp); err != nil {
					t.Fatalf("decode created: %v", err)
				}
				sid, _ := svcResp["sid"].(string)
				_, verCreated := goldenCall(handler, http.MethodPost, svcPath+"/"+sid+"/Verifications", url.Values{"To": {testTo}, "Channel": {"sms"}})
				var verResp map[string]any
				if err := json.Unmarshal([]byte(verCreated), &verResp); err != nil {
					t.Fatalf("decode verification: %v", err)
				}
				verSid, _ := verResp["sid"].(string)
				return goldenCall(handler, http.MethodGet, svcPath+"/"+sid+"/Verifications/"+verSid, nil)
			},
		},
	)
	return scenarios
}

func TestTwilio_Golden(t *testing.T) {
	for _, sc := range goldenScenarios() {
		t.Run(sc.name, func(t *testing.T) {
			handler, store := setupGoldenTwilio(t)
			status, body := sc.run(t, handler, store)

			var decoded any
			if err := json.Unmarshal([]byte(body), &decoded); err != nil {
				t.Fatalf("response is not JSON: %v (%s)", err, body)
			}
			pretty, err := json.MarshalIndent(decoded, "", "  ")
			if err != nil {
				t.Fatalf("re-encode response: %v", err)
			}
			record := map[string]any{
				"request": map[string]any{
					"method": sc.method,
					"path":   sc.path,
				},
				"status": status,
				"body":   json.RawMessage(pretty),
			}
			if sc.form != nil {
				record["request"].(map[string]any)["form"] = sc.form
			}
			golden, err := json.MarshalIndent(record, "", "  ")
			if err != nil {
				t.Fatalf("encode golden: %v", err)
			}
			golden = append(golden, '\n')
			// Normalize the whole record so request paths and URIs lose
			// their volatile SIDs and account credentials too.
			golden = []byte(normalizeGolden(string(golden)))

			path := filepath.Join("testdata", sc.name+".golden.json")
			if *goldenUpdate {
				if err := os.WriteFile(path, golden, 0644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden (run with -update to create): %v", err)
			}
			if diff := cmp.Diff(string(want), string(golden)); diff != "" {
				t.Errorf("golden mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
