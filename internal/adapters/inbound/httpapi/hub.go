package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"nexus-ai/internal/core/ports"
)

// Hub manages SSE subscriber connections and broadcasts TaskEvents to all of them.
// It implements ports.EventBroadcaster and is safe for concurrent use.
type Hub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

// NewHub creates an empty Hub with no subscribers.
func NewHub() *Hub {
	return &Hub{clients: make(map[chan []byte]struct{})}
}

// Broadcast sends event to all active subscribers. Slow clients are skipped
// (non-blocking channel send) to prevent one slow browser from stalling others.
func (h *Hub) Broadcast(event ports.TaskEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	msg := []byte(fmt.Sprintf("data: %s\n\n", data))

	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- msg:
		default: // slow client — drop this event rather than blocking
		}
	}
}

// ServeSSE handles a single SSE connection for the lifetime of the request.
func (h *Hub) ServeSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, ch)
		close(ch)
		h.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering if behind proxy
	// Initial "connected" ping
	fmt.Fprint(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := w.Write(msg); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
