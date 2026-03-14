// Package main is the entry point for the nexus CLI binary.
// It connects to a running nexusOrchestrator daemon via the HTTP API.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var version = "dev"

func main() {
	// Use a lightweight HTTP-backed orchestrator stub that talks to the daemon.
	orch := &remoteOrchestrator{baseURL: "http://127.0.0.1:63987"}

	root := cli.NewRootCmd(orch)
	root.Version = version
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// remoteOrchestrator forwards calls to the running nexusOrchestrator HTTP API.
type remoteOrchestrator struct{ baseURL string }

func (r *remoteOrchestrator) SubmitTask(task domain.Task) (string, error) {
	body, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("remote: marshal task: %w", err)
	}
	resp, err := http.Post(r.baseURL+"/api/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("remote: submit task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("remote: submit task: unexpected status %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("remote: decode response: %w", err)
	}
	return result["task_id"], nil
}

func (r *remoteOrchestrator) GetTask(id string) (domain.Task, error) {
	resp, err := http.Get(r.baseURL + "/api/tasks/" + url.PathEscape(id))
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: get task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return domain.Task{}, fmt.Errorf("remote: get task: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return domain.Task{}, fmt.Errorf("remote: get task: unexpected status %d", resp.StatusCode)
	}

	var task domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return domain.Task{}, fmt.Errorf("remote: decode task: %w", err)
	}
	return task, nil
}

func (r *remoteOrchestrator) GetQueue() ([]domain.Task, error) {
	resp, err := http.Get(r.baseURL + "/api/tasks")
	if err != nil {
		return nil, fmt.Errorf("remote: get queue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: get queue: unexpected status %d", resp.StatusCode)
	}

	var tasks []domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("remote: decode queue: %w", err)
	}
	return tasks, nil
}

func (r *remoteOrchestrator) GetAllTasks() ([]domain.Task, error) {
	resp, err := http.Get(r.baseURL + "/api/tasks/all")
	if err != nil {
		return nil, fmt.Errorf("remote: get all tasks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: get all tasks: unexpected status %d", resp.StatusCode)
	}

	var tasks []domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("remote: decode all tasks: %w", err)
	}
	return tasks, nil
}

func (r *remoteOrchestrator) GetProviders() ([]ports.ProviderInfo, error) {
	resp, err := http.Get(r.baseURL + "/api/providers")
	if err != nil {
		return nil, fmt.Errorf("remote: get providers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: get providers: unexpected status %d", resp.StatusCode)
	}

	var providers []ports.ProviderInfo
	if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
		return nil, fmt.Errorf("remote: decode providers: %w", err)
	}
	return providers, nil
}

func (r *remoteOrchestrator) CancelTask(id string) error {
	req, err := http.NewRequest(http.MethodDelete, r.baseURL+"/api/tasks/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("remote: build cancel request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("remote: cancel task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remote: cancel task: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *remoteOrchestrator) RegisterCloudProvider(cfg domain.ProviderConfig) error {
	body, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("remote: marshal provider config: %w", err)
	}
	resp, err := http.Post(r.baseURL+"/api/providers", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("remote: register provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("remote: register provider: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *remoteOrchestrator) RemoveProvider(name string) error {
	req, err := http.NewRequest(http.MethodDelete, r.baseURL+"/api/providers/"+url.PathEscape(name), nil)
	if err != nil {
		return fmt.Errorf("remote: build remove-provider request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("remote: remove provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("remote: remove provider: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remote: remove provider: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *remoteOrchestrator) GetProviderModels(name string) ([]string, error) {
	resp, err := http.Get(r.baseURL + "/api/providers/" + url.PathEscape(name) + "/models")
	if err != nil {
		return nil, fmt.Errorf("remote: get provider models: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("remote: get provider models: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: get provider models: unexpected status %d", resp.StatusCode)
	}
	var models []string
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("remote: decode models: %w", err)
	}
	return models, nil
}

func (r *remoteOrchestrator) AddProviderConfig(ctx context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	body, err := json.Marshal(cfg)
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("remote: marshal provider config: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/api/providers/config", bytes.NewReader(body))
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("remote: build add provider config request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("remote: add provider config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return domain.ProviderConfig{}, fmt.Errorf("remote: add provider config: unexpected status %d", resp.StatusCode)
	}
	var created domain.ProviderConfig
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("remote: decode provider config: %w", err)
	}
	return created, nil
}

func (r *remoteOrchestrator) UpdateProviderConfig(ctx context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	body, err := json.Marshal(cfg)
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("remote: marshal provider config: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, r.baseURL+"/api/providers/config/"+url.PathEscape(cfg.ID), bytes.NewReader(body))
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("remote: build update provider config request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("remote: update provider config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return domain.ProviderConfig{}, fmt.Errorf("remote: update provider config: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return domain.ProviderConfig{}, fmt.Errorf("remote: update provider config: unexpected status %d", resp.StatusCode)
	}
	var updated domain.ProviderConfig
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("remote: decode provider config: %w", err)
	}
	return updated, nil
}

func (r *remoteOrchestrator) RemoveProviderConfig(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, r.baseURL+"/api/providers/config/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("remote: build remove provider config request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("remote: remove provider config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("remote: remove provider config: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remote: remove provider config: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *remoteOrchestrator) ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+"/api/providers/config", nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build list provider configs request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remote: list provider configs: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: list provider configs: unexpected status %d", resp.StatusCode)
	}
	var cfgs []domain.ProviderConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfgs); err != nil {
		return nil, fmt.Errorf("remote: decode provider configs: %w", err)
	}
	return cfgs, nil
}

func (r *remoteOrchestrator) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	resp, err := http.Get(r.baseURL + "/api/providers/discovered")
	if err != nil {
		return nil, fmt.Errorf("remote: get discovered providers: %w", err)
	}
	defer resp.Body.Close()
	var providers []domain.DiscoveredProvider
	if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
		return nil, fmt.Errorf("remote: decode discovered providers: %w", err)
	}
	return providers, nil
}

func (r *remoteOrchestrator) TriggerScan(_ context.Context) ([]domain.DiscoveredProvider, error) {
	resp, err := http.Post(r.baseURL+"/api/providers/discovered/scan", "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("remote: trigger scan: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("remote: trigger scan: unexpected status %d", resp.StatusCode)
	}
	var providers []domain.DiscoveredProvider
	_ = json.NewDecoder(resp.Body).Decode(&providers)
	return providers, nil
}

func (r *remoteOrchestrator) PromoteProvider(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/api/providers/promote/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("cli: promote provider: build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("cli: promote provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cli: promote provider: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *remoteOrchestrator) CreateDraft(task domain.Task) (string, error) {
	body, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("remote: marshal draft task: %w", err)
	}
	resp, err := http.Post(r.baseURL+"/api/tasks/draft", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("remote: create draft: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("remote: create draft: unexpected status %d", resp.StatusCode)
	}
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("remote: decode draft response: %w", err)
	}
	return result["id"], nil
}

func (r *remoteOrchestrator) GetBacklog(projectPath string) ([]domain.Task, error) {
	params := url.Values{}
	params.Set("project", projectPath)
	resp, err := http.Get(r.baseURL + "/api/tasks/backlog?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("remote: get backlog: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: get backlog: unexpected status %d", resp.StatusCode)
	}
	var tasks []domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("remote: decode backlog: %w", err)
	}
	return tasks, nil
}

func (r *remoteOrchestrator) PromoteTask(id string) error {
	resp, err := http.Post(r.baseURL+"/api/tasks/"+url.PathEscape(id)+"/promote", "application/json", nil)
	if err != nil {
		return fmt.Errorf("remote: promote task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("remote: promote task: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remote: promote task: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *remoteOrchestrator) UpdateTask(id string, updates domain.Task) (domain.Task, error) {
	body, err := json.Marshal(updates)
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: marshal task updates: %w", err)
	}
	req, err := http.NewRequest(http.MethodPut, r.baseURL+"/api/tasks/"+url.PathEscape(id), bytes.NewReader(body))
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: build update task request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: update task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return domain.Task{}, fmt.Errorf("remote: update task: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return domain.Task{}, fmt.Errorf("remote: update task: unexpected status %d", resp.StatusCode)
	}
	var updated domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		return domain.Task{}, fmt.Errorf("remote: decode updated task: %w", err)
	}
	return updated, nil
}

func (r *remoteOrchestrator) RegisterAISession(ctx context.Context, s domain.AISession) (domain.AISession, error) {
	body, err := json.Marshal(s)
	if err != nil {
		return domain.AISession{}, fmt.Errorf("remote: marshal ai session: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/api/ai-sessions", bytes.NewReader(body))
	if err != nil {
		return domain.AISession{}, fmt.Errorf("remote: build register ai session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return domain.AISession{}, fmt.Errorf("remote: register ai session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return domain.AISession{}, fmt.Errorf("remote: register ai session: unexpected status %d", resp.StatusCode)
	}
	var created domain.AISession
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return domain.AISession{}, fmt.Errorf("remote: decode ai session: %w", err)
	}
	return created, nil
}

func (r *remoteOrchestrator) ListAISessions(ctx context.Context) ([]domain.AISession, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+"/api/ai-sessions", nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build list ai sessions request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remote: list ai sessions: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: list ai sessions: unexpected status %d", resp.StatusCode)
	}
	var sessions []domain.AISession
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("remote: decode ai sessions: %w", err)
	}
	return sessions, nil
}

func (r *remoteOrchestrator) DeregisterAISession(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, r.baseURL+"/api/ai-sessions/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("remote: build deregister ai session request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("remote: deregister ai session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("remote: deregister ai session: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remote: deregister ai session: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *remoteOrchestrator) HeartbeatAISession(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/api/ai-sessions/"+url.PathEscape(id)+"/heartbeat", nil)
	if err != nil {
		return fmt.Errorf("remote: build heartbeat ai session request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("remote: heartbeat ai session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("remote: heartbeat ai session: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remote: heartbeat ai session: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *remoteOrchestrator) ClaimTask(ctx context.Context, taskID string, sessionID string) (domain.Task, error) {
	body := fmt.Sprintf(`{"sessionId":%q}`, sessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/api/tasks/"+url.PathEscape(taskID)+"/claim", strings.NewReader(body))
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: build claim task request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: claim task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return domain.Task{}, fmt.Errorf("remote: claim task: %w", domain.ErrNotFound)
	}
	var task domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return domain.Task{}, fmt.Errorf("remote: claim task: decode: %w", err)
	}
	return task, nil
}

func (r *remoteOrchestrator) UpdateTaskStatus(ctx context.Context, taskID string, sessionID string, status domain.TaskStatus, logs string) (domain.Task, error) {
	payload := struct {
		SessionID string `json:"sessionId"`
		Status    string `json:"status"`
		Logs      string `json:"logs,omitempty"`
	}{sessionID, string(status), logs}
	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, r.baseURL+"/api/tasks/"+url.PathEscape(taskID)+"/status", bytes.NewReader(data))
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: build update task status request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: update task status: %w", err)
	}
	defer resp.Body.Close()
	var task domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return domain.Task{}, fmt.Errorf("remote: update task status: decode: %w", err)
	}
	return task, nil
}

func (r *remoteOrchestrator) PurgeDisconnectedSessions(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, r.baseURL+"/api/ai-sessions", nil)
	if err != nil {
		return 0, fmt.Errorf("remote: build purge disconnected sessions request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("remote: purge disconnected sessions: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		Deleted int `json:"deleted"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("remote: purge disconnected sessions: decode: %w", err)
	}
	return result.Deleted, nil
}
func (r *remoteOrchestrator) GetDiscoveredAgents(_ context.Context) ([]domain.DiscoveredAgent, error) {
	return nil, nil
}

func (r *remoteOrchestrator) DelegateToNexus(_ context.Context, _ string) (string, error) {
	return "", nil
}
