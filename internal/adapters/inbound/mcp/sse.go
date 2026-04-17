package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// sseKeepaliveInterval controls how often a SSE comment ping is sent to keep
// the connection alive through proxies and VS Code's MCP client. Override in
// tests by setting this to a shorter duration before the call.
var sseKeepaliveInterval = 15 * time.Second

// ----- SSE session & manager -----

type sseSession struct {
	id      string
	mu      sync.Mutex
	writer  http.ResponseWriter
	flusher http.Flusher
	done    chan struct{}
}

func (s *sseSession) sendEvent(event, data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := fmt.Fprintf(s.writer, "event: %s\ndata: %s\n\n", event, data)
	if err != nil {
		return fmt.Errorf("mcp: sse send: %w", err)
	}
	s.flusher.Flush()
	return nil
}

type sseManager struct {
	mu       sync.RWMutex
	sessions map[string]*sseSession
}

func (m *sseManager) add(id string, w http.ResponseWriter, f http.Flusher) *sseSession {
	s := &sseSession{
		id:      id,
		writer:  w,
		flusher: f,
		done:    make(chan struct{}),
	}
	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()
	return s
}

func (m *sseManager) get(id string) (*sseSession, bool) {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	return s, ok
}

func (m *sseManager) remove(id string) {
	m.mu.Lock()
	if s, ok := m.sessions[id]; ok {
		close(s.done)
		delete(m.sessions, id)
	}
	m.mu.Unlock()
}

// ----- captureWriter -----

// captureWriter implements http.ResponseWriter by buffering the response body
// so that processRPC output can be forwarded over an SSE stream.
type captureWriter struct {
	header http.Header
	body   []byte
	status int
}

func (c *captureWriter) Header() http.Header {
	if c.header == nil {
		c.header = make(http.Header)
	}
	return c.header
}

func (c *captureWriter) Write(b []byte) (int, error) {
	c.body = append(c.body, b...)
	return len(b), nil
}

func (c *captureWriter) WriteHeader(code int) {
	c.status = code
}

// ----- SSE HTTP handlers -----

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.handleSSEMessage(w, r)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	sessionID := fmt.Sprintf("mcp-sse-%d", time.Now().UnixNano())

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	session := s.sse.add(sessionID, w, flusher)
	defer s.sse.remove(sessionID)

	log.Printf("mcp: sse session %s connected", sessionID)

	// Send the endpoint event per 2024-11-05 spec.
	if err := session.sendEvent("endpoint", fmt.Sprintf("/messages?sessionId=%s", sessionID)); err != nil {
		log.Printf("mcp: sse session %s endpoint send failed: %v", sessionID, err)
		return
	}

	// Block until client disconnects or session is closed, sending periodic
	// SSE comment-pings to keep the connection alive through proxies.
	pingTicker := time.NewTicker(sseKeepaliveInterval)
	defer pingTicker.Stop()
loop:
	for {
		select {
		case <-pingTicker.C:
			session.mu.Lock()
			_, err := fmt.Fprintf(session.writer, ": ping\n\n")
			if err == nil {
				session.flusher.Flush()
			}
			session.mu.Unlock()
			if err != nil {
				break loop
			}
		case <-r.Context().Done():
			break loop
		case <-session.done:
			break loop
		}
	}
	log.Printf("mcp: sse session %s disconnected", sessionID)
}

func (s *Server) handleSSEMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "missing sessionId", http.StatusBadRequest)
		return
	}

	session, ok := s.sse.get(sessionID)
	if !ok {
		// Session not found (likely daemon restart). Return 204 No Content so
		// clients can treat this as a reconnect signal without receiving an
		// empty plain-text 400 body that confuses MCP clients.
		w.WriteHeader(http.StatusNoContent)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errResp := rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: codeParseError, Message: "parse error"}}
		data, _ := json.Marshal(errResp)
		if err := session.sendEvent("message", string(data)); err != nil {
			log.Printf("mcp: sse send: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
		return
	}

	if req.JSONRPC != "2.0" {
		errResp := rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: codeInvalidRequest, Message: `invalid request: jsonrpc must be "2.0"`}}
		data, _ := json.Marshal(errResp)
		if err := session.sendEvent("message", string(data)); err != nil {
			log.Printf("mcp: sse send: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Process the request, capturing the response body.
	capture := &captureWriter{}
	s.processRPC(capture, r, req)

	// Send the captured response via SSE.
	if err := session.sendEvent("message", string(capture.body)); err != nil {
		log.Printf("mcp: sse send: %v", err)
	}

	// Return 202 Accepted per spec.
	w.WriteHeader(http.StatusAccepted)
}
