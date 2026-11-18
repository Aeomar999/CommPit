package sendgrid_test

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

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/adapters/sendgrid"
	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/sim"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

var sendgridGoldenUpdate = flag.Bool("update", false, "regenerate golden files")

func normalizeSendGrid(body map[string]any) map[string]any {
	out := make(map[string]any, len(body))
	for k, v := range body {
		switch k {
		case "X-Message-Id":
			out[k] = "msg_normalized"
		default:
			out[k] = v
		}
	}
	return out
}

func sendGridGoldenPost(handler http.Handler, payload map[string]any) (int, map[string]any, http.Header) {
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v3/mail/send", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sg_test_key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec.Code, resp, rec.Header()
}

func setupGoldenSendGrid(t *testing.T) http.Handler {
	t.Helper()
	store, err := sqlite.NewStore(":memory:", 1)
	if err != nil {
		t.Fatal(err)
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
	return kit.Wrap(sendgrid.New(svc), sendgrid.Extractor(), adapterkit.WithRequired(true))
}

func TestSendGrid_Golden(t *testing.T) {
	handler := setupGoldenSendGrid(t)
	scenarios := []struct {
		name    string
		payload map[string]any
	}{
		{"mail_send_success", map[string]any{
			"personalizations": []map[string]any{{"to": []map[string]string{{"email": "alice@example.com"}}}},
			"from":             map[string]string{"email": "sender@example.com"},
			"subject":          "Hello",
			"content":          []map[string]string{{"type": "text/plain", "value": "hi"}},
		}},
		{"mail_send_missing_to", map[string]any{
			"personalizations": []map[string]any{{"to": []map[string]string{}}},
			"from":             map[string]string{"email": "sender@example.com"},
			"subject":          "Hello",
		}},
	}
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			status, body, headers := sendGridGoldenPost(handler, sc.payload)
			golden := map[string]any{"status": float64(status)}
			if body != nil {
				golden["body"] = body
			}
			if mid := headers.Get("X-Message-Id"); mid != "" {
				golden["x_message_id"] = "msg_normalized"
			}
			path := filepath.Join("testdata", sc.name+".golden.json")
			if *sendgridGoldenUpdate {
				raw, _ := json.MarshalIndent(golden, "", "  ")
				_ = os.MkdirAll("testdata", 0o755)
				_ = os.WriteFile(path, append(raw, '\n'), 0o644)
				return
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden: %v (run with -update)", err)
			}
			var want map[string]any
			if err := json.Unmarshal(raw, &want); err != nil {
				t.Fatalf("parse golden: %v", err)
			}
			if diff := cmp.Diff(want, golden); diff != "" {
				t.Errorf("golden mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
