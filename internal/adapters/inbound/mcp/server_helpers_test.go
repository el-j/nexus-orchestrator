package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

type mockOrch struct {
	submitID   string
	submitErr  error
	submitTask domain.Task // captures the most recent SubmitTask argument
	getTask    domain.Task
	getErr     error
	queue      []domain.Task
	queueErr   error
	cancelErr  error
	providers  []ports.ProviderInfo
	provErr    error

	draftTask      domain.Task // captures the most recent CreateDraft argument
	createDraftID  string
	createDraftErr error
	backlogTasks   []domain.Task
	backlogErr     error
	promoteErr     error
	updateArg      domain.Task
	updateResult   domain.Task
	updateErr      error

	triggerScanResult  []domain.DiscoveredProvider
	triggerScanErr     error
	promoteProviderErr error
}

func (m *mockOrch) SubmitTask(t domain.Task) (string, error) {
	m.submitTask = t
	return m.submitID, m.submitErr
}

func (m *mockOrch) GetTask(_ string) (domain.Task, error) { return m.getTask, m.getErr }

func (m *mockOrch) GetQueue() ([]domain.Task, error) { return m.queue, m.queueErr }

func (m *mockOrch) GetQueueForProject(projectPath string) ([]domain.Task, error) {
	var out []domain.Task
	for _, t := range m.queue {
		if t.ProjectPath == projectPath {
			out = append(out, t)
		}
	}
	return out, m.queueErr
}

func (m *mockOrch) GetAllTasks() ([]domain.Task, error) { return m.queue, m.queueErr }

func (m *mockOrch) GetTasksForProject(projectPath string) ([]domain.Task, error) {
	var out []domain.Task
	for _, t := range m.queue {
		if t.ProjectPath == projectPath {
			out = append(out, t)
		}
	}
	return out, m.queueErr
}

func (m *mockOrch) CancelTask(_ string) error { return m.cancelErr }

func (m *mockOrch) GetProviders() ([]ports.ProviderInfo, error) {
	return m.providers, m.provErr
}

func (m *mockOrch) RegisterCloudProvider(_ domain.ProviderConfig) error { return nil }

func (m *mockOrch) RemoveProvider(_ string) error { return nil }

func (m *mockOrch) GetProviderModels(_ string) ([]string, error) { return nil, nil }

func (m *mockOrch) AddProviderConfig(_ context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	return cfg, nil
}

func (m *mockOrch) UpdateProviderConfig(_ context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	return cfg, nil
}

func (m *mockOrch) RemoveProviderConfig(_ context.Context, _ string) error { return nil }

func (m *mockOrch) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	return nil, nil
}

func (m *mockOrch) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	return nil, nil
}

func (m *mockOrch) TriggerScan(_ context.Context) ([]domain.DiscoveredProvider, error) {
	return m.triggerScanResult, m.triggerScanErr
}

func (m *mockOrch) PromoteProvider(_ context.Context, _ string) error { return m.promoteProviderErr }

func (m *mockOrch) CreateDraft(t domain.Task) (string, error) {
	m.draftTask = t
	return m.createDraftID, m.createDraftErr
}

func (m *mockOrch) GetBacklog(_ string) ([]domain.Task, error) { return m.backlogTasks, m.backlogErr }

func (m *mockOrch) PromoteTask(_ string) (ports.PromoteResult, error) {
	return ports.PromoteResult{Promoted: m.promoteErr == nil}, m.promoteErr
}

func (m *mockOrch) UpdateTask(_ string, t domain.Task) (domain.Task, error) {
	m.updateArg = t
	return m.updateResult, m.updateErr
}

func (m *mockOrch) RegisterAISession(_ context.Context, s domain.AISession) (domain.AISession, error) {
	return s, nil
}

func (m *mockOrch) ListAISessions(_ context.Context) ([]domain.AISession, error) { return nil, nil }

func (m *mockOrch) DeregisterAISession(_ context.Context, _ string) error { return nil }

func (m *mockOrch) HeartbeatAISession(_ context.Context, _ string) error { return nil }

func (m *mockOrch) ClaimTask(_ context.Context, _ string, _ string) (domain.Task, error) {
	return domain.Task{}, nil
}

func (m *mockOrch) UpdateTaskStatus(_ context.Context, _ string, _ string, _ domain.TaskStatus, _ string) (domain.Task, error) {
	return domain.Task{}, nil
}

func (m *mockOrch) PurgeDisconnectedSessions(_ context.Context) (int, error) {
	return 0, nil
}

func (m *mockOrch) GetDiscoveredAgents(_ context.Context) ([]domain.DiscoveredAgent, error) {
	return nil, nil
}

func (m *mockOrch) DelegateToNexus(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (m *mockOrch) HeartbeatTask(_ context.Context, _, _ string) error { return nil }

func (m *mockOrch) TerminateAISession(_ context.Context, _ string, _ bool) error { return nil }

func (m *mockOrch) GetDiscoveredPlanFiles(_ context.Context, _ string) ([]domain.DiscoveredPlanFile, error) {
	return nil, nil
}

func (m *mockOrch) GetRuntimeConfig(_ context.Context) (domain.RuntimeConfig, error) {
	return domain.RuntimeConfig{QueueCap: 50}, nil
}

func (m *mockOrch) UpdateRuntimeConfig(_ context.Context, update domain.RuntimeConfigUpdate) (domain.RuntimeConfig, error) {
	cfg := domain.RuntimeConfig{QueueCap: 50}
	if update.QueueCap != nil {
		cfg.QueueCap = *update.QueueCap
	}
	if update.APIToken != nil {
		cfg.APIToken = *update.APIToken
	}
	if update.MCPToken != nil {
		cfg.MCPToken = *update.MCPToken
	}
	return cfg, nil
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

func postRPC(t *testing.T, srv *httptest.Server, body any) rpcResp {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/mcp", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	defer resp.Body.Close()
	var r rpcResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return r
}

func postRPCRaw(t *testing.T, srv *httptest.Server, body any, authHeader string) (int, string) {
	t.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/mcp", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(payload)
}

func newServer(t *testing.T, orch *mockOrch) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(mcp.NewMcpServer(orch, nil))
	t.Cleanup(srv.Close)
	return srv
}
