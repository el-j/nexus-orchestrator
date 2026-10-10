package httpapi_test

import (
	"bufio"
	"context"
	"crypto/tls"
	"html"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// readUntil reads SSE lines until one contains want, or the deadline passes.
var tlsStateStub = tls.ConnectionState{}

func readUntil(t *testing.T, r *bufio.Reader, want string) string {
	t.Helper()
	done := make(chan string, 1)
	go func() {
		var sb strings.Builder
		for {
			line, err := r.ReadString('\n')
			sb.WriteString(line)
			if strings.Contains(sb.String(), want) || err != nil {
				done <- sb.String()
				return
			}
		}
	}()
	select {
	case got := <-done:
		return got
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %q", want)
		return ""
	}
}

func TestSSEStreamsTaskSessionActivityAndLogEvents(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	hub := httpapi.NewHub()
	logHub := httpapi.NewLogHubWithWriter(io.Discard)
	srv := httptest.NewServer(httpapi.NewServer(newFailOrch(), nil, hub).WithLogHub(logHub).Handler())
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}
	br := bufio.NewReader(resp.Body)
	readUntil(t, br, `"type":"connected"`)

	hub.Broadcast(ports.TaskEvent{Type: ports.EventTaskQueued, TaskID: "t1", Status: domain.StatusQueued})
	if got := readUntil(t, br, "t1"); !strings.Contains(got, "task.queued") {
		t.Errorf("task event: %q", got)
	}
	hub.BroadcastAISessionEvent(domain.AISessionEvent{Type: "ai_session_changed", AISessionID: "s9"})
	if got := readUntil(t, br, "s9"); !strings.Contains(got, "ai_session_changed") {
		t.Errorf("session event: %q", got)
	}
	hub.BroadcastActivityEvent(domain.AIActivity{ID: "act-7", AgentName: "claude"})
	if got := readUntil(t, br, "act-7"); !strings.Contains(got, "ai_activity_new") {
		t.Errorf("activity event: %q", got)
	}
	_, _ = logHub.Write([]byte("ERROR something broke\n"))
	if got := readUntil(t, br, "something broke"); !strings.Contains(got, "event: log") {
		t.Errorf("log event: %q", got)
	}

	// Disconnecting the client must release the subscription (no goroutine/channel leak).
	cancel()
	time.Sleep(50 * time.Millisecond)
	hub.Broadcast(ports.TaskEvent{Type: ports.EventTaskQueued, TaskID: "after-close"}) // must not block or panic
}

func TestHubDropsEventsForSlowClientsInsteadOfBlocking(t *testing.T) {
	hub := httpapi.NewHub()
	ch := hub.Subscribe()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ { // far more than the 16-slot buffer, with nobody reading
			hub.Broadcast(ports.TaskEvent{Type: ports.EventTaskQueued, TaskID: "x"})
			hub.BroadcastAISessionEvent(domain.AISessionEvent{AISessionID: "s"})
			hub.BroadcastActivityEvent(domain.AIActivity{ID: "a"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a slow subscriber stalled the broadcaster")
	}
	hub.Unsubscribe(ch)
	hub.Unsubscribe(ch) // idempotent
	if _, ok := <-ch; ok {
		// drain buffered messages, then the closed channel reports !ok
		for range ch {
		}
	}
}

func TestEventsEndpointWithoutHubIsUnavailable(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	rec := send(httpapi.NewServer(newFailOrch(), nil, nil).Handler(), "GET", "/api/events", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status %d", rec.Code)
	}
}

func TestLogsEndpoint(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	if rec := send(httpapi.NewServer(newFailOrch(), nil, nil).Handler(), "GET", "/api/logs", ""); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("no log hub: %q", rec.Body)
	}
	lh := httpapi.NewLogHubWithWriter(io.Discard)
	_, _ = lh.Write([]byte("hello world\n"))
	rec := send(httpapi.NewServer(newFailOrch(), nil, nil).WithLogHub(lh).Handler(), "GET", "/api/logs", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "hello world") {
		t.Errorf("buffered logs: %d %q", rec.Code, rec.Body)
	}
	if lh2 := httpapi.NewLogHub(); lh2 == nil {
		t.Error("NewLogHub must return a hub")
	}
}

func TestDashboardServesAFreshCSPNoncePerRequest(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	h := realHandler(newFailOrch(), nil)
	nonces := map[string]bool{}
	for i := 0; i < 3; i++ {
		rec := send(h, "GET", "/ui", "")
		csp := rec.Header().Get("Content-Security-Policy")
		i := strings.Index(csp, "'nonce-")
		if rec.Code != 200 || i < 0 {
			t.Fatalf("status %d csp %q", rec.Code, csp)
		}
		nonce := csp[i+len("'nonce-") : i+len("'nonce-")+strings.Index(csp[i+len("'nonce-"):], "'")]
		// html/template HTML-escapes '+' and '=' in the attribute; browsers decode them back.
		if !strings.Contains(html.UnescapeString(rec.Body.String()), `nonce="`+nonce+`"`) {
			t.Error("the page must carry the nonce its CSP allows")
		}
		if nonces[nonce] {
			t.Error("nonce reused across requests")
		}
		nonces[nonce] = true
		if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
			t.Errorf("content type %q", rec.Header().Get("Content-Type"))
		}
	}
}

func TestDiscoveryDocumentsDeriveTheirBaseURLFromTheRequest(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	h := realHandler(newFailOrch(), nil)
	rec := send(h, "GET", "/.well-known/nexus.json", "", "Host", "nexus.example:9000")
	if !strings.Contains(rec.Body.String(), "http://nexus.example:9000/api/howto") {
		t.Errorf("well-known: %s", rec.Body)
	}
	req := httptest.NewRequest("GET", "/api/howto", nil)
	req.Host = "secure.example"
	req.TLS = &tlsStateStub
	r2 := httptest.NewRecorder()
	h.ServeHTTP(r2, req)
	if !strings.Contains(r2.Body.String(), "https://secure.example") {
		t.Errorf("howto must use https when TLS is in use: %s", r2.Body)
	}
	req = httptest.NewRequest("GET", "/.well-known/nexus.json", nil)
	req.TLS = &tlsStateStub
	r3 := httptest.NewRecorder()
	h.ServeHTTP(r3, req)
	if !strings.Contains(r3.Body.String(), "https://") {
		t.Errorf("well-known must use https under TLS: %s", r3.Body)
	}
}

func TestActivityEndpoints(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	// Without an activity service both routes report 503 as JSON.
	h := realHandler(newFailOrch(), nil)
	for _, p := range []string{"/api/activities", "/api/activities/timeline"} {
		rec := send(h, "GET", p, "")
		if rec.Code != 503 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
			t.Errorf("%s: %d %q", p, rec.Code, rec.Header().Get("Content-Type"))
		}
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

func waitHTTP(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(url); err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s never came up", url)
}

func TestStartServer_ServesUntilContextIsCancelled(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	for name, start := range map[string]func(ctx context.Context, addr string) error{
		"StartServer": func(ctx context.Context, addr string) error {
			return httpapi.StartServer(ctx, newFailOrch(), nil, addr, httpapi.NewLogHubWithWriter(io.Discard))
		},
		"StartServerFull": func(ctx context.Context, addr string) error {
			return httpapi.StartServerFull(ctx, newFailOrch(), nil, addr, nil, httpapi.NewLogHubWithWriter(io.Discard))
		},
	} {
		t.Run(name, func(t *testing.T) {
			addr := freeAddr(t)
			ctx, cancel := context.WithCancel(context.Background())
			errCh := make(chan error, 1)
			go func() { errCh <- start(ctx, addr) }()
			waitHTTP(t, "http://"+addr+"/api/health")

			// A browser page on another site cannot drive the live server (guard is wired).
			req, _ := http.NewRequest(http.MethodPost, "http://"+addr+"/api/tasks", strings.NewReader(`{"instruction":"x"}`))
			req.Header.Set("Origin", "https://evil.example")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("live server accepted a cross-site POST: %d", resp.StatusCode)
			}
			// ... and a rebinding Host header is refused on a loopback bind.
			req, _ = http.NewRequest(http.MethodGet, "http://"+addr+"/api/health", nil)
			req.Host = "attacker.example"
			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("live server accepted a rebinding Host: %d", resp.StatusCode)
			}

			cancel()
			select {
			case err := <-errCh:
				if err != nil {
					t.Errorf("clean shutdown must return nil, got %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("server did not stop after cancel")
			}
		})
	}
}

func TestStartServer_ReportsListenErrors(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := httpapi.StartServer(ctx, newFailOrch(), nil, l.Addr().String()); err == nil {
		t.Error("an address already in use must be reported")
	}
	if err := httpapi.StartServerFull(ctx, newFailOrch(), nil, l.Addr().String(), nil); err == nil {
		t.Error("an address already in use must be reported (full)")
	}
}

func TestStartServer_ShutsDownPromptlyWithAConnectedEventsClient(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	addr := freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- httpapi.StartServer(ctx, newFailOrch(), nil, addr) }()
	waitHTTP(t, "http://"+addr+"/api/health")

	resp, err := http.Get("http://" + addr + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	readUntil(t, bufio.NewReader(resp.Body), `"type":"connected"`)

	start := time.Now()
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("returned %v", err)
		}
		if took := time.Since(start); took > 3*time.Second {
			t.Errorf("shutdown took %v with an /api/events stream open", took)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("shutdown hung on the events client")
	}
}
