package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/core"
)

type SSEHub struct {
	bus         *bus.EventBus
	mu          sync.RWMutex
	clients     map[string]map[chan []byte]bool
	projectFilter string
}

func NewSSEHub(b *bus.EventBus) *SSEHub {
	h := &SSEHub{
		bus:     b,
		clients: make(map[string]map[chan []byte]bool),
	}
	h.subscribe()
	return h
}

func (h *SSEHub) subscribe() {
	eventTypes := []core.EventType{
		core.EventMessageCreated,
		core.EventMessageStatus,
		core.EventVerificationUpdated,
		core.EventBatchUpdated,
		core.EventRequestLogged,
		core.EventWebhookDelivered,
	}
	
	for _, et := range eventTypes {
		h.bus.Subscribe(string(et), func(e core.Event) {
			h.broadcast(e)
		})
	}
}

func (h *SSEHub) broadcast(e core.Event) {
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	// Broadcast to all clients (no project filter in hub, handler filters)
	for _, clients := range h.clients {
		for ch := range clients {
			select {
			case ch <- data:
			default:
				// Client buffer full, skip
			}
		}
	}
}

func (h *SSEHub) Register(projectID string) chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.clients[projectID] == nil {
		h.clients[projectID] = make(map[chan []byte]bool)
	}
	
	ch := make(chan []byte, 256)
	h.clients[projectID][ch] = true
	return ch
}

func (h *SSEHub) Unregister(projectID string, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if clients, ok := h.clients[projectID]; ok {
		delete(clients, ch)
		close(ch)
		if len(clients) == 0 {
			delete(h.clients, projectID)
		}
	}
}

func (h *SSEHub) SSEHandler(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	
	ch := h.Register(projectID)
	defer h.Unregister(projectID, ch)
	
	// Send initial connection event
	connEvent := map[string]interface{}{
		"event": "connected",
		"data": map[string]string{
			"project_id": projectID,
			"timestamp":  time.Now().Format(time.RFC3339),
		},
	}
	if data, err := json.Marshal(connEvent); err == nil {
		_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()
	}
	
	ctx := r.Context()
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-ch:
			_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()
		case <-heartbeat.C:
			// Send heartbeat comment to keep connection alive
			_, _ = w.Write([]byte(": heartbeat\n\n"))
			flusher.Flush()
		}
	}
}