package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
)

// mockBrainService implements ports.BrainService with configurable return values
// and call counters. All fields are set at construction time and only read during
// handler execution, so no additional locking is required for these tests.
type mockBrainService struct {
	// IngestFromFile
	ingestFromFileCount int
	ingestFromFileErr   error

	// GetStatus
	retStatus    domain.BrainStatus
	retStatusErr error

	// GetContext
	retContext    domain.ContextResponse
	retContextErr error

	// GetFocusedContext
	retFocusedContext    domain.ContextResponse
	retFocusedContextErr error

	// SearchKnowledge
	retSearchResults []domain.ContextSection
	retSearchErr     error

	// InitProject
	retInitStatus domain.BrainStatus
	retInitErr    error

	// ListKnowledge
	retKnowledgeList []domain.ProjectKnowledge
	retListErr       error

	// DeleteKnowledge
	retDeleteErr error

	// GetFileMap
	retFilePaths  []string
	retFileMapErr error

	// IngestKnowledge
	retIngestKnowledge    domain.ProjectKnowledge
	retIngestKnowledgeErr error

	// call counters
	ingestFromFileCalls    int
	getStatusCalls         int
	getContextCalls        int
	getFocusedContextCalls int
	searchKnowledgeCalls   int
	initProjectCalls       int
	listKnowledgeCalls     int
	deleteKnowledgeCalls   int
	getFileMapCalls        int
	ingestKnowledgeCalls   int
}

func (m *mockBrainService) IngestFromFile(_ context.Context, _, _ string) (int, error) {
	m.ingestFromFileCalls++
	return m.ingestFromFileCount, m.ingestFromFileErr
}

func (m *mockBrainService) GetStatus(_ context.Context, _ string) (domain.BrainStatus, error) {
	m.getStatusCalls++
	return m.retStatus, m.retStatusErr
}

func (m *mockBrainService) GetContext(_ context.Context, _ domain.ContextQuery) (domain.ContextResponse, error) {
	m.getContextCalls++
	return m.retContext, m.retContextErr
}

func (m *mockBrainService) GetFocusedContext(_ context.Context, _ domain.ContextQuery) (domain.ContextResponse, error) {
	m.getFocusedContextCalls++
	return m.retFocusedContext, m.retFocusedContextErr
}

func (m *mockBrainService) SearchKnowledge(_ context.Context, _, _ string, _ int) ([]domain.ContextSection, error) {
	m.searchKnowledgeCalls++
	return m.retSearchResults, m.retSearchErr
}

func (m *mockBrainService) InitProject(_ context.Context, _, _ string) (domain.BrainStatus, error) {
	m.initProjectCalls++
	return m.retInitStatus, m.retInitErr
}

func (m *mockBrainService) ListKnowledge(_ context.Context, _, _ string) ([]domain.ProjectKnowledge, error) {
	m.listKnowledgeCalls++
	return m.retKnowledgeList, m.retListErr
}

func (m *mockBrainService) DeleteKnowledge(_ context.Context, _ string) error {
	m.deleteKnowledgeCalls++
	return m.retDeleteErr
}

func (m *mockBrainService) GetFileMap(_ context.Context, _, _ string) ([]string, error) {
	m.getFileMapCalls++
	return m.retFilePaths, m.retFileMapErr
}

func (m *mockBrainService) IngestKnowledge(_ context.Context, k domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	m.ingestKnowledgeCalls++
	return m.retIngestKnowledge, m.retIngestKnowledgeErr
}

// serveWithBrain creates a Server with a minimal mockOrchestrator and the given
// BrainService, then serves req through the server's router into rec.
func serveWithBrain(brain *mockBrainService, req *http.Request, rec *httptest.ResponseRecorder) {
	orch := &mockOrchestrator{}
	srv := httpapi.NewServer(orch, brain, nil)
	srv.Handler().ServeHTTP(rec, req)
}

// serveNilBrain creates a Server with nil brain service.
func serveNilBrain(req *http.Request, rec *httptest.ResponseRecorder) {
	orch := &mockOrchestrator{}
	srv := httpapi.NewServer(orch, nil, nil)
	srv.Handler().ServeHTTP(rec, req)
}

// --- handleIngestKnowledge ---

func TestHandleIngestKnowledge_OK(t *testing.T) {
	mock := &mockBrainService{ingestFromFileCount: 3}
	body, _ := json.Marshal(map[string]string{
		"projectPath": "/tmp/proj",
		"filePath":    "/tmp/proj/CLAUDE.md",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/brain/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	serveWithBrain(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]int
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["ingestedSections"] != 3 {
		t.Errorf("expected ingestedSections=3, got %d", resp["ingestedSections"])
	}
	if mock.ingestFromFileCalls != 1 {
		t.Errorf("expected 1 call to IngestFromFile, got %d", mock.ingestFromFileCalls)
	}
}

func TestHandleIngestKnowledge_MissingParams(t *testing.T) {
	mock := &mockBrainService{}
	// Empty body — missing projectPath and filePath
	req := httptest.NewRequest(http.MethodPost, "/api/brain/ingest", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	serveWithBrain(mock, req, rec)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleIngestKnowledge_NilBrain(t *testing.T) {
	body, _ := json.Marshal(map[string]string{
		"projectPath": "/tmp/proj",
		"filePath":    "/tmp/proj/CLAUDE.md",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/brain/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	serveNilBrain(req, rec)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// --- handleGetBrainStatus ---

func TestHandleGetBrainStatus_OK(t *testing.T) {
	mock := &mockBrainService{
		retStatus: domain.BrainStatus{
			ProjectPath: "/tmp/proj",
			Initialized: true,
			EntryCount:  5,
			KindCounts:  map[string]int{"architecture": 2, "convention": 3},
			TotalTokens: 1200,
			LastUpdated: time.Now(),
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/brain/status?projectPath=/tmp/proj", nil)
	rec := httptest.NewRecorder()

	serveWithBrain(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var resp domain.BrainStatus
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ProjectPath != "/tmp/proj" {
		t.Errorf("unexpected projectPath: %q", resp.ProjectPath)
	}
	if !resp.Initialized {
		t.Error("expected initialized=true")
	}
}

func TestHandleGetBrainStatus_MissingParam(t *testing.T) {
	mock := &mockBrainService{}
	req := httptest.NewRequest(http.MethodGet, "/api/brain/status", nil)
	rec := httptest.NewRecorder()

	serveWithBrain(mock, req, rec)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// --- handleGetProjectContext ---

func TestHandleGetProjectContext_OK(t *testing.T) {
	mock := &mockBrainService{
		retContext: domain.ContextResponse{
			ProjectPath: "/tmp/proj",
			Sections: []domain.ContextSection{
				{Kind: domain.KnowledgeArchitecture, Topic: "overview", Content: "hexagonal", Tokens: 10},
			},
			TokensUsed:  10,
			TokenBudget: 400,
		},
	}
	body, _ := json.Marshal(domain.ContextQuery{
		ProjectPath: "/tmp/proj",
		MaxTokens:   400,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/brain/context", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	serveWithBrain(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var resp domain.ContextResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ProjectPath != "/tmp/proj" {
		t.Errorf("unexpected projectPath: %q", resp.ProjectPath)
	}
	if len(resp.Sections) != 1 {
		t.Errorf("expected 1 section, got %d", len(resp.Sections))
	}
	if mock.getContextCalls != 1 {
		t.Errorf("expected 1 call to GetContext, got %d", mock.getContextCalls)
	}
}

// --- handleGetFocusedContext ---

func TestHandleGetFocusedContext_OK(t *testing.T) {
	mock := &mockBrainService{
		retFocusedContext: domain.ContextResponse{
			ProjectPath: "/tmp/proj",
			Sections: []domain.ContextSection{
				{Kind: domain.KnowledgeConvention, Topic: "naming", Content: "snake_case", Tokens: 5},
			},
			TokensUsed:  5,
			TokenBudget: 400,
		},
	}
	body, _ := json.Marshal(domain.ContextQuery{
		ProjectPath: "/tmp/proj",
		Question:    "how do we name variables?",
		MaxTokens:   400,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/brain/focused-context", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	serveWithBrain(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var resp domain.ContextResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Sections) != 1 {
		t.Errorf("expected 1 section, got %d", len(resp.Sections))
	}
	if mock.getFocusedContextCalls != 1 {
		t.Errorf("expected 1 call to GetFocusedContext, got %d", mock.getFocusedContextCalls)
	}
}

// --- handleSearchKnowledge ---

func TestHandleSearchKnowledge_OK(t *testing.T) {
	mock := &mockBrainService{
		retSearchResults: []domain.ContextSection{
			{Kind: domain.KnowledgeLearning, Topic: "pitfalls", Content: "avoid X", Tokens: 8},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/brain/search?projectPath=/tmp/proj&q=pitfalls", nil)
	rec := httptest.NewRecorder()

	serveWithBrain(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string][]domain.ContextSection
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	results, ok := resp["results"]
	if !ok {
		t.Fatal("response missing 'results' key")
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if mock.searchKnowledgeCalls != 1 {
		t.Errorf("expected 1 call to SearchKnowledge, got %d", mock.searchKnowledgeCalls)
	}
}

// --- handleInitProject ---

func TestHandleInitProject_OK(t *testing.T) {
	mock := &mockBrainService{
		retInitStatus: domain.BrainStatus{
			ProjectPath: "/tmp/proj",
			Initialized: true,
			EntryCount:  7,
		},
	}
	body, _ := json.Marshal(map[string]string{"projectPath": "/tmp/proj"})
	req := httptest.NewRequest(http.MethodPost, "/api/brain/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	serveWithBrain(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var resp domain.BrainStatus
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Initialized {
		t.Error("expected initialized=true")
	}
	if mock.initProjectCalls != 1 {
		t.Errorf("expected 1 call to InitProject, got %d", mock.initProjectCalls)
	}
}

// --- handleListKnowledge ---

func TestHandleListKnowledge_OK(t *testing.T) {
	now := time.Now()
	mock := &mockBrainService{
		retKnowledgeList: []domain.ProjectKnowledge{
			{
				ID:          "abc-123",
				ProjectPath: "/tmp/proj",
				Kind:        domain.KnowledgeArchitecture,
				Topic:       "overview",
				Content:     "hexagonal architecture",
				TokenCount:  10,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/brain/knowledge?projectPath=/tmp/proj", nil)
	rec := httptest.NewRecorder()

	serveWithBrain(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var resp []domain.ProjectKnowledge
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 entry, got %d", len(resp))
	}
	if resp[0].ID != "abc-123" {
		t.Errorf("unexpected ID: %q", resp[0].ID)
	}
	if mock.listKnowledgeCalls != 1 {
		t.Errorf("expected 1 call to ListKnowledge, got %d", mock.listKnowledgeCalls)
	}
}

// --- handleDeleteKnowledge ---

func TestHandleDeleteKnowledge_OK(t *testing.T) {
	mock := &mockBrainService{}
	req := httptest.NewRequest(http.MethodDelete, "/api/brain/knowledge/some-id", nil)
	rec := httptest.NewRecorder()

	serveWithBrain(mock, req, rec)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d; body: %s", rec.Code, rec.Body.String())
	}
	if mock.deleteKnowledgeCalls != 1 {
		t.Errorf("expected 1 call to DeleteKnowledge, got %d", mock.deleteKnowledgeCalls)
	}
}

func TestHandleDeleteKnowledge_NotFound(t *testing.T) {
	mock := &mockBrainService{retDeleteErr: domain.ErrNotFound}
	req := httptest.NewRequest(http.MethodDelete, "/api/brain/knowledge/missing-id", nil)
	rec := httptest.NewRecorder()

	serveWithBrain(mock, req, rec)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// --- handleGetFileMap ---

func TestHandleGetFileMap_OK(t *testing.T) {
	mock := &mockBrainService{
		retFilePaths: []string{
			"/tmp/proj/main.go",
			"/tmp/proj/internal/core/domain/task.go",
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/brain/file-map?projectPath=/tmp/proj", nil)
	rec := httptest.NewRecorder()

	serveWithBrain(mock, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var resp map[string][]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	filePaths, ok := resp["filePaths"]
	if !ok {
		t.Fatal("response missing 'filePaths' key")
	}
	if len(filePaths) != 2 {
		t.Errorf("expected 2 file paths, got %d", len(filePaths))
	}
	if mock.getFileMapCalls != 1 {
		t.Errorf("expected 1 call to GetFileMap, got %d", mock.getFileMapCalls)
	}
}
