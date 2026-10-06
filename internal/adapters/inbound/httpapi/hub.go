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

// BroadcastAISessionEvent sends an AI session lifecycle event to all active
// subscribers. Slow clients are skipped to prevent blocking.
func (h *Hub) BroadcastAISessionEvent(event domain.AISessionEvent) {
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

// BroadcastActivityEvent sends an AI activity event to all active SSE subscribers.
func (h *Hub) BroadcastActivityEvent(a domain.AIActivity) {
	type activityEvent struct {
		Type     string            `json:"type"`
		Activity domain.AIActivity `json:"activity"`
	}
	data, err := json.Marshal(activityEvent{Type: "ai_activity_new", Activity: a})
	if err != nil {
		return
	}
	msg := []byte(fmt.Sprintf("data: %s\n\n", data))

	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}
