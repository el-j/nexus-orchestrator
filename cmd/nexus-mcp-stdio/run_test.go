package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// failingWriter fails after n successful writes.
type failingWriter struct{ n int }

func (f *failingWriter) Write(p []byte) (int, error) {
	if f.n <= 0 {
		return 0, errors.New("pipe closed")
	}
	f.n--
	return len(p), nil
}

func TestWriteErrTo_AlwaysEmitsValidJSONRPCError(t *testing.T) {
	// Go's HTTP errors contain double quotes; the old fmt-interpolated writer
	// turned these into invalid JSON.
	msg := `Post "http://127.0.0.1:1/mcp": dial tcp 127.0.0.1:1: connect: connection refused`
	var out bytes.Buffer
	writeErrTo(&out, msg)

	var got struct {
		JSONRPC string `json:"jsonrpc"`
		ID      any    `json:"id"`
		Error   struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	if got.JSONRPC != "2.0" || got.ID != nil || got.Error.Code != -32603 || got.Error.Message != msg {
		t.Errorf("unexpected response: %+v", got)
	}
	if !strings.HasSuffix(out.String(), "\n") {
		t.Error("response must be newline-terminated")
	}
}

func TestProxy_UnreachableDaemonYieldsValidJSONErrorAndContinues(t *testing.T) {
	in := strings.NewReader("{\"id\":1}\n\n{\"id\":2}\n")
	var out, errOut bytes.Buffer
	if err := proxy(in, &out, &errOut, "http://127.0.0.1:1/mcp", ""); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 error responses (blank line skipped), got %d:\n%s", len(lines), out.String())
	}
	for _, l := range lines {
		if !json.Valid([]byte(l)) {
			t.Errorf("invalid JSON line: %s", l)
		}
	}
	if !strings.Contains(errOut.String(), "POST error") {
		t.Errorf("stderr = %q", errOut.String())
	}
}

func TestProxy_InvalidURLReportsRequestError(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := proxy(strings.NewReader("{}\n"), &out, &errOut, "http://bad host/", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "new request error") || !json.Valid(bytes.TrimSpace(out.Bytes())) {
		t.Errorf("stderr=%q stdout=%q", errOut.String(), out.String())
	}
}

func TestProxy_NoContentAndEmptyBodiesProduceNoOutput(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"204":   func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		"empty": func(w http.ResponseWriter, _ *http.Request) {},
	} {
		srv := httptest.NewServer(handler)
		var out, errOut bytes.Buffer
		if err := proxy(strings.NewReader("{}\n"), &out, &errOut, srv.URL, ""); err != nil || out.Len() != 0 {
			t.Errorf("%s: err=%v out=%q", name, err, out.String())
		}
		srv.Close()
	}
}

func TestProxy_AddsNewlineOnlyWhenMissingAndOmitsAuthWithoutToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("Authorization must not be sent without a token")
		}
		io.WriteString(w, "{\"ok\":true}\n")
	}))
	defer srv.Close()
	var out, errOut bytes.Buffer
	if err := proxy(strings.NewReader("{}\n"), &out, &errOut, srv.URL, ""); err != nil {
		t.Fatal(err)
	}
	if out.String() != "{\"ok\":true}\n" {
		t.Errorf("out = %q (no doubled newline expected)", out.String())
	}
}

func TestProxy_StdoutFailuresAreFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"ok":true}`) // no trailing newline → second write
	}))
	defer srv.Close()

	var errOut bytes.Buffer
	if err := proxy(strings.NewReader("{}\n"), &failingWriter{n: 0}, &errOut, srv.URL, ""); err == nil || !strings.Contains(err.Error(), "stdout write error") {
		t.Errorf("body write failure: %v", err)
	}
	if err := proxy(strings.NewReader("{}\n"), &failingWriter{n: 1}, &errOut, srv.URL, ""); err == nil || !strings.Contains(err.Error(), "stdout newline error") {
		t.Errorf("newline write failure: %v", err)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("stdin broke") }

func TestProxy_StdinReadErrorIsReturned(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := proxy(errReader{}, &out, &errOut, "http://127.0.0.1:1", ""); err == nil || !strings.Contains(err.Error(), "stdin broke") {
		t.Fatalf("got %v", err)
	}
}

func TestProxy_ResponseBodyReadFailure(t *testing.T) {
	// Declares a longer body than it sends, so ReadAll fails mid-stream.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		io.WriteString(w, "short")
	}))
	defer srv.Close()
	var out, errOut bytes.Buffer
	if err := proxy(strings.NewReader("{}\n"), &out, &errOut, srv.URL, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "read response") || !json.Valid(bytes.TrimSpace(out.Bytes())) {
		t.Errorf("stderr=%q stdout=%q", errOut.String(), out.String())
	}
}

func TestRun_ResolvesEnvAndReportsExitCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("token not forwarded: %q", r.Header.Get("Authorization"))
		}
		io.WriteString(w, "{\"ok\":true}\n")
	}))
	defer srv.Close()
	env := map[string]string{"NEXUS_MCP_URL": srv.URL, "NEXUS_MCP_TOKEN": "  tok  "}
	var out, errOut bytes.Buffer
	if code := run(strings.NewReader("{}\n"), &out, &errOut, func(k string) string { return env[k] }); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errOut.String())
	}

	// Default endpoint is used when NEXUS_MCP_URL is unset; a failing stdin gives exit 1.
	errOut.Reset()
	code := run(errReader{}, &out, &errOut, func(string) string { return "" })
	if code != 1 || !strings.Contains(errOut.String(), "127.0.0.1:63988/mcp") {
		t.Errorf("code=%d stderr=%q", code, errOut.String())
	}
}
