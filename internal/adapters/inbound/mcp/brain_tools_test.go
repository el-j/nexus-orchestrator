package mcp_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/core/domain"
)

// mockBrainForMCP implements ports.BrainService with configurable return values.
type mockBrainForMCP struct {
	contextResp     domain.ContextResponse
	contextErr      error
	focusedResp     domain.ContextResponse
	focusedErr      error
	ingestResult    domain.ProjectKnowledge
	ingestErr       error
	ingestFileCount int
	ingestFileErr   error
	searchResults   []domain.ContextSection
	searchErr       error
	fileMapPaths    []string
	fileMapErr      error
	initStatus      domain.BrainStatus
	initErr         error
	status          domain.BrainStatus
	statusErr       error
	listEntries     []domain.ProjectKnowledge
	listErr         error
	deleteErr       error
}

func (m *mockBrainForMCP) GetContext(_ context.Context, _ domain.ContextQuery) (domain.ContextResponse, error) {
	return m.contextResp, m.contextErr
}
func (m *mockBrainForMCP) GetFocusedContext(_ context.Context, _ domain.ContextQuery) (domain.ContextResponse, error) {
	return m.focusedResp, m.focusedErr
}
func (m *mockBrainForMCP) IngestKnowledge(_ context.Context, k domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	return m.ingestResult, m.ingestErr
}
func (m *mockBrainForMCP) IngestFromFile(_ context.Context, _, _ string) (int, error) {
	return m.ingestFileCount, m.ingestFileErr
}
func (m *mockBrainForMCP) SearchKnowledge(_ context.Context, _, _ string, _ int) ([]domain.ContextSection, error) {
	return m.searchResults, m.searchErr
}
func (m *mockBrainForMCP) GetFileMap(_ context.Context, _, _ string) ([]string, error) {
	return m.fileMapPaths, m.fileMapErr
}
func (m *mockBrainForMCP) InitProject(_ context.Context, _, _ string) (domain.BrainStatus, error) {
	return m.initStatus, m.initErr
}
func (m *mockBrainForMCP) GetStatus(_ context.Context, _ string) (domain.BrainStatus, error) {
	return m.status, m.statusErr
}
func (m *mockBrainForMCP) ListKnowledge(_ context.Context, _, _ string) ([]domain.ProjectKnowledge, error) {
	return m.listEntries, m.listErr
}
func (m *mockBrainForMCP) DeleteKnowledge(_ context.Context, _ string) error {
	return m.deleteErr
}

// newBrainToolServer creates a test HTTP server with the given brain mock (and a nil orch mock).
func newBrainToolServer(t *testing.T, brain *mockBrainForMCP) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(mcp.NewMcpServer(&mockOrch{}, brain))
	t.Cleanup(srv.Close)
	return srv
}

// newNilBrainToolServer creates a test HTTP server with brain=nil.
func newNilBrainToolServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(mcp.NewMcpServer(&mockOrch{}, nil))
	t.Cleanup(srv.Close)
	return srv
}

// --- Tests ---

func TestMCP_GetBrainStatus_OK(t *testing.T) {
	brain := &mockBrainForMCP{
		status: domain.BrainStatus{
			ProjectPath: "/repo",
			Initialized: true,
			EntryCount:  3,
		},
	}
	srv := newBrainToolServer(t, brain)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      100,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "get_brain_status",
			"arguments": map[string]any{"projectPath": "/repo"},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload domain.BrainStatus
	decodeToolText(t, r.Result, &payload)
	if payload.ProjectPath != "/repo" {
		t.Fatalf("expected projectPath=/repo, got %q", payload.ProjectPath)
	}
	if !payload.Initialized {
		t.Fatal("expected initialized=true")
	}
	if payload.EntryCount != 3 {
		t.Fatalf("expected entryCount=3, got %d", payload.EntryCount)
	}
}

func TestMCP_GetBrainStatus_NilBrain(t *testing.T) {
	srv := newNilBrainToolServer(t)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      101,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "get_brain_status",
			"arguments": map[string]any{"projectPath": "/repo"},
		},
	})
	if r.Error == nil {
		t.Fatal("expected error when brain is nil")
	}
}

func TestMCP_IngestKnowledge_OK(t *testing.T) {
	brain := &mockBrainForMCP{
		ingestFileCount: 7,
	}
	srv := newBrainToolServer(t, brain)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      102,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "ingest_knowledge",
			"arguments": map[string]any{
				"projectPath": "/repo",
				"filePath":    "/repo/CLAUDE.md",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload map[string]any
	decodeToolText(t, r.Result, &payload)
	count, ok := payload["ingestedSections"]
	if !ok {
		t.Fatal("expected ingestedSections key in result")
	}
	// JSON numbers unmarshal as float64
	if count.(float64) != 7 {
		t.Fatalf("expected ingestedSections=7, got %v", count)
	}
}

func TestMCP_GetProjectContext_OK(t *testing.T) {
	brain := &mockBrainForMCP{
		contextResp: domain.ContextResponse{
			ProjectPath: "/repo",
			Sections: []domain.ContextSection{
				{Kind: domain.KnowledgeArchitecture, Topic: "Overview", Content: "hexagonal", Tokens: 10},
			},
			TokensUsed:  10,
			TokenBudget: 400,
		},
	}
	srv := newBrainToolServer(t, brain)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      103,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_project_context",
			"arguments": map[string]any{
				"projectPath": "/repo",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload domain.ContextResponse
	decodeToolText(t, r.Result, &payload)
	if payload.ProjectPath != "/repo" {
		t.Fatalf("expected projectPath=/repo, got %q", payload.ProjectPath)
	}
	if len(payload.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(payload.Sections))
	}
	if payload.Sections[0].Topic != "Overview" {
		t.Fatalf("unexpected section topic: %q", payload.Sections[0].Topic)
	}
}

func TestMCP_GetFocusedContext_OK(t *testing.T) {
	brain := &mockBrainForMCP{
		focusedResp: domain.ContextResponse{
			ProjectPath: "/repo",
			Sections: []domain.ContextSection{
				{Kind: domain.KnowledgeConvention, Topic: "Tests", Content: "use table-driven tests", Tokens: 8},
			},
			TokensUsed:  8,
			TokenBudget: 400,
		},
	}
	srv := newBrainToolServer(t, brain)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      104,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_focused_context",
			"arguments": map[string]any{
				"projectPath": "/repo",
				"question":    "how are tests written?",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload domain.ContextResponse
	decodeToolText(t, r.Result, &payload)
	if len(payload.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(payload.Sections))
	}
	if payload.Sections[0].Kind != domain.KnowledgeConvention {
		t.Fatalf("unexpected section kind: %q", payload.Sections[0].Kind)
	}
}

func TestMCP_SearchKnowledge_OK(t *testing.T) {
	brain := &mockBrainForMCP{
		searchResults: []domain.ContextSection{
			{Kind: domain.KnowledgeLearning, Topic: "pitfall", Content: "avoid global state", Tokens: 5},
			{Kind: domain.KnowledgeGlossary, Topic: "orch", Content: "the orchestrator", Tokens: 4},
		},
	}
	srv := newBrainToolServer(t, brain)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      105,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "search_knowledge",
			"arguments": map[string]any{
				"projectPath": "/repo",
				"query":       "global state",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload []domain.ContextSection
	decodeToolText(t, r.Result, &payload)
	if len(payload) != 2 {
		t.Fatalf("expected 2 results, got %d", len(payload))
	}
	if payload[0].Topic != "pitfall" {
		t.Fatalf("unexpected first result topic: %q", payload[0].Topic)
	}
}

func TestMCP_InitProject_OK(t *testing.T) {
	brain := &mockBrainForMCP{
		initStatus: domain.BrainStatus{
			ProjectPath: "/repo",
			Initialized: true,
			EntryCount:  12,
		},
	}
	srv := newBrainToolServer(t, brain)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      106,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "init_project",
			"arguments": map[string]any{
				"projectPath":  "/repo",
				"claudeMDPath": "/repo/CLAUDE.md",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload domain.BrainStatus
	decodeToolText(t, r.Result, &payload)
	if !payload.Initialized {
		t.Fatal("expected initialized=true")
	}
	if payload.EntryCount != 12 {
		t.Fatalf("expected entryCount=12, got %d", payload.EntryCount)
	}
}

func TestMCP_ListKnowledge_OK(t *testing.T) {
	brain := &mockBrainForMCP{
		listEntries: []domain.ProjectKnowledge{
			{ID: "k1", ProjectPath: "/repo", Kind: domain.KnowledgeArchitecture, Topic: "arch-1"},
			{ID: "k2", ProjectPath: "/repo", Kind: domain.KnowledgeConvention, Topic: "conv-1"},
		},
	}
	srv := newBrainToolServer(t, brain)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      107,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_knowledge",
			"arguments": map[string]any{
				"projectPath": "/repo",
				"kind":        "",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload []domain.ProjectKnowledge
	decodeToolText(t, r.Result, &payload)
	if len(payload) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(payload))
	}
	if payload[0].ID != "k1" {
		t.Fatalf("unexpected first entry ID: %q", payload[0].ID)
	}
}

func TestMCP_DeleteKnowledge_OK(t *testing.T) {
	brain := &mockBrainForMCP{
		deleteErr: nil,
	}
	srv := newBrainToolServer(t, brain)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      108,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "delete_knowledge",
			"arguments": map[string]any{
				"id": "k1",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload map[string]any
	decodeToolText(t, r.Result, &payload)
	deleted, ok := payload["deleted"]
	if !ok {
		t.Fatal("expected 'deleted' key in result")
	}
	if deleted != true {
		t.Fatalf("expected deleted=true, got %v", deleted)
	}
}

func TestMCP_GetFileMap_OK(t *testing.T) {
	brain := &mockBrainForMCP{
		fileMapPaths: []string{
			"/repo/internal/core/domain/brain.go",
			"/repo/internal/adapters/inbound/mcp/brain_tools.go",
		},
	}
	srv := newBrainToolServer(t, brain)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      109,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_file_map",
			"arguments": map[string]any{
				"projectPath": "/repo",
				"focusArea":   "brain",
			},
		},
	})
	if r.Error != nil {
		t.Fatalf("unexpected error: %+v", r.Error)
	}
	var payload map[string]any
	decodeToolText(t, r.Result, &payload)
	paths, ok := payload["filePaths"]
	if !ok {
		t.Fatal("expected 'filePaths' key in result")
	}
	arr, ok := paths.([]any)
	if !ok {
		t.Fatalf("expected filePaths to be an array, got %T", paths)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 file paths, got %d", len(arr))
	}
}

func TestMCP_GetBrainStatus_ServiceError(t *testing.T) {
	brain := &mockBrainForMCP{
		statusErr: errors.New("db unavailable"),
	}
	srv := newBrainToolServer(t, brain)

	r := postRPC(t, srv, map[string]any{
		"jsonrpc": "2.0",
		"id":      110,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "get_brain_status",
			"arguments": map[string]any{"projectPath": "/repo"},
		},
	})
	if r.Error == nil {
		t.Fatal("expected error when service returns error")
	}
}
