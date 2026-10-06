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
	"slices"
	"strings"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// maxErrorBody bounds how much of an error response is read to extract the
// server's message.
const maxErrorBody = 4 << 10

// Compile-time proof that Client satisfies the orchestrator port.
var _ ports.Orchestrator = (*Client)(nil)

// Client forwards orchestrator calls to the running nexusOrchestrator HTTP API.
type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewClient returns a new Client that talks to the nexusOrchestrator daemon at baseURL.
// The bearer token, when the daemon requires one, is read from NEXUS_API_TOKEN.
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

// call describes one daemon request.
type call struct {
	op       string // short operation name used in error messages, e.g. "submit task"
	method   string
	path     string // request path, including any already-encoded query string
	body     any    // marshalled to JSON when non-nil
	ok       []int  // accepted status codes; defaults to {200}
	notFound bool   // map HTTP 404 to domain.ErrNotFound
}

// send performs c and, when out is non-nil and the response carries a body,
// decodes the JSON response into out. All failures are wrapped as
// "remote: <op>: ..."; a 404 wraps domain.ErrNotFound when c.notFound is set,
// and any other unexpected status includes the server's {"error": "..."} message.
func (r *Client) send(ctx context.Context, c call, out any) error {
	var reader io.Reader
	if c.body != nil {
		payload, err := json.Marshal(c.body)
		if err != nil {
			return fmt.Errorf("remote: %s: marshal request: %w", c.op, err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := r.newRequest(ctx, c.method, c.path, reader)
	if err != nil {
		return fmt.Errorf("remote: %s: build request: %w", c.op, err)
	}
	if c.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("remote: %s: %w", c.op, err)
	}
	defer resp.Body.Close()

	if c.notFound && resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("remote: %s: %w", c.op, domain.ErrNotFound)
	}
	ok := c.ok
	if len(ok) == 0 {
		ok = []int{http.StatusOK}
	}
	if !slices.Contains(ok, resp.StatusCode) {
		return fmt.Errorf("remote: %s: unexpected status %d%s", c.op, resp.StatusCode, serverMessage(resp.Body))
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("remote: %s: decode response: %w", c.op, err)
	}
	return nil
}

// serverMessage extracts ": <message>" from a {"error":"<message>"} body, or
// returns "" when the body has no such field.
func serverMessage(body io.Reader) string {
	var e struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(body, maxErrorBody)).Decode(&e); err != nil || e.Error == "" {
		return ""
	}
	return ": " + e.Error
}

// query returns "?key=value" with both parts URL-escaped.
func query(key, value string) string {
	return "?" + url.Values{key: {value}}.Encode()
}

func esc(s string) string { return url.PathEscape(s) }

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

var (
	statusCreated = []int{http.StatusCreated}
	statusNoBody  = []int{http.StatusNoContent}
	statusOKOrNil = []int{http.StatusOK, http.StatusNoContent}
	statusAny2xx  = []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent}
)

// ── Tasks ────────────────────────────────────────────────────────────────────

// SubmitTask queues task and returns its new ID.
func (r *Client) SubmitTask(task domain.Task) (string, error) {
	var out submitTaskResponse
	err := r.send(context.Background(), call{op: "submit task", method: http.MethodPost, path: "/api/tasks", body: task, ok: statusCreated}, &out)
	return out.TaskID, err
}

// GetTask returns the task with the given ID, or domain.ErrNotFound.
func (r *Client) GetTask(id string) (domain.Task, error) {
	var out domain.Task
	err := r.send(context.Background(), call{op: "get task", method: http.MethodGet, path: "/api/tasks/" + esc(id), notFound: true}, &out)
	return out, err
}

// GetQueue returns every QUEUED or PROCESSING task.
func (r *Client) GetQueue() ([]domain.Task, error) {
	var out []domain.Task
	err := r.send(context.Background(), call{op: "get queue", method: http.MethodGet, path: "/api/tasks"}, &out)
	return out, err
}

// GetAllTasks returns every task regardless of status.
func (r *Client) GetAllTasks() ([]domain.Task, error) {
	var out []domain.Task
	err := r.send(context.Background(), call{op: "get all tasks", method: http.MethodGet, path: "/api/tasks/all"}, &out)
	return out, err
}

// GetQueueForProject returns QUEUED and PROCESSING tasks of one project.
func (r *Client) GetQueueForProject(projectPath string) ([]domain.Task, error) {
	var out []domain.Task
	err := r.send(context.Background(), call{op: "get queue for project", method: http.MethodGet, path: "/api/tasks" + query("projectPath", projectPath)}, &out)
	return out, err
}

// GetTasksForProject returns every task of one project.
func (r *Client) GetTasksForProject(projectPath string) ([]domain.Task, error) {
	var out []domain.Task
	err := r.send(context.Background(), call{op: "get tasks for project", method: http.MethodGet, path: "/api/tasks/all" + query("projectPath", projectPath)}, &out)
	return out, err
}

// CancelTask cancels a task that has not started executing.
func (r *Client) CancelTask(id string) error {
	return r.send(context.Background(), call{op: "cancel task", method: http.MethodDelete, path: "/api/tasks/" + esc(id), ok: statusNoBody, notFound: true}, nil)
}

// CreateDraft stores task as a DRAFT and returns its ID.
func (r *Client) CreateDraft(task domain.Task) (string, error) {
	var out createDraftResponse
	err := r.send(context.Background(), call{op: "create draft", method: http.MethodPost, path: "/api/tasks/draft", body: task, ok: statusCreated}, &out)
	return out.ID, err
}

// GetBacklog returns DRAFT and BACKLOG tasks of a project.
func (r *Client) GetBacklog(projectPath string) ([]domain.Task, error) {
	var out []domain.Task
	err := r.send(context.Background(), call{op: "get backlog", method: http.MethodGet, path: "/api/tasks/backlog" + query("project", projectPath)}, &out)
	return out, err
}

// PromoteTask moves a DRAFT or BACKLOG task into the execution queue.
func (r *Client) PromoteTask(id string) (ports.PromoteResult, error) {
	var out ports.PromoteResult
	err := r.send(context.Background(), call{op: "promote task", method: http.MethodPost, path: "/api/tasks/" + esc(id) + "/promote", notFound: true}, &out)
	return out, err
}

// UpdateTask patches the mutable fields of a task.
func (r *Client) UpdateTask(id string, updates domain.Task) (domain.Task, error) {
	var out domain.Task
	err := r.send(context.Background(), call{op: "update task", method: http.MethodPut, path: "/api/tasks/" + esc(id), body: updates, notFound: true}, &out)
	return out, err
}

// ClaimTask binds a QUEUED task to an AI session and marks it PROCESSING.
// Any non-200 answer (for example 409 when the task is no longer QUEUED) is an
// error; a rejected claim is never reported as a successful empty task.
func (r *Client) ClaimTask(ctx context.Context, taskID string, sessionID string) (domain.Task, error) {
	var out domain.Task
	err := r.send(ctx, call{op: "claim task", method: http.MethodPost, path: "/api/tasks/" + esc(taskID) + "/claim", body: sessionIDRequest{SessionID: sessionID}, notFound: true}, &out)
	return out, err
}

// UpdateTaskStatus reports completion or failure of a task claimed by sessionID.
func (r *Client) UpdateTaskStatus(ctx context.Context, taskID string, sessionID string, status domain.TaskStatus, logs string) (domain.Task, error) {
	var out domain.Task
	body := updateTaskStatusRequest{SessionID: sessionID, Status: string(status), Logs: logs}
	err := r.send(ctx, call{op: "update task status", method: http.MethodPut, path: "/api/tasks/" + esc(taskID) + "/status", body: body, notFound: true}, &out)
	return out, err
}

// HeartbeatTask keeps a PROCESSING task alive.
func (r *Client) HeartbeatTask(ctx context.Context, taskID, sessionID string) error {
	return r.send(ctx, call{op: "heartbeat task", method: http.MethodPost, path: "/api/tasks/" + esc(taskID) + "/heartbeat", body: sessionIDRequest{SessionID: sessionID}, ok: statusOKOrNil, notFound: true}, nil)
}

// ── Providers ────────────────────────────────────────────────────────────────

// GetProviders lists the active providers.
func (r *Client) GetProviders() ([]ports.ProviderInfo, error) {
	var out []ports.ProviderInfo
	err := r.send(context.Background(), call{op: "get providers", method: http.MethodGet, path: "/api/providers"}, &out)
	return out, err
}

// RegisterCloudProvider registers an adapter for cfg at runtime.
func (r *Client) RegisterCloudProvider(cfg domain.ProviderConfig) error {
	return r.send(context.Background(), call{op: "register provider", method: http.MethodPost, path: "/api/providers", body: cfg, ok: statusCreated}, nil)
}

// RemoveProvider deregisters the named provider.
func (r *Client) RemoveProvider(name string) error {
	return r.send(context.Background(), call{op: "remove provider", method: http.MethodDelete, path: "/api/providers/" + esc(name), ok: statusNoBody, notFound: true}, nil)
}

// GetProviderModels lists the model catalogue of the named provider.
func (r *Client) GetProviderModels(name string) ([]string, error) {
	var out []string
	err := r.send(context.Background(), call{op: "get provider models", method: http.MethodGet, path: "/api/providers/" + esc(name) + "/models", notFound: true}, &out)
	return out, err
}

// AddProviderConfig persists a new provider configuration.
func (r *Client) AddProviderConfig(ctx context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	var out domain.ProviderConfig
	err := r.send(ctx, call{op: "add provider config", method: http.MethodPost, path: "/api/providers/config", body: cfg, ok: statusCreated}, &out)
	return out, err
}

// UpdateProviderConfig overwrites an existing provider configuration.
func (r *Client) UpdateProviderConfig(ctx context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	var out domain.ProviderConfig
	err := r.send(ctx, call{op: "update provider config", method: http.MethodPut, path: "/api/providers/config/" + esc(cfg.ID), body: cfg, notFound: true}, &out)
	return out, err
}

// RemoveProviderConfig deletes a persisted provider configuration.
func (r *Client) RemoveProviderConfig(ctx context.Context, id string) error {
	return r.send(ctx, call{op: "remove provider config", method: http.MethodDelete, path: "/api/providers/config/" + esc(id), ok: statusNoBody, notFound: true}, nil)
}

// ListProviderConfigs returns all persisted provider configurations.
func (r *Client) ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error) {
	var out []domain.ProviderConfig
	err := r.send(ctx, call{op: "list provider configs", method: http.MethodGet, path: "/api/providers/config"}, &out)
	return out, err
}

// GetDiscoveredProviders returns auto-detected providers that are not yet promoted.
func (r *Client) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	var out []domain.DiscoveredProvider
	err := r.send(context.Background(), call{op: "get discovered providers", method: http.MethodGet, path: "/api/providers/discovered"}, &out)
	return out, err
}

// TriggerScan asks the daemon to rescan the machine for providers. A 204 reply
// means the scan found nothing and yields an empty (non-nil) slice.
func (r *Client) TriggerScan(ctx context.Context) ([]domain.DiscoveredProvider, error) {
	out := []domain.DiscoveredProvider{}
	err := r.send(ctx, call{op: "trigger scan", method: http.MethodPost, path: "/api/providers/discovered/scan", ok: statusOKOrNil}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PromoteProvider promotes a discovered provider to a configured one.
func (r *Client) PromoteProvider(ctx context.Context, id string) error {
	return r.send(ctx, call{op: "promote provider", method: http.MethodPost, path: "/api/providers/promote/" + esc(id), ok: statusAny2xx, notFound: true}, nil)
}

// ── Runtime configuration ────────────────────────────────────────────────────

// GetRuntimeConfig returns the daemon's runtime configuration. Only the queue
// cap is exposed over GET; tokens are never returned by that endpoint.
func (r *Client) GetRuntimeConfig(ctx context.Context) (domain.RuntimeConfig, error) {
	var out struct {
		QueueCap int `json:"queueCap"`
	}
	err := r.send(ctx, call{op: "get config", method: http.MethodGet, path: "/api/config"}, &out)
	return domain.RuntimeConfig{QueueCap: out.QueueCap}, err
}

// UpdateRuntimeConfig applies a partial update and returns the resulting
// configuration, including any newly rotated tokens.
func (r *Client) UpdateRuntimeConfig(ctx context.Context, update domain.RuntimeConfigUpdate) (domain.RuntimeConfig, error) {
	var out struct {
		QueueCap int    `json:"queueCap"`
		APIToken string `json:"apiToken,omitempty"`
		MCPToken string `json:"mcpToken,omitempty"`
	}
	err := r.send(ctx, call{op: "update config", method: http.MethodPut, path: "/api/config", body: update}, &out)
	return domain.RuntimeConfig{QueueCap: out.QueueCap, APIToken: out.APIToken, MCPToken: out.MCPToken}, err
}

// ── AI sessions and agents ───────────────────────────────────────────────────

// RegisterAISession registers (or refreshes) an external agent session.
func (r *Client) RegisterAISession(ctx context.Context, s domain.AISession) (domain.AISession, error) {
	var out domain.AISession
	err := r.send(ctx, call{op: "register ai session", method: http.MethodPost, path: "/api/ai-sessions", body: s, ok: statusCreated}, &out)
	return out, err
}

// ListAISessions returns all persisted AI sessions.
func (r *Client) ListAISessions(ctx context.Context) ([]domain.AISession, error) {
	var out []domain.AISession
	err := r.send(ctx, call{op: "list ai sessions", method: http.MethodGet, path: "/api/ai-sessions"}, &out)
	return out, err
}

// DeregisterAISession marks a session disconnected.
func (r *Client) DeregisterAISession(ctx context.Context, id string) error {
	return r.send(ctx, call{op: "deregister ai session", method: http.MethodDelete, path: "/api/ai-sessions/" + esc(id), ok: statusNoBody, notFound: true}, nil)
}

// TerminateAISession asks the daemon to stop the agent process behind a session.
func (r *Client) TerminateAISession(ctx context.Context, id string, force bool) error {
	return r.send(ctx, call{op: "terminate ai session", method: http.MethodPost, path: "/api/ai-sessions/" + esc(id) + "/terminate", body: terminateAISessionRequest{Force: force}, ok: statusOKOrNil, notFound: true}, nil)
}

// HeartbeatAISession refreshes a session's last-activity timestamp.
func (r *Client) HeartbeatAISession(ctx context.Context, id string) error {
	return r.send(ctx, call{op: "heartbeat ai session", method: http.MethodPost, path: "/api/ai-sessions/" + esc(id) + "/heartbeat", ok: statusNoBody, notFound: true}, nil)
}

// PurgeDisconnectedSessions deletes long-disconnected sessions and returns how many.
func (r *Client) PurgeDisconnectedSessions(ctx context.Context) (int, error) {
	var out struct {
		Deleted int `json:"deleted"`
	}
	err := r.send(ctx, call{op: "purge disconnected sessions", method: http.MethodDelete, path: "/api/ai-sessions"}, &out)
	return out.Deleted, err
}

// GetDiscoveredAgents returns AI agent processes found on the machine.
func (r *Client) GetDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error) {
	var out []domain.DiscoveredAgent
	err := r.send(ctx, call{op: "get discovered agents", method: http.MethodGet, path: "/api/ai-sessions/discovered"}, &out)
	return out, err
}

// DelegateToNexus marks a session as delegated and returns the agent instruction text.
func (r *Client) DelegateToNexus(ctx context.Context, sessionID string) (string, error) {
	var out struct {
		Instruction string `json:"instruction"`
	}
	err := r.send(ctx, call{op: "delegate to nexus", method: http.MethodPost, path: "/api/ai-sessions/" + esc(sessionID) + "/delegate", notFound: true}, &out)
	return out.Instruction, err
}

// GetDiscoveredPlanFiles returns plan/task files found near projectPath.
func (r *Client) GetDiscoveredPlanFiles(ctx context.Context, projectPath string) ([]domain.DiscoveredPlanFile, error) {
	var out []domain.DiscoveredPlanFile
	err := r.send(ctx, call{op: "get discovered plan files", method: http.MethodGet, path: "/api/plans/discovered" + query("projectPath", projectPath)}, &out)
	return out, err
}
