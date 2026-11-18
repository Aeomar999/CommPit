package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/sim"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

func setupTestResend(t *testing.T) (http.Handler, core.Store) {
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
	return kit.Wrap(New(svc), Extractor(), adapterkit.WithRequired(true)), store
}

func TestResend_SendEmail(t *testing.T) {
	handler, _ := setupTestResend(t)
	body := map[string]any{
		"from":    "sender@example.com",
		"to":      []string{"alice@example.com"},
		"subject": "Hello",
		"text":    "hi",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/emails", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer re_test_key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["id"] == nil || resp["id"] == "" {
		t.Fatalf("missing id in response: %v", resp)
	}
}
