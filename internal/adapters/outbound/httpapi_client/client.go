// Package httpapi_client provides an HTTP client that implements ports.Orchestrator
// by forwarding calls to a running nexusOrchestrator daemon via its HTTP API.
package httpapi_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// Client forwards orchestrator calls to the running nexusOrchestrator HTTP API.
type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

type submitTaskResponse struct {
	TaskID string `json:"task_id"`
}

type createDraftResponse struct {
	ID string `json:"id"`
}

type terminateAISessionRequest struct {
	Force bool `json:"force"`
}

type sessionIDRequest struct {
	SessionID string `json:"sessionId"`
}

type updateTaskStatusRequest struct {
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
	Logs      string `json:"logs,omitempty"`
}

// NewClient returns a new Client that talks to the nexusOrchestrator daemon at baseURL.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   strings.TrimSpace(os.Getenv("NEXUS_API_TOKEN")),
		client:  http.DefaultClient,
	}
}

// NewClientWithHTTP returns a Client that uses the provided *http.Client for all requests.
// This allows injecting custom timeouts, TLS config, or test transports.
func NewClientWithHTTP(baseURL string, httpClient *http.Client) *Client {
	c := NewClient(baseURL)
	c.client = httpClient
	return c
}

func (r *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}
	return req, nil
}

func (r *Client) do(req *http.Request) (*http.Response, error) {
	return r.client.Do(req)
}

func (r *Client) SubmitTask(task domain.Task) (string, error) {
	body, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("remote: marshal task: %w", err)
	}
	req, err := r.newRequest(context.Background(), http.MethodPost, "/api/tasks", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("remote: build submit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
	if err != nil {
		return "", fmt.Errorf("remote: submit task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("remote: submit task: unexpected status %d", resp.StatusCode)
	}

	var result submitTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("remote: decode response: %w", err)
	}
	return result.TaskID, nil
}

func (r *Client) GetTask(id string) (domain.Task, error) {
	req, err := r.newRequest(context.Background(), http.MethodGet, "/api/tasks/"+url.PathEscape(id), nil)
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: build get task request: %w", err)
	}
	resp, err := r.do(req)
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

func (r *Client) GetQueue() ([]domain.Task, error) {
	req, err := r.newRequest(context.Background(), http.MethodGet, "/api/tasks", nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build get queue request: %w", err)
	}
	resp, err := r.do(req)
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

func (r *Client) GetAllTasks() ([]domain.Task, error) {
	req, err := r.newRequest(context.Background(), http.MethodGet, "/api/tasks/all", nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build get all tasks request: %w", err)
	}
	resp, err := r.do(req)
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

func (r *Client) GetQueueForProject(projectPath string) ([]domain.Task, error) {
	u := "/api/tasks?projectPath=" + projectPath
	req, err := r.newRequest(context.Background(), http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build get queue for project request: %w", err)
	}
	resp, err := r.do(req)
	if err != nil {
		return nil, fmt.Errorf("remote: get queue for project: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: get queue for project: unexpected status %d", resp.StatusCode)
	}

	var tasks []domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("remote: decode queue for project: %w", err)
	}
	return tasks, nil
}

func (r *Client) GetTasksForProject(projectPath string) ([]domain.Task, error) {
	u := "/api/tasks/all?projectPath=" + projectPath
	req, err := r.newRequest(context.Background(), http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build get tasks for project request: %w", err)
	}
	resp, err := r.do(req)
	if err != nil {
		return nil, fmt.Errorf("remote: get tasks for project: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: get tasks for project: unexpected status %d", resp.StatusCode)
	}

	var tasks []domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("remote: decode tasks for project: %w", err)
	}
	return tasks, nil
}

func (r *Client) GetProviders() ([]ports.ProviderInfo, error) {
	req, err := r.newRequest(context.Background(), http.MethodGet, "/api/providers", nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build get providers request: %w", err)
	}
	resp, err := r.do(req)
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

func (r *Client) GetRuntimeConfig(ctx context.Context) (domain.RuntimeConfig, error) {
	req, err := r.newRequest(ctx, http.MethodGet, "/api/config", nil)
	if err != nil {
		return domain.RuntimeConfig{}, fmt.Errorf("remote: build get config request: %w", err)
	}
	resp, err := r.do(req)
	if err != nil {
		return domain.RuntimeConfig{}, fmt.Errorf("remote: get config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return domain.RuntimeConfig{}, fmt.Errorf("remote: get config: unexpected status %d", resp.StatusCode)
	}
	var out struct {
		QueueCap int `json:"queueCap"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return domain.RuntimeConfig{}, fmt.Errorf("remote: decode config: %w", err)
	}
	return domain.RuntimeConfig{QueueCap: out.QueueCap}, nil
}

func (r *Client) UpdateRuntimeConfig(ctx context.Context, update domain.RuntimeConfigUpdate) (domain.RuntimeConfig, error) {
	body, err := json.Marshal(update)
	if err != nil {
		return domain.RuntimeConfig{}, fmt.Errorf("remote: marshal config update: %w", err)
	}
	req, err := r.newRequest(ctx, http.MethodPut, "/api/config", bytes.NewReader(body))
	if err != nil {
		return domain.RuntimeConfig{}, fmt.Errorf("remote: build update config request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
	if err != nil {
		return domain.RuntimeConfig{}, fmt.Errorf("remote: update config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return domain.RuntimeConfig{}, fmt.Errorf("remote: update config: unexpected status %d", resp.StatusCode)
	}
	var out struct {
		QueueCap int    `json:"queueCap"`
		APIToken string `json:"apiToken,omitempty"`
		MCPToken string `json:"mcpToken,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return domain.RuntimeConfig{}, fmt.Errorf("remote: decode updated config: %w", err)
	}
	return domain.RuntimeConfig{QueueCap: out.QueueCap, APIToken: out.APIToken, MCPToken: out.MCPToken}, nil
}

func (r *Client) CancelTask(id string) error {
	req, err := r.newRequest(context.Background(), http.MethodDelete, "/api/tasks/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("remote: build cancel request: %w", err)
	}
	resp, err := r.do(req)
	if err != nil {
		return fmt.Errorf("remote: cancel task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remote: cancel task: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *Client) RegisterCloudProvider(cfg domain.ProviderConfig) error {
	body, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("remote: marshal provider config: %w", err)
	}
	req, err := r.newRequest(context.Background(), http.MethodPost, "/api/providers", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("remote: build register provider request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
	if err != nil {
		return fmt.Errorf("remote: register provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("remote: register provider: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *Client) RemoveProvider(name string) error {
	req, err := r.newRequest(context.Background(), http.MethodDelete, "/api/providers/"+url.PathEscape(name), nil)
	if err != nil {
		return fmt.Errorf("remote: build remove-provider request: %w", err)
	}
	resp, err := r.do(req)
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

func (r *Client) GetProviderModels(name string) ([]string, error) {
	req, err := r.newRequest(context.Background(), http.MethodGet, "/api/providers/"+url.PathEscape(name)+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build get provider models request: %w", err)
	}
	resp, err := r.do(req)
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

func (r *Client) AddProviderConfig(ctx context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	body, err := json.Marshal(cfg)
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("remote: marshal provider config: %w", err)
	}
	req, err := r.newRequest(ctx, http.MethodPost, "/api/providers/config", bytes.NewReader(body))
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("remote: build add provider config request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
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

func (r *Client) UpdateProviderConfig(ctx context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	body, err := json.Marshal(cfg)
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("remote: marshal provider config: %w", err)
	}
	req, err := r.newRequest(ctx, http.MethodPut, "/api/providers/config/"+url.PathEscape(cfg.ID), bytes.NewReader(body))
	if err != nil {
		return domain.ProviderConfig{}, fmt.Errorf("remote: build update provider config request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
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

func (r *Client) RemoveProviderConfig(ctx context.Context, id string) error {
	req, err := r.newRequest(ctx, http.MethodDelete, "/api/providers/config/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("remote: build remove provider config request: %w", err)
	}
	resp, err := r.do(req)
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

func (r *Client) ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error) {
	req, err := r.newRequest(ctx, http.MethodGet, "/api/providers/config", nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build list provider configs request: %w", err)
	}
	resp, err := r.do(req)
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

func (r *Client) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	req, err := r.newRequest(context.Background(), http.MethodGet, "/api/providers/discovered", nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build get discovered providers request: %w", err)
	}
	resp, err := r.do(req)
	if err != nil {
		return nil, fmt.Errorf("remote: get discovered providers: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: get discovered providers: unexpected status %d", resp.StatusCode)
	}
	var providers []domain.DiscoveredProvider
	if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
		return nil, fmt.Errorf("remote: decode discovered providers: %w", err)
	}
	return providers, nil
}

func (r *Client) TriggerScan(_ context.Context) ([]domain.DiscoveredProvider, error) {
	req, err := r.newRequest(context.Background(), http.MethodPost, "/api/providers/discovered/scan", nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build trigger scan request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
	if err != nil {
		return nil, fmt.Errorf("remote: trigger scan: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("remote: trigger scan: unexpected status %d", resp.StatusCode)
	}
	var providers []domain.DiscoveredProvider
	if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
		return nil, fmt.Errorf("remote: trigger scan: decode: %w", err)
	}
	return providers, nil
}

func (r *Client) PromoteProvider(ctx context.Context, id string) error {
	req, err := r.newRequest(ctx, http.MethodPost, "/api/providers/promote/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("cli: promote provider: build request: %w", err)
	}
	resp, err := r.do(req)
	if err != nil {
		return fmt.Errorf("cli: promote provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cli: promote provider: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *Client) CreateDraft(task domain.Task) (string, error) {
	body, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("remote: marshal draft task: %w", err)
	}
	req, err := r.newRequest(context.Background(), http.MethodPost, "/api/tasks/draft", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("remote: build create draft request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
	if err != nil {
		return "", fmt.Errorf("remote: create draft: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("remote: create draft: unexpected status %d", resp.StatusCode)
	}
	var result createDraftResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("remote: decode draft response: %w", err)
	}
	return result.ID, nil
}

func (r *Client) GetBacklog(projectPath string) ([]domain.Task, error) {
	params := url.Values{}
	params.Set("project", projectPath)
	req, err := r.newRequest(context.Background(), http.MethodGet, "/api/tasks/backlog?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build get backlog request: %w", err)
	}
	resp, err := r.do(req)
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

func (r *Client) PromoteTask(id string) (ports.PromoteResult, error) {
	req, err := r.newRequest(context.Background(), http.MethodPost, "/api/tasks/"+url.PathEscape(id)+"/promote", nil)
	if err != nil {
		return ports.PromoteResult{}, fmt.Errorf("remote: build promote task request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
	if err != nil {
		return ports.PromoteResult{}, fmt.Errorf("remote: promote task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ports.PromoteResult{}, fmt.Errorf("remote: promote task: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return ports.PromoteResult{}, fmt.Errorf("remote: promote task: unexpected status %d", resp.StatusCode)
	}
	var result ports.PromoteResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ports.PromoteResult{}, fmt.Errorf("remote: promote task: decode response: %w", err)
	}
	return result, nil
}

func (r *Client) UpdateTask(id string, updates domain.Task) (domain.Task, error) {
	body, err := json.Marshal(updates)
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: marshal task updates: %w", err)
	}
	req, err := r.newRequest(context.Background(), http.MethodPut, "/api/tasks/"+url.PathEscape(id), bytes.NewReader(body))
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: build update task request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
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

func (r *Client) RegisterAISession(ctx context.Context, s domain.AISession) (domain.AISession, error) {
	body, err := json.Marshal(s)
	if err != nil {
		return domain.AISession{}, fmt.Errorf("remote: marshal ai session: %w", err)
	}
	req, err := r.newRequest(ctx, http.MethodPost, "/api/ai-sessions", bytes.NewReader(body))
	if err != nil {
		return domain.AISession{}, fmt.Errorf("remote: build register ai session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
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

func (r *Client) ListAISessions(ctx context.Context) ([]domain.AISession, error) {
	req, err := r.newRequest(ctx, http.MethodGet, "/api/ai-sessions", nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build list ai sessions request: %w", err)
	}
	resp, err := r.do(req)
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

func (r *Client) DeregisterAISession(ctx context.Context, id string) error {
	req, err := r.newRequest(ctx, http.MethodDelete, "/api/ai-sessions/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("remote: build deregister ai session request: %w", err)
	}
	resp, err := r.do(req)
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

func (r *Client) TerminateAISession(ctx context.Context, id string, force bool) error {
	payload, err := json.Marshal(terminateAISessionRequest{Force: force})
	if err != nil {
		return fmt.Errorf("remote: marshal terminate ai session body: %w", err)
	}
	req, err := r.newRequest(ctx, http.MethodPost, "/api/ai-sessions/"+url.PathEscape(id)+"/terminate", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("remote: build terminate ai session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
	if err != nil {
		return fmt.Errorf("remote: terminate ai session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("remote: terminate ai session: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remote: terminate ai session: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *Client) HeartbeatAISession(ctx context.Context, id string) error {
	req, err := r.newRequest(ctx, http.MethodPost, "/api/ai-sessions/"+url.PathEscape(id)+"/heartbeat", nil)
	if err != nil {
		return fmt.Errorf("remote: build heartbeat ai session request: %w", err)
	}
	resp, err := r.do(req)
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

func (r *Client) HeartbeatTask(ctx context.Context, taskID, sessionID string) error {
	body, err := json.Marshal(sessionIDRequest{SessionID: sessionID})
	if err != nil {
		return fmt.Errorf("remote: marshal heartbeat task body: %w", err)
	}
	req, err := r.newRequest(ctx, http.MethodPost, "/api/tasks/"+url.PathEscape(taskID)+"/heartbeat", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("remote: build heartbeat task request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
	if err != nil {
		return fmt.Errorf("remote: heartbeat task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remote: heartbeat task: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *Client) ClaimTask(ctx context.Context, taskID string, sessionID string) (domain.Task, error) {
	body, err := json.Marshal(sessionIDRequest{SessionID: sessionID})
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: marshal claim task body: %w", err)
	}
	req, err := r.newRequest(ctx, http.MethodPost, "/api/tasks/"+url.PathEscape(taskID)+"/claim", bytes.NewReader(body))
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: build claim task request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
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

func (r *Client) UpdateTaskStatus(ctx context.Context, taskID string, sessionID string, status domain.TaskStatus, logs string) (domain.Task, error) {
	payload := updateTaskStatusRequest{
		SessionID: sessionID,
		Status:    string(status),
		Logs:      logs,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: marshal update task status body: %w", err)
	}
	req, err := r.newRequest(ctx, http.MethodPut, "/api/tasks/"+url.PathEscape(taskID)+"/status", bytes.NewReader(data))
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: build update task status request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: update task status: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return domain.Task{}, fmt.Errorf("remote: update task status: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return domain.Task{}, fmt.Errorf("remote: update task status: unexpected status %d", resp.StatusCode)
	}
	var task domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return domain.Task{}, fmt.Errorf("remote: update task status: decode: %w", err)
	}
	return task, nil
}

func (r *Client) PurgeDisconnectedSessions(ctx context.Context) (int, error) {
	req, err := r.newRequest(ctx, http.MethodDelete, "/api/ai-sessions", nil)
	if err != nil {
		return 0, fmt.Errorf("remote: build purge disconnected sessions request: %w", err)
	}
	resp, err := r.do(req)
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

func (r *Client) GetDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error) {
	req, err := r.newRequest(ctx, http.MethodGet, "/api/ai-sessions/discovered", nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build get discovered agents request: %w", err)
	}
	resp, err := r.do(req)
	if err != nil {
		return nil, fmt.Errorf("remote: get discovered agents: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: get discovered agents: unexpected status %d", resp.StatusCode)
	}
	var agents []domain.DiscoveredAgent
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		return nil, fmt.Errorf("remote: get discovered agents: decode: %w", err)
	}
	return agents, nil
}

func (r *Client) DelegateToNexus(ctx context.Context, sessionID string) (string, error) {
	req, err := r.newRequest(ctx, http.MethodPost, "/api/ai-sessions/"+url.PathEscape(sessionID)+"/delegate", nil)
	if err != nil {
		return "", fmt.Errorf("remote: build delegate to nexus request: %w", err)
	}
	resp, err := r.do(req)
	if err != nil {
		return "", fmt.Errorf("remote: delegate to nexus: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("remote: delegate to nexus: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("remote: delegate to nexus: unexpected status %d", resp.StatusCode)
	}
	var result struct {
		Instruction string `json:"instruction"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("remote: delegate to nexus: decode: %w", err)
	}
	return result.Instruction, nil
}

func (r *Client) GetDiscoveredPlanFiles(ctx context.Context, projectPath string) ([]domain.DiscoveredPlanFile, error) {
	params := url.Values{}
	params.Set("projectPath", projectPath)
	req, err := r.newRequest(ctx, http.MethodGet, "/api/plans/discovered?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build get discovered plan files request: %w", err)
	}
	resp, err := r.do(req)
	if err != nil {
		return nil, fmt.Errorf("remote: get discovered plan files: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: get discovered plan files: unexpected status %d", resp.StatusCode)
	}
	var files []domain.DiscoveredPlanFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("remote: get discovered plan files: decode: %w", err)
	}
	return files, nil
}
