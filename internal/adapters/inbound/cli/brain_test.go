package cli_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/core/domain"
)

// mockBrainService implements ports.BrainService with configurable responses.
type mockBrainService struct {
	getStatusResult     domain.BrainStatus
	getStatusErr        error
	ingestFromFileCount int
	ingestFromFileErr   error
	searchResult        []domain.ContextSection
	searchErr           error
	initProjectResult   domain.BrainStatus
	initProjectErr      error
	deleteErr           error
	listResult          []domain.ProjectKnowledge
	listErr             error
	getContextResult    domain.ContextResponse
	getContextErr       error
	getFileMapResult    []string
	getFileMapErr       error
	ingestKnowledgeErr  error
	ingested            domain.ProjectKnowledge // the entry the last IngestKnowledge call received
}

func (m *mockBrainService) GetStatus(_ context.Context, _ string) (domain.BrainStatus, error) {
	return m.getStatusResult, m.getStatusErr
}

func (m *mockBrainService) IngestFromFile(_ context.Context, _, _ string) (int, error) {
	return m.ingestFromFileCount, m.ingestFromFileErr
}

func (m *mockBrainService) SearchKnowledge(_ context.Context, _, _ string, _ int) ([]domain.ContextSection, error) {
	return m.searchResult, m.searchErr
}

func (m *mockBrainService) InitProject(_ context.Context, _, _ string) (domain.BrainStatus, error) {
	return m.initProjectResult, m.initProjectErr
}

func (m *mockBrainService) DeleteKnowledge(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockBrainService) ListKnowledge(_ context.Context, _, _ string) ([]domain.ProjectKnowledge, error) {
	return m.listResult, m.listErr
}

func (m *mockBrainService) GetContext(_ context.Context, _ domain.ContextQuery) (domain.ContextResponse, error) {
	return m.getContextResult, m.getContextErr
}

func (m *mockBrainService) GetFocusedContext(_ context.Context, _ domain.ContextQuery) (domain.ContextResponse, error) {
	return m.getContextResult, m.getContextErr
}

func (m *mockBrainService) IngestKnowledge(_ context.Context, k domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	m.ingested = k
	return k, m.ingestKnowledgeErr
}

func (m *mockBrainService) GetFileMap(_ context.Context, _, _ string) ([]string, error) {
	return m.getFileMapResult, m.getFileMapErr
}

func (m *mockBrainService) GetOnboardingContext(_ context.Context, _ string, _ int) (string, error) {
	return "", nil
}

// TestBrainCLI_Status_ServiceError verifies that a brain service error surfaces in the command error.
func TestBrainCLI_Status_ServiceError(t *testing.T) {
	brain := &mockBrainService{getStatusErr: errors.New("database unavailable")}
	root := cli.NewRootCmd(&mockOrchestrator{}, brain)
	root.SetArgs([]string{"brain", "status", "--project", "/proj"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error from status command, got nil")
	}
	if !strings.Contains(err.Error(), "database unavailable") {
		t.Errorf("expected error to contain %q; got: %q", "database unavailable", err.Error())
	}
}

// TestBrainCLI_Ingest_OK verifies that a successful ingest prints the ingested count to stdout.
func TestBrainCLI_Ingest_OK(t *testing.T) {
	brain := &mockBrainService{ingestFromFileCount: 3}
	root := cli.NewRootCmd(&mockOrchestrator{}, brain)
	root.SetArgs([]string{"brain", "ingest", "--project", "/proj", "--file", "/tmp/CLAUDE.md"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "3") {
		t.Errorf("expected output to contain count %q; got: %q", "3", out)
	}
}

// TestBrainCLI_Search_OK verifies that search results are printed to stdout.
func TestBrainCLI_Search_OK(t *testing.T) {
	brain := &mockBrainService{
		searchResult: []domain.ContextSection{
			{Kind: domain.KnowledgeArchitecture, Topic: "Arch Topic", Content: "arch content", Tokens: 10},
			{Kind: domain.KnowledgeConvention, Topic: "Conv Topic", Content: "conv content", Tokens: 5},
		},
	}
	root := cli.NewRootCmd(&mockOrchestrator{}, brain)
	root.SetArgs([]string{"brain", "search", "--project", "/proj", "--query", "architecture"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "Arch Topic") {
		t.Errorf("expected output to contain %q; got: %q", "Arch Topic", out)
	}
	if !strings.Contains(out, "Conv Topic") {
		t.Errorf("expected output to contain %q; got: %q", "Conv Topic", out)
	}
}

// TestBrainCLI_Search_AllProjects_OK verifies that --all runs search across all projects without --project.
func TestBrainCLI_Search_AllProjects_OK(t *testing.T) {
	brain := &mockBrainService{
		searchResult: []domain.ContextSection{
			{ProjectPath: "/proj/a", Kind: domain.KnowledgeArchitecture, Topic: "Arch Topic", Content: "arch content", Tokens: 10},
		},
	}
	root := cli.NewRootCmd(&mockOrchestrator{}, brain)
	root.SetArgs([]string{"brain", "search", "--all", "--query", "architecture"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "Arch Topic") {
		t.Errorf("expected output to contain %q; got: %q", "Arch Topic", out)
	}
}

// TestBrainCLI_Init_OK verifies that a successful init prints the project path or "initialized" in output.
func TestBrainCLI_Init_OK(t *testing.T) {
	brain := &mockBrainService{
		initProjectResult: domain.BrainStatus{
			ProjectPath: "/proj",
			EntryCount:  5,
		},
	}
	root := cli.NewRootCmd(&mockOrchestrator{}, brain)
	root.SetArgs([]string{"brain", "init", "--project", "/proj"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "/proj") {
		t.Errorf("expected output to contain projectPath %q; got: %q", "/proj", out)
	}
}

// TestBrainCLI_Delete_OK verifies that a successful delete prints "deleted" to stdout.
func TestBrainCLI_Delete_OK(t *testing.T) {
	brain := &mockBrainService{deleteErr: nil}
	root := cli.NewRootCmd(&mockOrchestrator{}, brain)
	root.SetArgs([]string{"brain", "delete", "--id", "entry-uuid-123"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "deleted") {
		t.Errorf("expected output to contain %q; got: %q", "deleted", out)
	}
}

// TestBrainCLI_Context_OK verifies that the context command prints JSON output to stdout.
func TestBrainCLI_Context_OK(t *testing.T) {
	brain := &mockBrainService{
		getContextResult: domain.ContextResponse{
			ProjectPath: "/proj",
			TokenBudget: 500,
			Sections: []domain.ContextSection{
				{Topic: "Architecture", Kind: domain.KnowledgeArchitecture, Content: "Hexagonal"},
			},
		},
	}
	root := cli.NewRootCmd(&mockOrchestrator{}, brain)
	root.SetArgs([]string{"brain", "context", "--project", "/proj"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "Hexagonal") {
		t.Errorf("expected output to contain %q; got: %q", "Hexagonal", out)
	}
}

func TestBrainCLI_Ingest_StructuredEntry_OK(t *testing.T) {
	brain := &mockBrainService{}
	root := cli.NewRootCmd(&mockOrchestrator{}, brain)
	root.SetArgs([]string{"brain", "ingest", "--project", "/proj", "--kind", "learning", "--topic", "Auth", "--content", "Use constant time"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "Auth") {
		t.Errorf("expected output to mention topic Auth; got: %q", out)
	}
}

func TestBrainCLI_Context_FocusedQuestion_OK(t *testing.T) {
	brain := &mockBrainService{
		getContextResult: domain.ContextResponse{
			ProjectPath: "/proj",
			Sections: []domain.ContextSection{
				{Topic: "Auth", Kind: domain.KnowledgeLearning, Content: "Focused answer"},
			},
		},
	}
	root := cli.NewRootCmd(&mockOrchestrator{}, brain)
	root.SetArgs([]string{"brain", "context", "--project", "/proj", "--question", "how does auth work?"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "Focused answer") {
		t.Errorf("expected output to contain focused answer; got: %q", out)
	}
}

func TestBrainCLI_Standalone_Search_OK(t *testing.T) {
	brain := &mockBrainService{
		searchResult: []domain.ContextSection{
			{Topic: "Standalone", Content: "Found in standalone"},
		},
	}
	cmd := cli.NewBrainStandaloneCmd(brain)
	cmd.SetArgs([]string{"search", "--all", "my positional search"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Found in standalone") {
		t.Errorf("expected output to contain 'Found in standalone', got: %s", buf.String())
	}
}

func TestBrainCLI_Search_EmptyQuery_Fails(t *testing.T) {
	brain := &mockBrainService{}
	cmd := cli.NewBrainCmd(brain)
	cmd.SetArgs([]string{"search"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("expected query is required error, got: %v", err)
	}
}

func TestBrainCLI_MCP_Stdio_OK(t *testing.T) {
	brain := &mockBrainService{}
	cmd := cli.NewBrainCmd(brain)
	cmd.SetArgs([]string{"mcp", "--stdio"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	var in bytes.Buffer
	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")
	cmd.SetIn(&in)
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), `"jsonrpc":"2.0"`) {
		t.Errorf("expected JSON-RPC response, got: %s", out.String())
	}
}

func TestBrainCLI_MCP_HTTP_Cancelled_OK(t *testing.T) {
	brain := &mockBrainService{}
	cmd := cli.NewBrainCmd(brain)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel context
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"mcp", "--stdio=false", "--port", "64998"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	// Starts server and exits on cancelled context
	_ = cmd.Execute()
}
