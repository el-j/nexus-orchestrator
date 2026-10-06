package mcp_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/mcp"
)

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

func TestStartMCPServer_ServesGuardsAndShutsDown(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "")
	t.Setenv("NEXUS_ALLOWED_HOSTS", "")
	addr := freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- mcp.StartMCPServer(ctx, newFailOrch(), newFailBrain(), addr) }()

	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get("http://" + addr + "/health")
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("MCP server never came up")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// DNS rebinding: a hostile hostname resolving to 127.0.0.1 must be refused.
	req, _ := http.NewRequest(http.MethodGet, "http://"+addr+"/health", nil)
	req.Host = "attacker.example"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("rebinding Host: status %d, want 403", resp.StatusCode)
	}
	// A cross-site browser POST is refused too.
	req, _ = http.NewRequest(http.MethodPost, "http://"+addr+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	req.Header.Set("Origin", "https://evil.example")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("foreign Origin: status %d, want 403", resp.StatusCode)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("clean shutdown returned %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("MCP server did not stop")
	}
}

func TestStartMCPServer_ReportsListenError(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	err = mcp.StartMCPServer(context.Background(), newFailOrch(), newFailBrain(), l.Addr().String())
	if err == nil || !strings.Contains(err.Error(), "mcp: listen") {
		t.Errorf("got %v", err)
	}
}

func TestOriginAllowListHonoursEnvironmentExtras(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "")
	t.Setenv("NEXUS_ALLOWED_ORIGINS", "https://tool.example.com")
	srv := httptest.NewServer(mcp.NewMcpServer(newFailOrch(), newFailBrain()))
	defer srv.Close()
	do := func(origin string) int {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if c := do("https://tool.example.com"); c != 200 {
		t.Errorf("configured origin: %d", c)
	}
	if c := do("https://other.example.com"); c != 403 {
		t.Errorf("unlisted origin: %d", c)
	}
	if c := do("http://localhost:5173"); c != 200 {
		t.Errorf("local origin: %d", c)
	}
}

func TestRPCEnvelopeValidation(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "")
	srv := httptest.NewServer(mcp.NewMcpServer(newFailOrch(), newFailBrain()))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/mcp", nil)
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("PUT /mcp: %d", resp.StatusCode)
	}

	if r := postRPC(t, srv, map[string]any{"jsonrpc": "1.0", "id": 1, "method": "ping"}); r.Error == nil || r.Error.Code != -32600 {
		t.Errorf("wrong jsonrpc version: %+v", r.Error)
	}
	for _, method := range []string{"ping", "resources/list", "prompts/list", "tools/list"} {
		if r := postRPC(t, srv, map[string]any{"jsonrpc": "2.0", "id": 1, "method": method}); r.Error != nil {
			t.Errorf("%s: %+v", method, r.Error)
		}
	}
	// notifications/initialized has no response body.
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	resp, err := http.Post(srv.URL+"/mcp", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("notification: %d", resp.StatusCode)
	}
}

// sseClient opens /sse and returns the message endpoint plus a reader for later events.
func sseClient(t *testing.T, srvURL string) (endpoint string, r *bufio.Reader, closeFn func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srvURL+"/sse", nil)
	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose // closed by the returned closeFn
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	br := bufio.NewReader(resp.Body)
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			cancel()
			t.Fatalf("reading endpoint event: %v", err)
		}
		if strings.HasPrefix(line, "data: ") {
			endpoint = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			break
		}
	}
	return endpoint, br, func() { cancel(); resp.Body.Close() }
}

func nextData(t *testing.T, r *bufio.Reader) string {
	t.Helper()
	done := make(chan string, 1)
	go func() {
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				done <- ""
				return
			}
			if strings.HasPrefix(line, "data: ") {
				done <- strings.TrimSpace(strings.TrimPrefix(line, "data: "))
				return
			}
		}
	}()
	select {
	case d := <-done:
		return d
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for an SSE message")
		return ""
	}
}

func TestSSEMessageErrorsAreDeliveredOverTheStream(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "")
	srv := httptest.NewServer(mcp.NewMcpServer(newFailOrch(), newFailBrain()))
	defer srv.Close()
	endpoint, stream, closeFn := sseClient(t, srv.URL)
	defer closeFn()

	post := func(body string) int {
		resp, err := http.Post(srv.URL+endpoint, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return resp.StatusCode
	}
	if c := post(`{not json`); c != http.StatusAccepted {
		t.Errorf("unparsable body: %d", c)
	}
	if d := nextData(t, stream); !strings.Contains(d, "-32700") {
		t.Errorf("parse error event: %q", d)
	}
	if c := post(`{"jsonrpc":"1.0","id":3,"method":"ping"}`); c != http.StatusAccepted {
		t.Errorf("wrong version: %d", c)
	}
	if d := nextData(t, stream); !strings.Contains(d, "-32600") {
		t.Errorf("invalid request event: %q", d)
	}
	if c := post(`{"jsonrpc":"2.0","id":4,"method":"ping"}`); c != http.StatusAccepted {
		t.Errorf("valid request: %d", c)
	}
	if d := nextData(t, stream); !strings.Contains(d, `"id":4`) {
		t.Errorf("response event: %q", d)
	}
}
