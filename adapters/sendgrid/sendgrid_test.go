package sendgrid

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

func setupTestSendGrid(t *testing.T) (http.Handler, core.Store) {
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
	handler := kit.Wrap(New(svc), Extractor(), adapterkit.WithRequired(true))
	return handler, store
}

func TestSendGrid_SendMail(t *testing.T) {
	handler, _ := setupTestSendGrid(t)
	body := map[string]any{
		"personalizations": []map[string]any{
			{"to": []map[string]string{{"email": "alice@example.com"}}},
		},
		"from":    map[string]string{"email": "sender@example.com"},
		"subject": "Hello",
		"content": []map[string]string{{"type": "text/plain", "value": "hi"}},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v3/mail/send", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sg_test_key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("X-Message-Id") == "" {
		t.Fatalf("missing X-Message-Id header")
	}
}
