package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/config"
	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/sim"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

type fakeRedispatcher struct {
	calledWith *core.WebhookDelivery
	returns    *core.WebhookDelivery
	err        error
}

func (f *fakeRedispatcher) Redispatch(ctx context.Context, original *core.WebhookDelivery) (*core.WebhookDelivery, error) {
	f.calledWith = original
	if f.err != nil {
		return nil, f.err
	}
	return f.returns, nil
}

func setupReplayTest(t *testing.T, redispatch WebhookRedispatcher) (*Handlers, core.Store, core.Project) {
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
	handlers := NewHandlers(svc, resolver, eventBus, redispatch, "test", &config.SecurityConfig{})

	project := core.Project{ID: "prj_replay", Name: "replay", Settings: map[string]interface{}{}, CreatedAt: time.Now()}
	if err := store.CreateProject(context.Background(), &project); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	return handlers, store, project
}

func seedReplayDelivery(t *testing.T, store core.Store, projectID string) *core.WebhookDelivery {
	t.Helper()
	delivery := &core.WebhookDelivery{
		ID:        core.NewWebhookDeliveryID(),
		ProjectID: projectID,
		Kind:      "status",
		URL:       "https://example.com/hook",
		Payload:   map[string]interface{}{"event": "message.status"},
		Headers:   map[string]string{"X-Sig": "stale"},
		Attempt:   6,
		Status:    core.WebhookFailed,
		CreatedAt: time.Now(),
	}
	if err := store.CreateWebhookDelivery(context.Background(), delivery); err != nil {
		t.Fatalf("CreateWebhookDelivery: %v", err)
	}
	return delivery
}

func callReplay(t *testing.T, handlers *Handlers, projectID, deliveryID string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/"+deliveryID+"/replay", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", deliveryID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, projectKey{}, projectID)
	rec := httptest.NewRecorder()
	handlers.ReplayWebhook(rec, req.WithContext(ctx), deliveryID, ReplayWebhookParams{})
	var decoded map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("replay response is not JSON: %v (%s)", err, rec.Body.String())
	}
	return rec.Code, decoded
}

func TestReplayWebhook_Redispatched(t *testing.T) {
	fresh := &core.WebhookDelivery{
		ID:        core.NewWebhookDeliveryID(),
		ProjectID: "prj_replay",
		Kind:      "status",
		URL:       "https://example.com/hook",
		Payload:   map[string]interface{}{"event": "message.status"},
		Headers:   map[string]string{"X-Sig": "fresh"},
		Attempt:   0,
		Status:    core.WebhookPending,
		CreatedAt: time.Now(),
	}
	fake := &fakeRedispatcher{returns: fresh}
	handlers, store, project := setupReplayTest(t, fake)
	original := seedReplayDelivery(t, store, project.ID)

	code, decoded := callReplay(t, handlers, project.ID, original.ID)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", code, decoded)
	}
	if fake.calledWith == nil || fake.calledWith.ID != original.ID {
		t.Fatalf("expected redispatcher to receive the original, got %+v", fake.calledWith)
	}
	if decoded["id"] != fresh.ID {
		t.Errorf("expected fresh record ID, got %v", decoded["id"])
	}
	if decoded["attempt"] != float64(0) {
		t.Errorf("expected attempt reset, got %v", decoded["attempt"])
	}
	headers, _ := decoded["headers"].(map[string]interface{})
	if headers["X-Sig"] != "fresh" {
		t.Errorf("expected re-signed headers, got %v", headers)
	}
}

func TestReplayWebhook_NilFallbackClones(t *testing.T) {
	handlers, store, project := setupReplayTest(t, nil)
	original := seedReplayDelivery(t, store, project.ID)

	code, decoded := callReplay(t, handlers, project.ID, original.ID)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", code, decoded)
	}
	if decoded["id"] == original.ID {
		t.Errorf("expected a new record, got the original ID")
	}
	if decoded["attempt"] != float64(0) {
		t.Errorf("expected attempt reset, got %v", decoded["attempt"])
	}
	headers, _ := decoded["headers"].(map[string]interface{})
	if headers["X-Sig"] != "stale" {
		t.Errorf("expected verbatim headers, got %v", headers)
	}
}

func TestReplayWebhook_NotFound(t *testing.T) {
	handlers, _, project := setupReplayTest(t, nil)
	code, _ := callReplay(t, handlers, project.ID, "whd_missing")
	if code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", code)
	}
}
