//go:build integration

package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/adapters/outbound/fs_writer"
	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

type blackboxStack struct {
	orch *services.OrchestratorService
	repo *repo_sqlite.Repository

	httpSrv *httptest.Server
	mcpSrv  *httptest.Server
}

func newBlackboxStack(t *testing.T) *blackboxStack {
	t.Helper()

	dbFile, err := os.CreateTemp(t.TempDir(), "nexus-e2e-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	_ = dbFile.Close()

	repo, err := repo_sqlite.New(dbFile.Name())
	if err != nil {
		t.Fatalf("open sqlite repo: %v", err)
	}

	sessionRepo := repo_sqlite.NewSessionRepo(repo)
	aiSessionRepo := repo_sqlite.NewAISessionRepo(repo)
	orch := services.NewOrchestrator(services.NewDiscoveryService(), repo, fs_writer.New(), sessionRepo)
	orch.SetAISessionRepo(aiSessionRepo)
	hub := httpapi.NewHub()
	orch.SetBroadcaster(hub)

	knowledgeRepo := repo_sqlite.NewKnowledgeRepo(repo)
	brainSvc := services.NewBrainService(knowledgeRepo, repo)

	httpSrv := httptest.NewServer(httpapi.NewServer(orch, brainSvc, hub).Handler())
	mcpSrv := httptest.NewServer(mcp.NewMcpServer(orch, brainSvc))

	t.Cleanup(func() {
		httpSrv.Close()
		mcpSrv.Close()
		orch.Stop()
		_ = repo.Close()
	})

	return &blackboxStack{orch: orch, repo: repo, httpSrv: httpSrv, mcpSrv: mcpSrv}
}

func httpJSON(t *testing.T, client *http.Client, method string, url string, body any) (*http.Response, []byte) {
	t.Helper()
	var reqBody *bytes.Reader
	if body == nil {
		reqBody = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reqBody = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	data, err := ioReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp, data
}

func ioReadAll(r io.Reader) ([]byte, error) { return io.ReadAll(r) }

func postRPC(t *testing.T, url string, body any) rpcResp {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal rpc body: %v", err)
	}
	resp, err := http.Post(url+"/mcp", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	defer resp.Body.Close()
	var out rpcResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode rpc response: %v", err)
	}
	return out
}

type rpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeToolPayload(t *testing.T, raw json.RawMessage, dest any) {
	t.Helper()
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal tool result envelope: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected tool result content")
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), dest); err != nil {
		t.Fatalf("unmarshal tool payload: %v", err)
	}
}

func TestBlackboxHTTPDraftPromoteCancel_NoProvider(t *testing.T) {
	stack := newBlackboxStack(t)
	client := stack.httpSrv.Client()
	projectPath := t.TempDir()

	resp, body := httpJSON(t, client, http.MethodPost, stack.httpSrv.URL+"/api/tasks/draft", map[string]any{
		"projectPath": projectPath,
		"instruction": "Create a release checklist",
		"priority":    1,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 from draft create, got %d: %s", resp.StatusCode, string(body))
	}
	var created map[string]string
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode create draft response: %v", err)
	}
	id := created["id"]
	if id == "" {
		t.Fatal("expected draft id")
	}

	resp, body = httpJSON(t, client, http.MethodPost, stack.httpSrv.URL+"/api/tasks/"+id+"/promote", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from promote, got %d: %s", resp.StatusCode, string(body))
	}
	var promote map[string]any
	if err := json.Unmarshal(body, &promote); err != nil {
		t.Fatalf("decode promote response: %v", err)
	}
	if promote["warning"] == "" {
		t.Fatalf("expected promote warning when no provider exists, got %+v", promote)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, body = httpJSON(t, client, http.MethodGet, stack.httpSrv.URL+"/api/tasks/"+id, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 from get task, got %d: %s", resp.StatusCode, string(body))
		}
		var task domain.Task
		if err := json.Unmarshal(body, &task); err != nil {
			t.Fatalf("decode task response: %v", err)
		}
		if task.Status == domain.StatusNoProvider {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	resp, body = httpJSON(t, client, http.MethodDelete, stack.httpSrv.URL+"/api/tasks/"+id, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 from cancel, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestBlackboxMCPCreateDraftAndReadBacklog(t *testing.T) {
	stack := newBlackboxStack(t)
	projectPath := t.TempDir()

	created := postRPC(t, stack.mcpSrv.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "create_draft",
			"arguments": map[string]any{
				"projectPath": projectPath,
				"instruction": "Create black-box coverage",
				"priority":    1,
			},
		},
	})
	if created.Error != nil {
		t.Fatalf("unexpected create_draft error: %+v", created.Error)
	}

	registered := postRPC(t, stack.mcpSrv.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "register_session",
			"arguments": map[string]any{
				"agent_name":   "GitHub Copilot",
				"external_id":  "copilot-test",
				"project_path": projectPath,
			},
		},
	})
	if registered.Error != nil {
		t.Fatalf("unexpected register_session error: %+v", registered.Error)
	}
	var registeredPayload map[string]string
	decodeToolPayload(t, registered.Result, &registeredPayload)
	if registeredPayload["session_id"] == "" {
		t.Fatalf("expected session_id in register_session payload: %+v", registeredPayload)
	}

	backlog := postRPC(t, stack.mcpSrv.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_backlog",
			"arguments": map[string]any{
				"projectPath": projectPath,
			},
		},
	})
	if backlog.Error != nil {
		t.Fatalf("unexpected get_backlog error: %+v", backlog.Error)
	}
	var backlogPayload []domain.Task
	decodeToolPayload(t, backlog.Result, &backlogPayload)
	if len(backlogPayload) != 1 {
		t.Fatalf("expected one backlog task, got %d", len(backlogPayload))
	}
	if backlogPayload[0].ProjectPath != projectPath {
		t.Fatalf("unexpected backlog projectPath: %+v", backlogPayload[0])
	}

	howto := postRPC(t, stack.mcpSrv.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "howto_brief",
			"arguments": map[string]any{},
		},
	})
	if howto.Error != nil {
		t.Fatalf("unexpected howto_brief error: %+v", howto.Error)
	}
	var howtoPayload struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(howto.Result, &howtoPayload); err != nil {
		t.Fatalf("decode howto_brief response: %v", err)
	}
	if len(howtoPayload.Content) == 0 || howtoPayload.Content[0].Text == "" {
		t.Fatal("expected non-empty howto_brief content")
	}
}
