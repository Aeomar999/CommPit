package termii_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/Aeomar999/CommPit/core"
)

var termiiGoldenUpdate = flag.Bool("update", false, "regenerate golden files")

// normalizeTermii replaces volatile values by JSON key: numeric message IDs,
// UUID pin IDs and generated PINs. Phone numbers share the digit space, so
// whole-body regexes are unsafe here.
func normalizeTermii(body map[string]any) map[string]any {
	out := make(map[string]any, len(body))
	for k, v := range body {
		switch k {
		case "message_id":
			out[k] = float64(1234567890)
		case "pinId":
			out[k] = "00000000-0000-0000-0000-000000000000"
		case "pin":
			out[k] = "000000"
		default:
			out[k] = v
		}
	}
	return out
}

func termiiGoldenPost(handler http.Handler, target string, payload map[string]any) (int, map[string]any) {
	payload["api_key"] = testAPIKey
	raw, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	req := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		panic("response is not JSON: " + rec.Body.String())
	}
	return rec.Code, resp
}

type termiiGoldenScenario struct {
	name   string
	method string
	path   string
	form   map[string]any
	run    func(t *testing.T, handler http.Handler, store core.Store) (status int, body map[string]any)
}

func termiiGoldenScenarios() []termiiGoldenScenario {
	return []termiiGoldenScenario{
		{
			name: "sms_send_single", method: http.MethodPost, path: "/api/sms/send",
			form: map[string]any{"to": testToNG, "from": testSender, "sms": "Hello golden"},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, map[string]any) {
				return termiiGoldenPost(handler, "/api/sms/send", map[string]any{
					"to": testToNG, "from": testSender, "sms": "Hello golden",
				})
			},
		},
		{
			name: "sms_send_array", method: http.MethodPost, path: "/api/sms/send",
			form: map[string]any{"to": []string{testToNG, testToNG2}, "from": testSender, "sms": "Batch golden"},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, map[string]any) {
				return termiiGoldenPost(handler, "/api/sms/send", map[string]any{
					"to": []string{testToNG, testToNG2}, "from": testSender, "sms": "Batch golden",
				})
			},
		},
		{
			name: "sms_send_bulk", method: http.MethodPost, path: "/api/sms/send/bulk",
			form: map[string]any{"to": []string{testToNG, testToNG2, testToNG3}, "from": testSender, "sms": "Bulk golden"},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, map[string]any) {
				return termiiGoldenPost(handler, "/api/sms/send/bulk", map[string]any{
					"to": []string{testToNG, testToNG2, testToNG3}, "from": testSender, "sms": "Bulk golden",
				})
			},
		},
		{
			name: "sms_send_missing_to", method: http.MethodPost, path: "/api/sms/send",
			form: map[string]any{"from": testSender, "sms": "No to"},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, map[string]any) {
				return termiiGoldenPost(handler, "/api/sms/send", map[string]any{
					"from": testSender, "sms": "No to",
				})
			},
		},
		{
			name: "sms_send_invalid_to", method: http.MethodPost, path: "/api/sms/send",
			form: map[string]any{"to": "+15005550001", "from": testSender, "sms": "Bad to"},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, map[string]any) {
				return termiiGoldenPost(handler, "/api/sms/send", map[string]any{
					"to": "+15005550001", "from": testSender, "sms": "Bad to",
				})
			},
		},
		{
			name: "sms_send_unlisted_sender", method: http.MethodPost, path: "/api/sms/send",
			form: map[string]any{"to": testToNG, "from": "BlockedSender", "sms": "Blocked"},
			run: func(t *testing.T, handler http.Handler, store core.Store) (int, map[string]any) {
				projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "termii", testAPIKey)
				if err != nil {
					t.Fatalf("resolve project: %v", err)
				}
				project, err := store.GetProject(context.Background(), projectID)
				if err != nil {
					t.Fatalf("GetProject: %v", err)
				}
				if project.Settings == nil {
					project.Settings = map[string]interface{}{}
				}
				project.Settings["termii.sender_allowlist"] = []interface{}{"AllowedSender"}
				if err := store.UpdateProject(context.Background(), project); err != nil {
					t.Fatalf("UpdateProject: %v", err)
				}
				return termiiGoldenPost(handler, "/api/sms/send", map[string]any{
					"to": testToNG, "from": "BlockedSender", "sms": "Blocked",
				})
			},
		},
		{
			name: "sms_number_send", method: http.MethodPost, path: "/api/sms/number/send",
			form: map[string]any{"to": testToNG, "from": "+2348012345678", "sms": "From a number"},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, map[string]any) {
				return termiiGoldenPost(handler, "/api/sms/number/send", map[string]any{
					"to": testToNG, "from": "+2348012345678", "sms": "From a number",
				})
			},
		},
		{
			name: "otp_send", method: http.MethodPost, path: "/api/sms/otp/send",
			form: map[string]any{"to": testToNG, "from": testSender, "message_text": "Your pin is < 1234 >", "pin_length": 6},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, map[string]any) {
				return termiiGoldenPost(handler, "/api/sms/otp/send", map[string]any{
					"to": testToNG, "from": testSender,
					"message_text": "Your pin is < 1234 >", "pin_length": 6,
				})
			},
		},
		{
			name: "otp_verify_wrong_pin", method: http.MethodPost, path: "/api/sms/otp/verify",
			form: map[string]any{"pin_id": "PINID", "pin": "000000"},
			run: func(t *testing.T, handler http.Handler, store core.Store) (int, map[string]any) {
				_, sent := termiiGoldenPost(handler, "/api/sms/otp/send", map[string]any{
					"to": testToNG, "from": testSender,
				})
				projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "termii", testAPIKey)
				if err != nil {
					t.Fatalf("resolve project: %v", err)
				}
				v, err := store.GetVerificationByProviderRef(context.Background(), projectID, sent["pinId"].(string))
				if err != nil {
					t.Fatalf("GetVerificationByProviderRef: %v", err)
				}
				wrong := "000000"
				if wrong == v.Code {
					wrong = "111111"
				}
				return termiiGoldenPost(handler, "/api/sms/otp/verify", map[string]any{
					"pin_id": sent["pinId"].(string), "pin": wrong,
				})
			},
		},
		{
			name: "otp_verify_ok", method: http.MethodPost, path: "/api/sms/otp/verify",
			form: map[string]any{"pin_id": "PINID", "pin": "CODE"},
			run: func(t *testing.T, handler http.Handler, store core.Store) (int, map[string]any) {
				_, sent := termiiGoldenPost(handler, "/api/sms/otp/send", map[string]any{
					"to": testToNG, "from": testSender,
				})
				projectID, err := core.NewProjectResolver(store).Resolve(context.Background(), "termii", testAPIKey)
				if err != nil {
					t.Fatalf("resolve project: %v", err)
				}
				v, err := store.GetVerificationByProviderRef(context.Background(), projectID, sent["pinId"].(string))
				if err != nil {
					t.Fatalf("GetVerificationByProviderRef: %v", err)
				}
				return termiiGoldenPost(handler, "/api/sms/otp/verify", map[string]any{
					"pin_id": sent["pinId"].(string), "pin": v.Code,
				})
			},
		},
		{
			name: "otp_generate", method: http.MethodPost, path: "/api/sms/otp/generate",
			form: map[string]any{"pin_type": "NUMERIC", "pin_length": 6},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, map[string]any) {
				return termiiGoldenPost(handler, "/api/sms/otp/generate", map[string]any{
					"pin_type": "NUMERIC", "pin_length": 6,
				})
			},
		},
		{
			name: "email_otp_send", method: http.MethodPost, path: "/api/email/otp/send",
			form: map[string]any{"email_address": "user@example.com", "code": "731946"},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, map[string]any) {
				return termiiGoldenPost(handler, "/api/email/otp/send", map[string]any{
					"email_address": "user@example.com", "code": "731946",
				})
			},
		},
		{
			name: "email_otp_bad_address", method: http.MethodPost, path: "/api/email/otp/send",
			form: map[string]any{"email_address": "not-an-email", "code": "731946"},
			run: func(t *testing.T, handler http.Handler, _ core.Store) (int, map[string]any) {
				return termiiGoldenPost(handler, "/api/email/otp/send", map[string]any{
					"email_address": "not-an-email", "code": "731946",
				})
			},
		},
	}
}

func TestTermii_Golden(t *testing.T) {
	for _, sc := range termiiGoldenScenarios() {
		t.Run(sc.name, func(t *testing.T) {
			handler, store := setupTestTermii(t)
			status, body := sc.run(t, handler, store)

			record := map[string]any{
				"request": map[string]any{
					"method": sc.method,
					"path":   sc.path,
				},
				"status": status,
				"body":   normalizeTermii(body),
			}
			if sc.form != nil {
				record["request"].(map[string]any)["form"] = sc.form
			}
			golden, err := json.MarshalIndent(record, "", "  ")
			if err != nil {
				t.Fatalf("encode golden: %v", err)
			}
			golden = append(golden, '\n')

			path := filepath.Join("testdata", sc.name+".golden.json")
			if *termiiGoldenUpdate {
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
