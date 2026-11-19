package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

type mockLLMClientForRunner struct {
	name string
	code string
	err  error
}

func (m *mockLLMClientForRunner) ProviderName() string {
	return m.name
}

func (m *mockLLMClientForRunner) ActiveModel() string {
	return "test-model"
}

func (m *mockLLMClientForRunner) BaseURL() string {
	return "http://localhost:11434"
}

func (m *mockLLMClientForRunner) GetAvailableModels(ctx context.Context) ([]string, error) {
	return []string{"test-model"}, nil
}

func (m *mockLLMClientForRunner) ContextLimit(ctx context.Context) int {
	return 4096
}

func (m *mockLLMClientForRunner) GenerateCode(ctx context.Context, prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.code, nil
}

func (m *mockLLMClientForRunner) Chat(ctx context.Context, messages []domain.Message) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.code, nil
}

func (m *mockLLMClientForRunner) Ping(ctx context.Context) bool {
	return true
}

func TestProviderGenerator(t *testing.T) {
	client := &mockLLMClientForRunner{
		name: "test-client",
		code: "package main",
	}
	pg := NewProviderGenerator(client)
	if pg.ProviderName() != "test-client" {
		t.Fatalf("unexpected provider name: %s", pg.ProviderName())
	}

	res, err := pg.Generate(context.Background(), "hello")
	if err != nil || res != "package main" {
		t.Fatalf("unexpected generate result: %s, err: %v", res, err)
	}
}

func TestRunnerCLI_Verify(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Success verify
	_ = os.WriteFile(filepath.Join(tmpDir, "app.go"), []byte("package app\n"), 0644)
	cmd := NewRunnerStandaloneCmd(nil, tmpDir)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"verify", "--cmd", "echo ok"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected verify to pass: %v, out: %s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "All verification checks passed") {
		t.Fatalf("unexpected output: %s", buf.String())
	}

	// 2. Failure verify with violations and failing cmd
	badFile := filepath.Join(tmpDir, "oversized.go")
	_ = os.WriteFile(badFile, []byte("1\n2\n3\n4\n"), 0644)
	cmd = NewRunnerStandaloneCmd(nil, tmpDir)
	buf.Reset()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"verify", "--max-source", "2", "--max-test", "2", "--cmd", "exit 1", "--file", "oversized.go", "--test-first"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected verify to fail on violations and command failure")
	}
	if !strings.Contains(buf.String(), "Invariant violations detected") {
		t.Fatalf("expected violations in output: %s", buf.String())
	}

	// 3. Bad policy path
	cmd = NewRunnerStandaloneCmd(nil, tmpDir)
	cmd.SetArgs([]string{"verify", "--policy", filepath.Join(tmpDir, "missing-policy.json")})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error on missing policy")
	}

	// 4. Empty project defaults to "."
	cmd = NewRunnerStandaloneCmd(nil, "")
	cmd.SetArgs([]string{"verify", "--cmd", "echo ok"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected verify with default root to pass: %v", err)
	}

	// 5. Verify error on non-existent directory
	cmd = NewRunnerStandaloneCmd(nil, "")
	cmd.SetArgs([]string{"verify", "--project", filepath.Join(tmpDir, "missing-dir")})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected verify error on missing directory")
	}
}

func TestRunnerCLI_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	gen := &mockGenerator{
		name:      "gen-test",
		responses: []string{"```go\npackage main\n```"},
	}

	// 1. Successful execute
	cmd := NewRunnerStandaloneCmd(gen, tmpDir)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"execute", "--instruction", "build", "--target", "main.go", "--cmd", "echo ok", "--heal=true", "--turns", "2"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected execute to succeed: %v, out: %s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "task completed successfully") {
		t.Fatalf("unexpected execute output: %s", buf.String())
	}

	// 2. Failed execute (verification failure without heal)
	failGen := &mockGenerator{
		name:      "gen-fail",
		responses: []string{"code"},
	}
	cmd = NewRunnerStandaloneCmd(failGen, tmpDir)
	buf.Reset()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"execute", "--instruction", "build", "--target", "fail.go", "--cmd", "exit 1", "--heal=false"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected execute to fail when verification fails")
	}

	// 3. Generator error
	errGen := &mockGenerator{
		name:   "gen-err",
		errors: []error{errors.New("gen error")},
	}
	cmd = NewRunnerStandaloneCmd(errGen, tmpDir)
	cmd.SetArgs([]string{"execute", "--instruction", "build"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected execute error on generator failure")
	}

	// 4. Empty project defaults to "."
	cmd = NewRunnerStandaloneCmd(nil, "")
	cmd.SetArgs([]string{"execute", "--cmd", "echo ok"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected execute with default root to pass: %v", err)
	}
}

func TestRunnerCLI_Inspect(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte("package main\n"), 0644)

	// 1. Inspect text format success
	cmd := NewRunnerStandaloneCmd(nil, tmpDir)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"inspect"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected inspect to pass: %v, out: %s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "Status:       PASSED") {
		t.Fatalf("unexpected inspect output: %s", buf.String())
	}

	// 2. Inspect JSON format
	cmd = NewRunnerStandaloneCmd(nil, tmpDir)
	buf.Reset()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"inspect", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected JSON inspect to pass: %v", err)
	}
	if !strings.Contains(buf.String(), `"passed": true`) {
		t.Fatalf("unexpected JSON output: %s", buf.String())
	}

	// 3. Inspect failure with violation
	_ = os.WriteFile(filepath.Join(tmpDir, "oversized.go"), []byte("1\n2\n3\n4\n"), 0644)
	polFile := filepath.Join(tmpDir, "policy.json")
	_ = os.WriteFile(polFile, []byte(`{"max_source_lines": 2}`), 0644)

	cmd = NewRunnerStandaloneCmd(nil, tmpDir)
	buf.Reset()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"inspect", "--policy", polFile})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected inspect to fail on violations")
	}
	if !strings.Contains(buf.String(), "Status:       FAILED") {
		t.Fatalf("unexpected output on failure: %s", buf.String())
	}

	// 4. Bad policy file in inspect
	cmd = NewRunnerStandaloneCmd(nil, tmpDir)
	cmd.SetArgs([]string{"inspect", "--policy", filepath.Join(tmpDir, "missing-policy.json")})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected inspect error on missing policy")
	}

	// 5. Inspect with empty project flag using os.Getwd()
	cmd = NewRunnerStandaloneCmd(nil, "")
	buf.Reset()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"inspect"})
	// Running inspect in current working directory
	_ = cmd.Execute()

	// 6. getwd error in inspect
	origGetwd := getwd
	getwd = func() (string, error) {
		return "", errors.New("mock getwd error")
	}
	cmd = NewRunnerStandaloneCmd(nil, "")
	cmd.SetArgs([]string{"inspect"})
	err := cmd.Execute()
	getwd = origGetwd
	if err == nil {
		t.Fatal("expected inspect error on getwd failure")
	}

	// 7. Inspect error on missing directory
	cmd = NewRunnerStandaloneCmd(nil, "")
	cmd.SetArgs([]string{"inspect", "--project", filepath.Join(tmpDir, "missing-dir")})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected inspect error on missing directory")
	}
}

type mockGenerator struct {
	name      string
	responses []string
	errors    []error
	calls     int
}

func (m *mockGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	idx := m.calls
	m.calls++
	if idx < len(m.errors) && m.errors[idx] != nil {
		return "", m.errors[idx]
	}
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return "default code", nil
}

func (m *mockGenerator) ProviderName() string {
	if m.name != "" {
		return m.name
	}
	return "mock-generator"
}
