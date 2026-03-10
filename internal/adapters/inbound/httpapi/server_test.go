package httpapi_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
			http.Error(w, err.Error(), http.StatusNotFound)
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
	mock := &mockOrchestrator{cancelTaskErr: errors.New("not found")}
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
