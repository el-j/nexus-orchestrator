package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// TestHandleEvents_EndsWhenTheHubClosesTheClientChannel covers the defensive exit of the event loop.
func TestHandleEvents_EndsWhenTheHubClosesTheClientChannel(t *testing.T) {
	hub := NewHub()
	s := NewServer(nil, nil, hub)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		s.handleEvents(rec, httptest.NewRequest(http.MethodGet, "/api/events", nil))
		close(done)
	}()
	deadline := time.Now().Add(2 * time.Second)
	for {
		hub.mu.Lock()
		n := len(hub.clients)
		for ch := range hub.clients {
			close(ch)
			delete(hub.clients, ch) // the handler's Unsubscribe then finds nothing left to close
		}
		hub.mu.Unlock()
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the handler never subscribed")
		}
		time.Sleep(5 * time.Millisecond)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the event stream did not end after its channel was closed")
	}
}

// TestHandleEvents_SkipsALogEntryItCannotEncodeAndEndsWhenTheLogFeedCloses covers the log branch of
// the event loop.
func TestHandleEvents_SkipsALogEntryItCannotEncodeAndEndsWhenTheLogFeedCloses(t *testing.T) {
	logs := NewLogHub()
	s := NewServer(nil, nil, NewHub()).WithLogHub(logs)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		s.handleEvents(rec, httptest.NewRequest(http.MethodGet, "/api/events", nil))
		close(done)
	}()
	deadline := time.Now().Add(2 * time.Second)
	for {
		logs.mu.Lock()
		subs := append([]chan domain.LogEntry(nil), logs.subs...)
		logs.mu.Unlock()
		if len(subs) > 0 {
			subs[0] <- domain.LogEntry{Timestamp: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC), Message: "cannot be written"}
			subs[0] <- domain.LogEntry{Timestamp: time.Now(), Message: "readable"}
			logs.mu.Lock()
			logs.subs = nil // the handler's Unsubscribe then has nothing left to close
			logs.mu.Unlock()
			close(subs[0])
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the handler never subscribed to the log feed")
		}
		time.Sleep(5 * time.Millisecond)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the event stream did not end after the log feed closed")
	}
	body := rec.Body.String()
	if strings.Contains(body, "cannot be written") || !strings.Contains(body, "readable") {
		t.Errorf("stream = %q", body)
	}
}

func TestMCPEndpoint_UsesTheCallersHostWhenTheListenAddressHasNone(t *testing.T) {
	for _, tc := range []struct{ host, addr, want string }{
		{"nexus.local", ":63988", "http://nexus.local:63988/mcp"},              // a host without a port
		{"nexus.local:63987", "0.0.0.0:63988", "http://nexus.local:63988/mcp"}, // host and port
		{"nexus.local", "127.0.0.1:63988", "http://127.0.0.1:63988/mcp"},       // an explicit host wins
		{"nexus.local", "", ""},        // no MCP server
		{"nexus.local", "garbage", ""}, // not an address
	} {
		if got := mcpEndpoint("http", tc.host, tc.addr); got != tc.want {
			t.Errorf("mcpEndpoint(%q, %q) = %q, want %q", tc.host, tc.addr, got, tc.want)
		}
	}
}

func TestAccessLog_AHandlerThatWritesNothingIsAnOKRead(t *testing.T) {
	rec := httptest.NewRecorder()
	accessLog(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestDetectLevel_ReadsTheLevelFromTheMessage(t *testing.T) {
	logs := NewLogHub()
	for _, line := range []string{"WARN: slow", "[DEBUG] detail", "plain line"} {
		_, _ = logs.Write([]byte(line + "\n"))
	}
	var got []domain.LogLevel
	for _, e := range logs.Buffer() {
		got = append(got, e.Level)
	}
	want := []domain.LogLevel{domain.LogLevelWarn, domain.LogLevelDebug, domain.LogLevelInfo}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("levels = %v, want %v", got, want)
	}
}

// flushingBroken is a stream whose client has gone: writes fail, Flush works.
type flushingBroken struct{ header http.Header }

func (f *flushingBroken) Header() http.Header {
	if f.header == nil {
		f.header = http.Header{}
	}
	return f.header
}
func (f *flushingBroken) Write([]byte) (int, error) { return 0, errors.New("client went away") }
func (f *flushingBroken) WriteHeader(int)           {}
func (f *flushingBroken) Flush()                    {}

func TestHandleEvents_EndsWhenTheClientCannotBeWrittenTo(t *testing.T) {
	hub := NewHub()
	s := NewServer(nil, nil, hub)
	done := make(chan struct{})
	go func() {
		s.handleEvents(&flushingBroken{}, httptest.NewRequest(http.MethodGet, "/api/events", nil))
		close(done)
	}()
	deadline := time.After(2 * time.Second)
	for {
		hub.BroadcastProviderDiscovered(1)
		select {
		case <-done:
			return
		case <-deadline:
			t.Fatal("the stream kept running for a client that cannot be written to")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestLogHub_DropsEntriesForASubscriberThatDoesNotRead(t *testing.T) {
	logs := NewLogHub()
	slow := logs.Subscribe() // never read
	defer logs.Unsubscribe(slow)
	for i := 0; i < cap(slow)+50; i++ {
		_, _ = logs.Write([]byte("line\n"))
	}
	if len(slow) != cap(slow) {
		t.Errorf("the slow subscriber holds %d entries, want its %d-entry buffer full and the rest dropped", len(slow), cap(slow))
	}
	if len(logs.Buffer()) < cap(slow) {
		t.Error("the ring buffer must keep every entry the subscriber could not take")
	}
}
