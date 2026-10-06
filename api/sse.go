package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/Aeomar999/CommPit/core"
)

type sseClient struct {
	projectID string
	ch        chan sseMessage
}

type sseMessage struct {
	eventType string
	data      []byte
}

type SSEHub struct {
	bus     core.Bus
	mu      sync.RWMutex
	clients map[string][]*sseClient
}

func NewSSEHub(b core.Bus) *SSEHub {
	h := &SSEHub{
		bus:     b,
		clients: make(map[string][]*sseClient),
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
	data, err := json.Marshal(e.Payload)
	if err != nil {
		return
	}

	msg := sseMessage{
		eventType: string(e.Type),
		data:      data,
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	// Broadcast to clients matching the event's project, plus "all" project
	projectIDs := []string{e.ProjectID, "all"}
	for _, pid := range projectIDs {
		if clients, ok := h.clients[pid]; ok {
			for _, client := range clients {
				select {
				case client.ch <- msg:
				default:
					// Client buffer full, skip
				}
			}
		}
	}
}

func (h *SSEHub) Register(projectID string) *sseClient {
	h.mu.Lock()
	defer h.mu.Unlock()

	if projectID == "" {
		projectID = "all"
	}

	ch := make(chan sseMessage, 256)
	client := &sseClient{projectID: projectID, ch: ch}

	h.clients[projectID] = append(h.clients[projectID], client)
	return client
}

func (h *SSEHub) Unregister(client *sseClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.projectID]; ok {
		for i, c := range clients {
			if c == client {
				h.clients[client.projectID] = append(clients[:i], clients[i+1:]...)
				close(c.ch)
				if len(h.clients[client.projectID]) == 0 {
					delete(h.clients, client.projectID)
				}
				break
			}
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

	// Clear write deadline so long-lived SSE connections don't time out
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	client := h.Register(projectID)
	defer h.Unregister(client)

	// Send initial connection event with event: line
	connData, _ := json.Marshal(map[string]interface{}{
		"project_id": projectID,
		"timestamp":  time.Now().Format(time.RFC3339),
	})
	_, _ = w.Write([]byte("event: connected\n"))
	_, _ = w.Write([]byte("data: " + string(connData) + "\n\n"))
	flusher.Flush()

	ctx := r.Context()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-client.ch:
			// Write with event: line for EventSource filtering
			_, _ = w.Write([]byte("event: " + msg.eventType + "\n"))
			_, _ = w.Write([]byte("data: " + string(msg.data) + "\n\n"))
			flusher.Flush()
		case <-heartbeat.C:
			// Send heartbeat comment to keep connection alive
			_, _ = w.Write([]byte(": heartbeat\n\n"))
			flusher.Flush()
		}
	}
}
