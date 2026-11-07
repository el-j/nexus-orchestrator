package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
)

// TestPostTask_ErrNoPlan_Returns422 verifies that an ErrNoPlan from the orchestrator
// results in a 422 Unprocessable Entity response.
func TestPostTask_ErrNoPlan_Returns422(t *testing.T) {
	mock := &mockOrchestrator{
		submitTaskErr: fmt.Errorf("orchestrator: submit task: %w", domain.ErrNoPlan),
	}
	srv := httpapi.NewServer(mock, nil, nil)
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

func TestHandleCreateDraft_Returns201(t *testing.T) {
	mock := &mockOrchestrator{createDraftID: "draft-xyz"}
	srv := httpapi.NewServer(mock, nil, nil)
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
	srv := httpapi.NewServer(mock, nil, nil)
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
	srv := httpapi.NewServer(mock, nil, nil)
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
	srv := httpapi.NewServer(mock, nil, nil)
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

// TestHandleListTasks_ProjectPathFilter verifies GET /api/tasks?projectPath= scopes the queue.
func TestHandleListTasks_ProjectPathFilter_ReturnsMatchingTasks(t *testing.T) {
	tasks := []domain.Task{
		{ID: "q-a1", ProjectPath: "/proj/a", Status: domain.StatusQueued},
		{ID: "q-b1", ProjectPath: "/proj/b", Status: domain.StatusQueued},
		{ID: "q-a2", ProjectPath: "/proj/a", Status: domain.StatusProcessing},
	}
	mock := &mockOrchestrator{getQueueResult: tasks}
	srv := httpapi.NewServer(mock, nil, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/tasks?projectPath=/proj/a")
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
		t.Fatalf("want 2 tasks for /proj/a, got %d", len(result))
	}
	for _, task := range result {
		if task.ProjectPath != "/proj/a" {
			t.Fatalf("unexpected project in queue result: %s", task.ProjectPath)
		}
	}
}

func TestHandleGetAllTasks_ProjectPathFilter_ReturnsMatchingTasks(t *testing.T) {
	tasks := []domain.Task{
		{ID: "task-a", ProjectPath: "/repo/a", Status: domain.StatusDraft},
		{ID: "task-b", ProjectPath: "/repo/b", Status: domain.StatusCompleted},
		{ID: "task-a2", ProjectPath: "/repo/a", Status: domain.StatusFailed},
	}
	mock := &mockOrchestrator{getQueueResult: tasks}
	srv := httpapi.NewServer(mock, nil, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/tasks/all?projectPath=/repo/a")
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
		t.Fatalf("want 2 filtered tasks, got %d", len(result))
	}
	for _, task := range result {
		if task.ProjectPath != "/repo/a" {
			t.Fatalf("unexpected project path in result: %s", task.ProjectPath)
		}
	}
}

func TestHandlePromoteTask_Returns204(t *testing.T) {
	mock := &mockOrchestrator{}
	srv := httpapi.NewServer(mock, nil, nil)
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

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
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
	srv := httpapi.NewServer(mock, nil, nil)
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
