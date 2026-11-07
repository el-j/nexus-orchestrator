package httpapi

import (
	"encoding/json"
	"fmt"
	"sync"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
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

// Subscribe adds a new SSE client channel and returns it.
// The caller must call Unsubscribe when done to avoid leaking the channel.
func (h *Hub) Subscribe() chan []byte {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes and closes a client channel.
func (h *Hub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
}

// Broadcast sends event to all active subscribers. Slow clients are skipped
// (non-blocking channel send) to prevent one slow browser from stalling others.
func (h *Hub) Broadcast(event ports.TaskEvent) { h.broadcastJSON(event) }

// BroadcastAISessionEvent sends an AI session lifecycle event to all active
// subscribers. Slow clients are skipped to prevent blocking.
func (h *Hub) BroadcastAISessionEvent(event domain.AISessionEvent) { h.broadcastJSON(event) }

// BroadcastProviderDiscovered announces that a provider scan finished, so
// connected UIs can refresh their provider lists immediately. count is the
// number of providers the scan found.
func (h *Hub) BroadcastProviderDiscovered(count int) {
	h.broadcastJSON(struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	}{"provider_discovered", count})
}

// BroadcastActivityEvent sends an AI activity event to all active SSE subscribers.
func (h *Hub) BroadcastActivityEvent(a domain.AIActivity) {
	h.broadcastJSON(struct {
		Type     string            `json:"type"`
		Activity domain.AIActivity `json:"activity"`
	}{"ai_activity_new", a})
}

// broadcastJSON encodes v as one server-sent event and fans it out. A value that cannot be encoded
// (a time outside the years JSON can write) is dropped: there is nothing a subscriber could read.
func (h *Hub) broadcastJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.broadcastRaw(fmt.Sprintf("data: %s\n\n", data))
}

// broadcastRaw fans a pre-encoded SSE message out to every subscriber without blocking.
func (h *Hub) broadcastRaw(msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- []byte(msg):
		default: // slow client — drop this event rather than blocking
		}
	}
}
