package httpapi_client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
	"nexus-orchestrator/internal/core/domain"
)

// TestBrainClient_IngestKnowledge_OK verifies POST /api/brain/ingest decodes ingestedSections.
func TestBrainClient_IngestKnowledge_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/brain/ingest" {
			http.Error(w, "wrong route", http.StatusNotFound)
			return
		}
		assertContentType(t, r)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"ingestedSections": 3}) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewBrainClient(srv.URL)
	n, err := c.IngestFromFile(context.Background(), "/proj", "/proj/file.md")
	if err != nil {
		t.Fatalf("IngestFromFile: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 ingestedSections, got %d", n)
	}
}

// TestBrainClient_IngestKnowledge_Error verifies a 500 response surfaces an error.
func TestBrainClient_IngestKnowledge_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := httpapi_client.NewBrainClient(srv.URL)
	_, err := c.IngestFromFile(context.Background(), "/proj", "/proj/file.md")
	if err == nil {
		t.Fatal("expected error on 500 response, got nil")
	}
}

// TestBrainClient_GetBrainStatus_OK verifies GET /api/brain/status?projectPath=... decodes BrainStatus.
func TestBrainClient_GetBrainStatus_OK(t *testing.T) {
	want := domain.BrainStatus{
		ProjectPath: "/my/project",
		Initialized: true,
		EntryCount:  7,
		TotalTokens: 1234,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/brain/status" {
			http.Error(w, "wrong route", http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("projectPath") == "" {
			http.Error(w, "missing projectPath", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewBrainClient(srv.URL)
	got, err := c.GetStatus(context.Background(), "/my/project")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if got.ProjectPath != want.ProjectPath || got.EntryCount != want.EntryCount {
		t.Errorf("unexpected BrainStatus: %+v", got)
	}
}

// TestBrainClient_GetBrainStatus_Error verifies a 404 response surfaces an error.
func TestBrainClient_GetBrainStatus_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := httpapi_client.NewBrainClient(srv.URL)
	_, err := c.GetStatus(context.Background(), "/missing/project")
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
}

// TestBrainClient_GetProjectContext_OK verifies POST /api/brain/context decodes ContextResponse.
func TestBrainClient_GetProjectContext_OK(t *testing.T) {
	want := domain.ContextResponse{
		ProjectPath: "/ctx/project",
		TokensUsed:  100,
		TokenBudget: 4000,
		Sections: []domain.ContextSection{
			{Topic: "overview", Content: "hello world", Tokens: 50},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/brain/context" {
			http.Error(w, "wrong route", http.StatusNotFound)
			return
		}
		assertContentType(t, r)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewBrainClient(srv.URL)
	q := domain.ContextQuery{ProjectPath: "/ctx/project", MaxTokens: 4000}
	got, err := c.GetContext(context.Background(), q)
	if err != nil {
		t.Fatalf("GetContext: %v", err)
	}
	if got.ProjectPath != want.ProjectPath || len(got.Sections) != 1 {
		t.Errorf("unexpected ContextResponse: %+v", got)
	}
}

// TestBrainClient_GetFocusedContext_OK verifies POST /api/brain/focused-context decodes ContextResponse.
func TestBrainClient_GetFocusedContext_OK(t *testing.T) {
	want := domain.ContextResponse{
		ProjectPath: "/focus/project",
		TokensUsed:  200,
		TokenBudget: 8000,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/brain/focused-context" {
			http.Error(w, "wrong route", http.StatusNotFound)
			return
		}
		assertContentType(t, r)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewBrainClient(srv.URL)
	q := domain.ContextQuery{ProjectPath: "/focus/project", FocusArea: "tests", MaxTokens: 8000}
	got, err := c.GetFocusedContext(context.Background(), q)
	if err != nil {
		t.Fatalf("GetFocusedContext: %v", err)
	}
	if got.ProjectPath != want.ProjectPath || got.TokenBudget != want.TokenBudget {
		t.Errorf("unexpected ContextResponse: %+v", got)
	}
}

// TestBrainClient_SearchKnowledge_OK verifies GET /api/brain/search?... decodes results.
func TestBrainClient_SearchKnowledge_OK(t *testing.T) {
	wantResults := []domain.ContextSection{
		{Topic: "auth", Content: "jwt tokens", Tokens: 20},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/brain/search" {
			http.Error(w, "wrong route", http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("projectPath") == "" || r.URL.Query().Get("q") == "" {
			http.Error(w, "missing params", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"results": wantResults}) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewBrainClient(srv.URL)
	got, err := c.SearchKnowledge(context.Background(), "/my/project", "auth", 5)
	if err != nil {
		t.Fatalf("SearchKnowledge: %v", err)
	}
	if len(got) != 1 || got[0].Topic != "auth" {
		t.Errorf("unexpected search results: %+v", got)
	}
}

// TestBrainClient_InitProject_OK verifies POST /api/brain/init decodes BrainStatus.
func TestBrainClient_InitProject_OK(t *testing.T) {
	want := domain.BrainStatus{
		ProjectPath: "/init/project",
		Initialized: true,
		EntryCount:  0,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/brain/init" {
			http.Error(w, "wrong route", http.StatusNotFound)
			return
		}
		assertContentType(t, r)
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if body["projectPath"] == "" {
			t.Error("expected projectPath in body")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewBrainClient(srv.URL)
	got, err := c.InitProject(context.Background(), "/init/project", "/init/project/.claude/CLAUDE.md")
	if err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	if got.ProjectPath != want.ProjectPath || !got.Initialized {
		t.Errorf("unexpected BrainStatus: %+v", got)
	}
}

// TestBrainClient_ListKnowledge_OK verifies GET /api/brain/knowledge?projectPath=... decodes array.
func TestBrainClient_ListKnowledge_OK(t *testing.T) {
	wantItems := []domain.ProjectKnowledge{
		{ID: "k1", ProjectPath: "/list/project", Topic: "setup", Content: "do this"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/brain/knowledge" {
			http.Error(w, "wrong route", http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("projectPath") == "" {
			http.Error(w, "missing projectPath", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wantItems) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewBrainClient(srv.URL)
	got, err := c.ListKnowledge(context.Background(), "/list/project", "")
	if err != nil {
		t.Fatalf("ListKnowledge: %v", err)
	}
	if len(got) != 1 || got[0].ID != "k1" {
		t.Errorf("unexpected knowledge list: %+v", got)
	}
}

// TestBrainClient_DeleteKnowledge_OK verifies DELETE /api/brain/knowledge/{id} returns 204.
func TestBrainClient_DeleteKnowledge_OK(t *testing.T) {
	const knowledgeID = "k-delete-42"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/brain/knowledge/"+knowledgeID {
			http.Error(w, "wrong route", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := httpapi_client.NewBrainClient(srv.URL)
	if err := c.DeleteKnowledge(context.Background(), knowledgeID); err != nil {
		t.Fatalf("DeleteKnowledge: %v", err)
	}
}

// TestBrainClient_GetFileMap_OK verifies GET /api/brain/file-map?... decodes filePaths.
func TestBrainClient_GetFileMap_OK(t *testing.T) {
	wantPaths := []string{"/proj/main.go", "/proj/handler.go"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/brain/file-map" {
			http.Error(w, "wrong route", http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("projectPath") == "" {
			http.Error(w, "missing projectPath", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"filePaths": wantPaths}) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewBrainClient(srv.URL)
	got, err := c.GetFileMap(context.Background(), "/my/project", "api")
	if err != nil {
		t.Fatalf("GetFileMap: %v", err)
	}
	if len(got) != 2 || got[0] != "/proj/main.go" {
		t.Errorf("unexpected file map: %+v", got)
	}
}
