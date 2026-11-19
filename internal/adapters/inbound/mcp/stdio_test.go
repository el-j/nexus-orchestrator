package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// mockBrainServiceForStdio implements ports.BrainService for stdio testing.
type mockBrainServiceForStdio struct {
	searchResult []domain.ContextSection
	searchErr    error
}

func (m *mockBrainServiceForStdio) GetContext(context.Context, domain.ContextQuery) (domain.ContextResponse, error) {
	return domain.ContextResponse{}, nil
}
func (m *mockBrainServiceForStdio) GetFocusedContext(context.Context, domain.ContextQuery) (domain.ContextResponse, error) {
	return domain.ContextResponse{}, nil
}
func (m *mockBrainServiceForStdio) IngestKnowledge(context.Context, domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	return domain.ProjectKnowledge{}, nil
}
func (m *mockBrainServiceForStdio) IngestFromFile(context.Context, string, string) (int, error) {
	return 1, nil
}
func (m *mockBrainServiceForStdio) SearchKnowledge(context.Context, string, string, int) ([]domain.ContextSection, error) {
	return m.searchResult, m.searchErr
}
func (m *mockBrainServiceForStdio) GetFileMap(context.Context, string, string) ([]string, error) {
	return []string{"a.go"}, nil
}
func (m *mockBrainServiceForStdio) InitProject(context.Context, string, string) (domain.BrainStatus, error) {
	return domain.BrainStatus{Initialized: true}, nil
}
func (m *mockBrainServiceForStdio) GetStatus(context.Context, string) (domain.BrainStatus, error) {
	return domain.BrainStatus{Initialized: true, EntryCount: 10}, nil
}
func (m *mockBrainServiceForStdio) ListKnowledge(context.Context, string, string) ([]domain.ProjectKnowledge, error) {
	return nil, nil
}
func (m *mockBrainServiceForStdio) DeleteKnowledge(context.Context, string) error {
	return nil
}
func (m *mockBrainServiceForStdio) GetOnboardingContext(context.Context, string, int) (string, error) {
	return "test onboarding", nil
}

func TestServeStdio_ProtocolFlow(t *testing.T) {
	brain := &mockBrainServiceForStdio{
		searchResult: []domain.ContextSection{{Topic: "test topic", Content: "test content"}},
	}
	server := NewMcpServer(nil, brain)

	// Send initialize, empty line, tools/list, and ping
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
		``, // empty line should be skipped
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":4,"method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"search_knowledge","arguments":{"projectPath":"/p","query":"test"}}}`,
	}, "\n") + "\n"

	var in bytes.Buffer
	in.WriteString(input)
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := server.ServeStdio(ctx, &in, &out); err != nil {
		t.Fatalf("ServeStdio: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 4 { // initialize, tools/list, ping, tools/call (notifications has no response body)
		t.Fatalf("expected 4 response lines, got %d:\n%s", len(lines), out.String())
	}

	// Verify tools/list returned only brain tools (since orch == nil)
	var toolsResp rpcResponse
	if err := json.Unmarshal([]byte(lines[1]), &toolsResp); err != nil {
		t.Fatalf("unmarshal tools/list response: %v", err)
	}
	resMap, ok := toolsResp.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", toolsResp.Result)
	}
	toolSlice, ok := resMap["tools"].([]any)
	if !ok {
		t.Fatalf("expected tools slice, got %T", resMap["tools"])
	}
	// Standalone brain tools count should be 10 (the brainToolDefs)
	if len(toolSlice) != 10 {
		t.Errorf("expected 10 brain tools in standalone mode, got %d", len(toolSlice))
	}
}

func TestServeStdio_Errors(t *testing.T) {
	server := NewMcpServer(nil, nil)

	input := strings.Join([]string{
		`{invalid json`,
		`{"jsonrpc":"1.0","id":1,"method":"ping"}`,
	}, "\n") + "\n"

	var in bytes.Buffer
	in.WriteString(input)
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := server.ServeStdio(ctx, &in, &out); err != nil {
		t.Fatalf("ServeStdio: %v", err)
	}

	outStr := out.String()
	if !strings.Contains(outStr, "parse error") {
		t.Errorf("expected parse error, got %q", outStr)
	}
	if !strings.Contains(outStr, `invalid request`) {
		t.Errorf("expected invalid request error, got %q", outStr)
	}
}

func TestServeStdio_ContextCancellation(t *testing.T) {
	server := NewMcpServer(nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	var in bytes.Buffer
	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")
	var out bytes.Buffer

	err := server.ServeStdio(ctx, &in, &out)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestStdioResponseWriter_Methods(t *testing.T) {
	var buf bytes.Buffer
	w := &stdioResponseWriter{out: &buf}
	h := w.Header()
	if h == nil {
		t.Fatal("expected non-nil header")
	}
	w.WriteHeader(200) // should be no-op
	n, err := w.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Fatalf("write failed: n=%d err=%v", n, err)
	}
	if buf.String() != "hello" {
		t.Errorf("buf = %q, want 'hello'", buf.String())
	}
}

type failWriter struct {
	failOnWrite bool
}

func (f *failWriter) Write(p []byte) (n int, err error) {
	if f.failOnWrite {
		return 0, errors.New("write error")
	}
	return len(p), nil
}

func TestServeStdio_WriteError(t *testing.T) {
	server := NewMcpServer(nil, nil)
	var in bytes.Buffer
	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")
	fw := &failWriter{failOnWrite: true}

	err := server.ServeStdio(context.Background(), &in, fw)
	if err == nil || !strings.Contains(err.Error(), "write error") {
		t.Fatalf("expected write error, got: %v", err)
	}
}
