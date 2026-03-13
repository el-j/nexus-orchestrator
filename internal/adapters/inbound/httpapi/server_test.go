package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// mockOrchestrator implements ports.Orchestrator with configurable responses.
// All fields are set at construction time and only read during handler execution,
// so no additional locking is required for these tests.
type mockOrchestrator struct {
	submitTaskID  string
	submitTaskErr error

	getTaskResult domain.Task
	getTaskErr    error

	getQueueResult []domain.Task
	getQueueErr    error

	getProvidersResult []ports.ProviderInfo
	getProvidersErr    error

	cancelTaskErr error

	registerProviderErr error

	removeProviderErr error

	getProviderModelsResult []string
	getProviderModelsErr    error

	createDraftID  string
	createDraftErr error

	getBacklogResult []domain.Task
	getBacklogErr    error

	promoteTaskErr error

	updateTaskResult domain.Task
	updateTaskErr    error

	discoveredProvidersResult []domain.DiscoveredProvider
	promoteProviderErr        error
}

func (m *mockOrchestrator) SubmitTask(_ domain.Task) (string, error) {
	return m.submitTaskID, m.submitTaskErr
}

func (m *mockOrchestrator) GetTask(_ string) (domain.Task, error) {
	return m.getTaskResult, m.getTaskErr
}

func (m *mockOrchestrator) GetQueue() ([]domain.Task, error) {
	return m.getQueueResult, m.getQueueErr
}

func (m *mockOrchestrator) GetProviders() ([]ports.ProviderInfo, error) {
	return m.getProvidersResult, m.getProvidersErr
}

func (m *mockOrchestrator) CancelTask(_ string) error {
	return m.cancelTaskErr
}

func (m *mockOrchestrator) RegisterCloudProvider(_ domain.ProviderConfig) error {
	return m.registerProviderErr
}

func (m *mockOrchestrator) RemoveProvider(_ string) error {
	return m.removeProviderErr
}

func (m *mockOrchestrator) GetProviderModels(_ string) ([]string, error) {
	return m.getProviderModelsResult, m.getProviderModelsErr
}

func (m *mockOrchestrator) AddProviderConfig(_ context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	return cfg, nil
}

func (m *mockOrchestrator) UpdateProviderConfig(_ context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	return cfg, nil
}

func (m *mockOrchestrator) RemoveProviderConfig(_ context.Context, _ string) error {
	return nil
}

func (m *mockOrchestrator) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	return nil, nil
}
func (m *mockOrchestrator) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	return m.discoveredProvidersResult, nil
}
func (m *mockOrchestrator) TriggerScan(_ context.Context) ([]domain.DiscoveredProvider, error) {
	return nil, nil
}
func (m *mockOrchestrator) PromoteProvider(_ context.Context, _ string) error {
	return m.promoteProviderErr
}

// configurable fields for backlog operations
var _ ports.Orchestrator = (*mockOrchestrator)(nil) // compile-time interface check

func (m *mockOrchestrator) CreateDraft(_ domain.Task) (string, error) {
	return m.createDraftID, m.createDraftErr
}
func (m *mockOrchestrator) GetBacklog(_ string) ([]domain.Task, error) {
	return m.getBacklogResult, m.getBacklogErr
}
func (m *mockOrchestrator) PromoteTask(_ string) error {
	return m.promoteTaskErr
}
func (m *mockOrchestrator) UpdateTask(_ string, _ domain.Task) (domain.Task, error) {
	return m.updateTaskResult, m.updateTaskErr
}
func (m *mockOrchestrator) RegisterAISession(_ context.Context, s domain.AISession) (domain.AISession, error) {
	return s, nil
}
func (m *mockOrchestrator) ListAISessions(_ context.Context) ([]domain.AISession, error) {
	return nil, nil
}
func (m *mockOrchestrator) DeregisterAISession(_ context.Context, _ string) error {
	return nil
}
func (m *mockOrchestrator) HeartbeatAISession(_ context.Context, _ string) error {
	return nil
}
func (m *mockOrchestrator) ClaimTask(_ context.Context, _ string, _ string) (domain.Task, error) {
	return domain.Task{}, nil
}
func (m *mockOrchestrator) UpdateTaskStatus(_ context.Context, _ string, _ string, _ domain.TaskStatus, _ string) (domain.Task, error) {
	return domain.Task{}, nil
}
func (m *mockOrchestrator) GetAllTasks() ([]domain.Task, error) {
	return m.getQueueResult, m.getQueueErr
}

// newTestHandler builds a chi router with the same route/handler logic as StartServer.
func newTestHandler(orch ports.Orchestrator) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Post("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		var req domain.Task
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		taskID, err := orch.SubmitTask(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"task_id": taskID, "status": string(domain.StatusQueued)})
	})

	r.Get("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		tasks, err := orch.GetQueue()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if tasks == nil {
			tasks = []domain.Task{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tasks)
	})

	r.Get("/api/tasks/all", func(w http.ResponseWriter, r *http.Request) {
		tasks, err := orch.GetAllTasks()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if tasks == nil {
			tasks = []domain.Task{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tasks)
	})

	r.Get("/api/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		task, err := orch.GetTask(id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				http.Error(w, "task not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(task)
	})

	r.Delete("/api/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := orch.CancelTask(id); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				http.Error(w, "task not found", http.StatusNotFound)
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	r.Get("/api/providers", func(w http.ResponseWriter, r *http.Request) {
		providers, err := orch.GetProviders()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if providers == nil {
			providers = []ports.ProviderInfo{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(providers)
	})

	r.Post("/api/providers", func(w http.ResponseWriter, r *http.Request) {
		var cfg domain.ProviderConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if cfg.Name == "" || cfg.Kind == "" {
			http.Error(w, "name and kind are required", http.StatusBadRequest)
			return
		}
		if err := orch.RegisterCloudProvider(cfg); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"name": cfg.Name, "kind": string(cfg.Kind)})
	})

	r.Delete("/api/providers/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if err := orch.RemoveProvider(name); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	r.Get("/api/providers/{name}/models", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		models, err := orch.GetProviderModels(name)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if models == nil {
			models = []string{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(models)
	})

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "nexus-orchestrator"})
	})

	return r
}

// TestPostTask_Success verifies that a valid POST /api/tasks returns 201 with task_id and status.
func TestPostTask_Success(t *testing.T) {
	mock := &mockOrchestrator{submitTaskID: "task-abc-123"}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	body := bytes.NewBufferString(`{"Instruction":"write a hello world"}`)
	resp, err := http.Post(ts.URL+"/api/tasks", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result["task_id"] != "task-abc-123" {
		t.Errorf("expected task_id %q, got %q", "task-abc-123", result["task_id"])
	}
	if result["status"] != string(domain.StatusQueued) {
		t.Errorf("expected status %q, got %q", domain.StatusQueued, result["status"])
	}
}

// TestPostTask_InvalidJSON verifies that malformed JSON returns 400.
func TestPostTask_InvalidJSON(t *testing.T) {
	mock := &mockOrchestrator{}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	body := bytes.NewBufferString(`not-valid-json`)
	resp, err := http.Post(ts.URL+"/api/tasks", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// TestPostTask_SubmitTaskError verifies that an orchestrator error returns 500.
func TestPostTask_SubmitTaskError(t *testing.T) {
	mock := &mockOrchestrator{submitTaskErr: errors.New("internal failure")}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	body := bytes.NewBufferString(`{"Instruction":"do something"}`)
	resp, err := http.Post(ts.URL+"/api/tasks", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestGetTasks_Success verifies GET /api/tasks returns 200 with a JSON array of tasks.
func TestGetTasks_Success(t *testing.T) {
	tasks := []domain.Task{
		{ID: "t1", Instruction: "task one", Status: domain.StatusQueued},
		{ID: "t2", Instruction: "task two", Status: domain.StatusProcessing},
	}
	mock := &mockOrchestrator{getQueueResult: tasks}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/tasks")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var result []domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(result))
	}
	if result[0].ID != "t1" {
		t.Errorf("expected first task ID %q, got %q", "t1", result[0].ID)
	}
	if result[1].ID != "t2" {
		t.Errorf("expected second task ID %q, got %q", "t2", result[1].ID)
	}
}

// TestGetTasks_EmptyQueue verifies GET /api/tasks returns 200 with an empty JSON array (not null).
func TestGetTasks_EmptyQueue(t *testing.T) {
	mock := &mockOrchestrator{getQueueResult: nil}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/tasks")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	// Must be a JSON array, not null.
	var result []domain.Task
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("body is not valid JSON array: %s", string(raw))
	}
	if result == nil {
		t.Error("expected non-nil empty slice in JSON response, got null")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(result))
	}
}

func TestGetAllTasks_Success(t *testing.T) {
	tasks := []domain.Task{
		{ID: "d1", Status: domain.StatusDraft},
		{ID: "c1", Status: domain.StatusCompleted},
	}
	mock := &mockOrchestrator{getQueueResult: tasks}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/tasks/all")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result []domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(result))
	}
	if result[0].ID != "d1" || result[1].ID != "c1" {
		t.Errorf("unexpected task order/result: %+v", result)
	}
}

// TestGetTask_Found verifies GET /api/tasks/{id} returns 200 with the task JSON when found.
func TestGetTask_Found(t *testing.T) {
	task := domain.Task{ID: "task-42", Instruction: "build something", Status: domain.StatusCompleted}
	mock := &mockOrchestrator{getTaskResult: task}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/tasks/task-42")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var result domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.ID != "task-42" {
		t.Errorf("expected task ID %q, got %q", "task-42", result.ID)
	}
	if result.Status != domain.StatusCompleted {
		t.Errorf("expected status %q, got %q", domain.StatusCompleted, result.Status)
	}
}

// TestGetTask_NotFound verifies GET /api/tasks/{id} returns 404 when domain.ErrNotFound is returned.
func TestGetTask_NotFound(t *testing.T) {
	mock := &mockOrchestrator{getTaskErr: domain.ErrNotFound}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/tasks/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// TestDeleteTask_Success verifies DELETE /api/tasks/{id} returns 204 on success.
func TestDeleteTask_Success(t *testing.T) {
	mock := &mockOrchestrator{cancelTaskErr: nil}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/tasks/task-99", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

// TestDeleteTask_NotFound verifies DELETE /api/tasks/{id} returns 404 when CancelTask errors.
func TestDeleteTask_NotFound(t *testing.T) {
	mock := &mockOrchestrator{cancelTaskErr: fmt.Errorf("orchestrator: cancel task: %w", domain.ErrNotFound)}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/tasks/missing", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// TestGetProviders_Success verifies GET /api/providers returns 200 with a JSON array.
func TestGetProviders_Success(t *testing.T) {
	providers := []ports.ProviderInfo{
		{Name: "lmstudio", Active: true},
		{Name: "ollama", Active: false},
	}
	mock := &mockOrchestrator{getProvidersResult: providers}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/providers")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var result []ports.ProviderInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(result))
	}
	if result[0].Name != "lmstudio" || !result[0].Active {
		t.Errorf("unexpected first provider: %+v", result[0])
	}
	if result[1].Name != "ollama" || result[1].Active {
		t.Errorf("unexpected second provider: %+v", result[1])
	}
}

// TestGetHealth verifies GET /api/health returns 200 with the expected JSON body.
func TestGetHealth(t *testing.T) {
	mock := &mockOrchestrator{}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status %q, got %q", "ok", result["status"])
	}
	if result["service"] != "nexus-orchestrator" {
		t.Errorf("expected service %q, got %q", "nexus-orchestrator", result["service"])
	}
}

// --- POST /api/providers tests -----------------------------------------------

func TestPostProvider_Success(t *testing.T) {
	mock := &mockOrchestrator{}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	body, _ := json.Marshal(domain.ProviderConfig{Name: "my-ollama", Kind: domain.ProviderKindOllama})
	resp, err := http.Post(ts.URL+"/api/providers", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result["name"] != "my-ollama" {
		t.Errorf("expected name %q, got %q", "my-ollama", result["name"])
	}
}

func TestPostProvider_MissingFields(t *testing.T) {
	mock := &mockOrchestrator{}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	body, _ := json.Marshal(domain.ProviderConfig{Name: "no-kind"}) // Kind is empty → 400
	resp, err := http.Post(ts.URL+"/api/providers", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPostProvider_FactoryError(t *testing.T) {
	mock := &mockOrchestrator{registerProviderErr: errors.New("unsupported kind")}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	body, _ := json.Marshal(domain.ProviderConfig{Name: "bad", Kind: "unknown"})
	resp, err := http.Post(ts.URL+"/api/providers", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

// --- DELETE /api/providers/{name} tests --------------------------------------

func TestDeleteProvider_Success(t *testing.T) {
	mock := &mockOrchestrator{}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/providers/my-provider", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

func TestDeleteProvider_NotFound(t *testing.T) {
	mock := &mockOrchestrator{removeProviderErr: domain.ErrNotFound}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/providers/ghost", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// --- GET /api/providers/{name}/models tests ----------------------------------

func TestGetProviderModels_Success(t *testing.T) {
	mock := &mockOrchestrator{getProviderModelsResult: []string{"llama3", "codellama"}}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/providers/my-ollama/models")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var models []string
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0] != "llama3" {
		t.Errorf("unexpected models: %v", models)
	}
}

func TestGetProviderModels_NotFound(t *testing.T) {
	mock := &mockOrchestrator{getProviderModelsErr: domain.ErrNotFound}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/providers/ghost/models")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// --- Cancel 404 vs 500 tests (production handler) ---------------------------

func TestDeleteTask_InternalError_Returns500(t *testing.T) {
	// Non-ErrNotFound errors should return 500, not 404.
	mock := &mockOrchestrator{cancelTaskErr: errors.New("database locked")}
	ts := httptest.NewServer(newTestHandler(mock))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/tasks/some-task", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 for non-ErrNotFound cancel error, got %d", resp.StatusCode)
	}
}

// --- Security headers (production handler via httpapi.Server) -----------------

func TestSecurityHeaders_OnAPIResponse(t *testing.T) {
	mock := &mockOrchestrator{}
	srv := httpapi.NewServer(mock, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("expected X-Frame-Options DENY, got %q", got)
	}
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("expected X-Content-Type-Options nosniff, got %q", got)
	}
}

func TestDashboard_CSPHeader(t *testing.T) {
	mock := &mockOrchestrator{}
	srv := httpapi.NewServer(mock, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/ui")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	csp := resp.Header.Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected Content-Security-Policy header on /ui, got empty")
	}
	if got := resp.Header.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("expected X-Frame-Options DENY on /ui, got %q", got)
	}
}

// TestPostTask_ErrNoPlan_Returns422 verifies that an ErrNoPlan from the orchestrator
// results in a 422 Unprocessable Entity response.
func TestPostTask_ErrNoPlan_Returns422(t *testing.T) {
	mock := &mockOrchestrator{
		submitTaskErr: fmt.Errorf("orchestrator: submit task: %w", domain.ErrNoPlan),
	}
	srv := httpapi.NewServer(mock, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := bytes.NewBufferString(`{"instruction":"execute now","command":"execute","projectPath":"/proj/x"}`)
	resp, err := http.Post(ts.URL+"/api/tasks", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	if got := string(b); !strings.Contains(got, "planning required") {
		t.Errorf("expected 'planning required' in body, got: %s", got)
	}
}

// --- Backlog lifecycle HTTP tests --------------------------------------------

func TestHandleCreateDraft_Returns201(t *testing.T) {
	mock := &mockOrchestrator{createDraftID: "draft-xyz"}
	srv := httpapi.NewServer(mock, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := bytes.NewBufferString(`{"projectPath":"/proj/a","instruction":"build it"}`)
	resp, err := http.Post(ts.URL+"/api/tasks/draft", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("want 201, got %d", resp.StatusCode)
	}
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result["id"] != "draft-xyz" {
		t.Errorf("id: want draft-xyz, got %q", result["id"])
	}
	if result["status"] != "DRAFT" {
		t.Errorf("status: want DRAFT, got %q", result["status"])
	}
}

func TestHandleGetBacklog_Returns200WithTasks(t *testing.T) {
	tasks := []domain.Task{
		{ID: "d1", Status: domain.StatusDraft},
		{ID: "b1", Status: domain.StatusBacklog},
	}
	mock := &mockOrchestrator{getBacklogResult: tasks}
	srv := httpapi.NewServer(mock, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/tasks/backlog?project=/some/path")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
	var result []domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("want 2 tasks, got %d", len(result))
	}
}

func TestHandleGetBacklog_WithoutProjectParam_Returns200(t *testing.T) {
	mock := &mockOrchestrator{getBacklogResult: []domain.Task{{ID: "draft-any", Status: domain.StatusDraft}}}
	srv := httpapi.NewServer(mock, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/tasks/backlog")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
	var result []domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].ID != "draft-any" {
		t.Errorf("unexpected backlog result: %+v", result)
	}
}

func TestHandleGetAllTasks_Returns200WithTasks(t *testing.T) {
	tasks := []domain.Task{
		{ID: "draft-1", Status: domain.StatusDraft},
		{ID: "done-1", Status: domain.StatusCompleted},
	}
	mock := &mockOrchestrator{getQueueResult: tasks}
	srv := httpapi.NewServer(mock, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/tasks/all")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
	var result []domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("want 2 tasks, got %d", len(result))
	}
}

func TestHandlePromoteTask_Returns204(t *testing.T) {
	mock := &mockOrchestrator{}
	srv := httpapi.NewServer(mock, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/tasks/some-id/promote", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("want 204, got %d", resp.StatusCode)
	}
}

func TestHandleUpdateTask_Returns200(t *testing.T) {
	updated := domain.Task{
		ID:          "task-99",
		Instruction: "new instruction",
		Priority:    1,
		Status:      domain.StatusDraft,
	}
	mock := &mockOrchestrator{updateTaskResult: updated}
	srv := httpapi.NewServer(mock, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := bytes.NewBufferString(`{"instruction":"new instruction","priority":1}`)
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/tasks/task-99", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
	var result domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.ID != "task-99" {
		t.Errorf("id: want task-99, got %q", result.ID)
	}
	if result.Instruction != "new instruction" {
		t.Errorf("instruction: want %q, got %q", "new instruction", result.Instruction)
	}
}

// --- Provider discovery HTTP tests ------------------------------------------

// TestHTTP_GetDiscoveredProviders_Returns200 verifies that GET /api/providers/discovered
// returns 200 with application/json content type.
func TestHTTP_GetDiscoveredProviders_Returns200(t *testing.T) {
	mock := &mockOrchestrator{
		discoveredProvidersResult: []domain.DiscoveredProvider{},
	}
	srv := httpapi.NewServer(mock, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/providers/discovered")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}

// TestHTTP_TriggerScan_Returns200 verifies that POST /api/providers/discovered/scan
// returns 200 when the scan succeeds.
func TestHTTP_TriggerScan_Returns200(t *testing.T) {
	mock := &mockOrchestrator{}
	srv := httpapi.NewServer(mock, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/providers/discovered/scan", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestHTTP_PromoteProvider_NotFound_Returns404 verifies that
// POST /api/providers/promote/{id} returns 404 when the orchestrator
// returns domain.ErrNotFound for the given provider ID.
func TestHTTP_PromoteProvider_NotFound_Returns404(t *testing.T) {
	mock := &mockOrchestrator{promoteProviderErr: domain.ErrNotFound}
	srv := httpapi.NewServer(mock, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/providers/promote/bad-id", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// TestErrorResponse_HasJSONContentType verifies that error responses from the
// production handler set Content-Type: application/json and return a valid
// {"error":"..."} JSON body rather than plain text.
func TestErrorResponse_HasJSONContentType(t *testing.T) {
	mock := &mockOrchestrator{}
	srv := httpapi.NewServer(mock, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Submitting malformed JSON triggers the decode-error path in handleCreateTask.
	body := bytes.NewBufferString(`not-valid-json`)
	resp, err := http.Post(ts.URL+"/api/tasks", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json for error response, got %q", ct)
	}
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("error response body is not valid JSON: %v", err)
	}
	if _, ok := result["error"]; !ok {
		t.Error("expected 'error' key in JSON error response body")
	}
}
