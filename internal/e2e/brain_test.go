//go:build integration

package e2e_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func TestBlackboxBrainE2E(t *testing.T) {
	stack := newBlackboxStack(t)
	client := stack.httpSrv.Client()
	projectPath := t.TempDir()

	// 1. Create a dummy CLAUDE.md for ingestion
	claudeMD := filepath.Join(projectPath, "CLAUDE.md")
	content := `## Project Conventions
- Do A which is a very important convection for us.
- Do B which we rely on every single day for this task.

## Architecture
- Layer 1 is the front-end layer that users interact with.
- Layer 2 is the back-end layer providing the API surface.
`
	if err := os.WriteFile(claudeMD, []byte(content), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	// 2. Ingest
	ingestReq := map[string]any{
		"projectPath": projectPath,
		"filePath":    claudeMD,
	}
	resp, body := httpJSON(t, client, http.MethodPost, stack.httpSrv.URL+"/api/brain/ingest", ingestReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from ingest, got %d: %s", resp.StatusCode, string(body))
	}
	var ingestRes map[string]int
	if err := json.Unmarshal(body, &ingestRes); err != nil {
		t.Fatalf("decode ingest response: %v", err)
	}
	if ingestRes["ingestedSections"] != 2 {
		t.Fatalf("expected 2 sections ingested, got %d", ingestRes["ingestedSections"])
	}

	// 3. Status
	resp, body = httpJSON(t, client, http.MethodGet, stack.httpSrv.URL+"/api/brain/status?projectPath="+projectPath, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from status, got %d: %s", resp.StatusCode, string(body))
	}
	var status domain.BrainStatus
	if err := json.Unmarshal(body, &status); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if status.EntryCount != 2 {
		t.Fatalf("expected 2 total entries in status, got %d", status.EntryCount)
	}

	// 4. Get Project Context
	ctxReq := domain.ContextQuery{
		ProjectPath: projectPath,
		MaxTokens:   500,
	}
	resp, body = httpJSON(t, client, http.MethodPost, stack.httpSrv.URL+"/api/brain/context", ctxReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from context, got %d: %s", resp.StatusCode, string(body))
	}
	var ctxRes domain.ContextResponse
	if err := json.Unmarshal(body, &ctxRes); err != nil {
		t.Fatalf("decode context response: %v", err)
	}
	if len(ctxRes.Sections) != 2 {
		t.Fatalf("expected 2 sections in context, got %d", len(ctxRes.Sections))
	}

	// 5. Get Focused Context
	fCtxReq := domain.ContextQuery{
		ProjectPath: projectPath,
		Question:    "layer",
		MaxTokens:   500,
	}
	resp, body = httpJSON(t, client, http.MethodPost, stack.httpSrv.URL+"/api/brain/focused-context", fCtxReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from focused-context, got %d: %s", resp.StatusCode, string(body))
	}
	var fCtxRes domain.ContextResponse
	if err := json.Unmarshal(body, &fCtxRes); err != nil {
		t.Fatalf("decode focused-context response: %v", err)
	}
	if len(fCtxRes.Sections) == 0 {
		t.Fatalf("expected sections in focused context")
	}

	// 6. MCP
	mcpIngest := postRPC(t, stack.mcpSrv.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "ingest_knowledge",
			"arguments": map[string]any{
				"projectPath": projectPath,
				"filePath":    claudeMD,
			},
		},
	})
	if mcpIngest.Error != nil {
		t.Fatalf("unexpected ingest_knowledge error: %+v", mcpIngest.Error)
	}
	var mcpIngestRes map[string]int
	decodeToolPayload(t, mcpIngest.Result, &mcpIngestRes)
	if mcpIngestRes["ingestedSections"] != 2 {
		t.Fatalf("expected 2 sections ingested via mcp, got %d", mcpIngestRes["ingestedSections"])
	}

	mcpCtx := postRPC(t, stack.mcpSrv.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_project_context",
			"arguments": map[string]any{
				"projectPath": projectPath,
			},
		},
	})
	if mcpCtx.Error != nil {
		t.Fatalf("unexpected get_project_context error: %+v", mcpCtx.Error)
	}
	var mcpCtxRes domain.ContextResponse
	decodeToolPayload(t, mcpCtx.Result, &mcpCtxRes)
	if len(mcpCtxRes.Sections) != 2 {
		t.Fatalf("expected 2 sections via mcp context, got %d", len(mcpCtxRes.Sections))
	}

	// 7. Re-ingest dedup: ingesting the same file again must not create duplicate entries.
	resp, body = httpJSON(t, client, http.MethodPost, stack.httpSrv.URL+"/api/brain/ingest", ingestReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from re-ingest, got %d: %s", resp.StatusCode, string(body))
	}
	resp, body = httpJSON(t, client, http.MethodGet, stack.httpSrv.URL+"/api/brain/status?projectPath="+projectPath, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from status after re-ingest, got %d: %s", resp.StatusCode, string(body))
	}
	var statusAfterReIngest domain.BrainStatus
	if err := json.Unmarshal(body, &statusAfterReIngest); err != nil {
		t.Fatalf("decode status response after re-ingest: %v", err)
	}
	if statusAfterReIngest.EntryCount != 2 {
		t.Fatalf("re-ingest dedup: expected 2 entries, got %d", statusAfterReIngest.EntryCount)
	}

	// 8. Search via HTTP.
	resp, body = httpJSON(t, client, http.MethodGet,
		stack.httpSrv.URL+"/api/brain/search?projectPath="+projectPath+"&q=architecture&limit=5", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from search, got %d: %s", resp.StatusCode, string(body))
	}
	var searchRes struct {
		Results []domain.ProjectKnowledge `json:"results"`
	}
	if err := json.Unmarshal(body, &searchRes); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if len(searchRes.Results) < 1 {
		t.Fatalf("expected at least 1 search result for 'architecture', got %d", len(searchRes.Results))
	}

	// 9. Search via MCP.
	mcpSearch := postRPC(t, stack.mcpSrv.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "search_knowledge",
			"arguments": map[string]any{
				"projectPath": projectPath,
				"query":       "architecture",
				"limit":       3,
			},
		},
	})
	if mcpSearch.Error != nil {
		t.Fatalf("unexpected search_knowledge error: %+v", mcpSearch.Error)
	}
	var mcpSearchResults []domain.ProjectKnowledge
	decodeToolPayload(t, mcpSearch.Result, &mcpSearchResults)
	if len(mcpSearchResults) < 1 {
		t.Fatalf("expected at least 1 search result via mcp search_knowledge, got %d", len(mcpSearchResults))
	}
}
