package twilio_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"

	"github.com/Aeomar999/CommPit/core"
)

// loadContractSpec loads a pinned spec for contract validation.
func loadContractSpec(t *testing.T, specFile string) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(filepath.Join("spec", specFile))
	if err != nil {
		t.Fatalf("load spec %s: %v", specFile, err)
	}
	return doc
}

// contractOperation resolves a spec operation by its exact path template and
// method. Templates are looked up directly instead of matched through a
// router: the bundled routers cannot match suffixed template segments such
// as Messages/{Sid}.json, while route matching itself is already covered by
// the adapter handler tests.
func contractOperation(t *testing.T, doc *openapi3.T, template, method string) *routers.Route {
	t.Helper()
	item := doc.Paths.Value(template)
	if item == nil {
		t.Fatalf("path %q not found in spec", template)
	}
	var op *openapi3.Operation
	switch method {
	case http.MethodGet:
		op = item.Get
	case http.MethodPost:
		op = item.Post
	case http.MethodDelete:
		op = item.Delete
	}
	if op == nil {
		t.Fatalf("method %s not found on path %q in spec", method, template)
	}
	return &routers.Route{Spec: doc, Path: template, PathItem: item, Method: method, Operation: op}
}

// assertContractResponse performs a live adapter request and validates the
// response status and body against the pinned spec operation.
func assertContractResponse(t *testing.T, doc *openapi3.T, handler http.Handler, template, method, path string, form url.Values) {
	t.Helper()
	status, body := goldenCall(handler, method, path, form)

	req := httptest.NewRequest(method, path, nil)
	reqInput := &openapi3filter.RequestValidationInput{
		Request: req,
		Route:   contractOperation(t, doc, template, method),
	}
	respInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: reqInput,
		Status:                 status,
		Header:                 http.Header{"Content-Type": []string{"application/json"}},
		Options:                &openapi3filter.Options{MultiError: true},
	}
	respInput.SetBodyBytes([]byte(body))
	if err := openapi3filter.ValidateResponse(context.Background(), respInput); err != nil {
		t.Errorf("contract violation %s %s (status %d):\n%v\nbody: %s", method, path, status, err, body)
	}
}

func TestTwilio_ContractMessages(t *testing.T) {
	handler, _ := setupGoldenTwilio(t)
	doc := loadContractSpec(t, "twilio_api_v2010.json")
	base := "/2010-04-01/Accounts/" + testAccountSID
	listTemplate := "/2010-04-01/Accounts/{AccountSid}/Messages.json"
	fetchTemplate := "/2010-04-01/Accounts/{AccountSid}/Messages/{Sid}.json"

	// Seed one message and capture its live SID for fetch validation.
	createForm := url.Values{"To": {testTo}, "From": {testFrom}, "Body": {"Contract check"}}
	_, created := goldenCall(handler, http.MethodPost, base+"/Messages.json", createForm)
	var createdResp map[string]any
	if err := json.Unmarshal([]byte(created), &createdResp); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	sid, _ := createdResp["sid"].(string)

	cases := []struct {
		name     string
		template string
		method   string
		path     string
		form     url.Values
	}{
		{name: "create", template: listTemplate, method: http.MethodPost, path: base + "/Messages.json", form: createForm},
		{name: "fetch", template: fetchTemplate, method: http.MethodGet, path: base + "/Messages/" + sid + ".json"},
		{name: "list", template: listTemplate, method: http.MethodGet, path: base + "/Messages.json"},
		{name: "list filtered and paged", template: listTemplate, method: http.MethodGet, path: base + "/Messages.json?To=" + url.QueryEscape(testTo) + "&PageSize=10&Page=0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertContractResponse(t, doc, handler, tc.template, tc.method, tc.path, tc.form)
		})
	}
}

func TestTwilio_ContractVerify(t *testing.T) {
	handler, store := setupGoldenTwilio(t)
	doc := loadContractSpec(t, "twilio_verify_v2.json")

	_, svcCreated := goldenCall(handler, http.MethodPost, "/v2/Services", url.Values{"FriendlyName": {"contract"}})
	var svcResp map[string]any
	if err := json.Unmarshal([]byte(svcCreated), &svcResp); err != nil {
		t.Fatalf("decode service: %v", err)
	}
	serviceSid, _ := svcResp["sid"].(string)
	svcBase := "/v2/Services/" + serviceSid

	_, verCreated := goldenCall(handler, http.MethodPost, svcBase+"/Verifications", url.Values{"To": {testTo}, "Channel": {"sms"}})
	var verResp map[string]any
	if err := json.Unmarshal([]byte(verCreated), &verResp); err != nil {
		t.Fatalf("decode verification: %v", err)
	}
	verifySid, _ := verResp["sid"].(string)

	projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "twilio", testAccountSID)
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	v, err := store.GetVerificationByProviderRef(context.Background(), projectID, verifySid)
	if err != nil {
		t.Fatalf("GetVerificationByProviderRef: %v", err)
	}
	wrong := "000000"
	if wrong == v.Code {
		wrong = "111111"
	}
	// Approve this one so the approved shape is exercised via fetch below.
	// The "verification check" case targets the newer pending verification
	// created by the "verification create" case, which runs first.
	_, _ = goldenCall(handler, http.MethodPost, svcBase+"/VerificationCheck", url.Values{"To": {testTo}, "Code": {v.Code}})

	cases := []struct {
		name     string
		template string
		method   string
		path     string
		form     url.Values
	}{
		{name: "service create", template: "/v2/Services", method: http.MethodPost, path: "/v2/Services", form: url.Values{"FriendlyName": {"contract-2"}}},
		{name: "service fetch", template: "/v2/Services/{Sid}", method: http.MethodGet, path: "/v2/Services/" + serviceSid},
		{name: "verification create", template: "/v2/Services/{ServiceSid}/Verifications", method: http.MethodPost, path: svcBase + "/Verifications", form: url.Values{"To": {"+15005550007"}, "Channel": {"sms"}}},
		{name: "verification fetch", template: "/v2/Services/{ServiceSid}/Verifications/{Sid}", method: http.MethodGet, path: svcBase + "/Verifications/" + verifySid},
		{name: "verification check", template: "/v2/Services/{ServiceSid}/VerificationCheck", method: http.MethodPost, path: svcBase + "/VerificationCheck", form: url.Values{"To": {testTo}, "Code": {wrong}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertContractResponse(t, doc, handler, tc.template, tc.method, tc.path, tc.form)
		})
	}
}
