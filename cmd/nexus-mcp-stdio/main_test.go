package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProxy_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			t.Errorf("expected Bearer secret-token, got %s", r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "tools/list") {
			t.Errorf("unexpected body: %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`))
	}))
	defer server.Close()

	in := bytes.NewBufferString("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/list\"}\n")
	var out, errOut bytes.Buffer

	err := proxy(in, &out, &errOut, server.URL, "secret-token")
	if err != nil {
		t.Fatalf("unexpected proxy error: %v", err)
	}

	if !strings.Contains(out.String(), "tools") {
		t.Errorf("expected response on stdout, got: %s", out.String())
	}
	if !strings.HasSuffix(out.String(), "\n") {
		t.Errorf("expected newline termination on stdout, got: %q", out.String())
	}
}

func TestProxy_BackendError(t *testing.T) {
	in := bytes.NewBufferString("{\"jsonrpc\":\"2.0\",\"id\":1}\n")
	var out, errOut bytes.Buffer

	// Target invalid endpoint to trigger connection error
	err := proxy(in, &out, &errOut, "http://127.0.0.1:0/mcp", "")
	if err != nil {
		t.Fatalf("proxy should handle request errors gracefully: %v", err)
	}

	if !strings.Contains(out.String(), "-32603") {
		t.Errorf("expected JSON-RPC error on connection failure, got: %s", out.String())
	}
}
