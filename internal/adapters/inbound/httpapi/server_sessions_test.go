package httpapi_test

import (
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

func TestHandleTerminateAISession(t *testing.T) {
	tests := []struct {
		name          string
		sessionID     string
		body          string
		contentLength int64
		terminateErr  error
		wantStatus    int
	}{
		{
			name:          "success no body",
			sessionID:     "sess-1",
			body:          "",
			contentLength: 0,
			wantStatus:    http.StatusNoContent,
		},
		{
			name:          "success force true",
			sessionID:     "sess-1",
			body:          `{"force":true}`,
			contentLength: -1,
			wantStatus:    http.StatusNoContent,
		},
		{
			name:          "not found",
			sessionID:     "missing",
			body:          `{}`,
			contentLength: -1,
			terminateErr:  domain.ErrNotFound,
			wantStatus:    http.StatusNotFound,
		},
		{
			name:          "bad json body",
			sessionID:     "sess-1",
			body:          `{bad}`,
			contentLength: -1,
			wantStatus:    http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockOrchestrator{terminateAISessionErr: tc.terminateErr}
			srv := httpapi.NewServer(mock, nil, nil)
			ts := httptest.NewServer(srv.Handler())
			defer ts.Close()

			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = strings.NewReader(tc.body)
			} else {
				bodyReader = http.NoBody
			}
			req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/ai-sessions/"+tc.sessionID+"/terminate", bodyReader)
			if err != nil {
				t.Fatal(err)
			}
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
				if tc.contentLength >= 0 {
					req.ContentLength = tc.contentLength
				}
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestHandleHeartbeatAISession(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		hbErr      error
		wantStatus int
	}{
		{"success", "sess-1", nil, http.StatusNoContent},
		{"not found", "missing", domain.ErrNotFound, http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockOrchestrator{heartbeatAISessionErr: tc.hbErr}
			srv := httpapi.NewServer(mock, nil, nil)
			ts := httptest.NewServer(srv.Handler())
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/ai-sessions/"+tc.sessionID+"/heartbeat", http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestHandleHeartbeatTask(t *testing.T) {
	tests := []struct {
		name       string
		taskID     string
		body       string
		hbErr      error
		wantStatus int
	}{
		{"success", "task-1", `{"session_id":"sess-1"}`, nil, http.StatusNoContent},
		{"missing session_id", "task-1", `{}`, nil, http.StatusBadRequest},
		{"not found", "task-1", `{"session_id":"sess-x"}`, domain.ErrNotFound, http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockOrchestrator{heartbeatTaskErr: tc.hbErr}
			srv := httpapi.NewServer(mock, nil, nil)
			ts := httptest.NewServer(srv.Handler())
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/tasks/"+tc.taskID+"/heartbeat", strings.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestHandleClaimTask(t *testing.T) {
	claimedTask := domain.Task{ID: "task-1", Status: domain.StatusProcessing, AISessionID: "sess-1"}

	tests := []struct {
		name        string
		taskID      string
		body        string
		claimResult domain.Task
		claimErr    error
		wantStatus  int
	}{
		{"success", "task-1", `{"sessionId":"sess-1"}`, claimedTask, nil, http.StatusOK},
		{"missing sessionId", "task-1", `{}`, domain.Task{}, nil, http.StatusBadRequest},
		{"not found", "task-1", `{"sessionId":"sess-x"}`, domain.Task{}, domain.ErrNotFound, http.StatusNotFound},
		{
			"project mismatch",
			"task-1",
			`{"sessionId":"sess-other-proj"}`,
			domain.Task{},
			fmt.Errorf("orchestrator: claim task: session project %q does not match task project %q", "/proj/b", "/proj/a"),
			http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockOrchestrator{claimTaskResult: tc.claimResult, claimTaskErr: tc.claimErr}
			srv := httpapi.NewServer(mock, nil, nil)
			ts := httptest.NewServer(srv.Handler())
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/tasks/"+tc.taskID+"/claim", strings.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, resp.StatusCode)
			}
			if tc.wantStatus == http.StatusOK {
				var result domain.Task
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatal(err)
				}
				if result.ID != tc.claimResult.ID {
					t.Errorf("task ID: want %q, got %q", tc.claimResult.ID, result.ID)
				}
			}
		})
	}
}

func TestHandleUpdateTaskStatus(t *testing.T) {
	completedTask := domain.Task{ID: "task-1", Status: domain.StatusCompleted}

	tests := []struct {
		name         string
		taskID       string
		body         string
		updateResult domain.Task
		updateErr    error
		wantStatus   int
	}{
		{
			name:         "success completed",
			taskID:       "task-1",
			body:         `{"sessionId":"sess-1","status":"COMPLETED"}`,
			updateResult: completedTask,
			wantStatus:   http.StatusOK,
		},
		{
			name:       "invalid JSON",
			taskID:     "task-1",
			body:       `{bad json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing status",
			taskID:     "task-1",
			body:       `{"sessionId":"sess-1"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found",
			taskID:     "task-1",
			body:       `{"sessionId":"sess-1","status":"COMPLETED"}`,
			updateErr:  domain.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockOrchestrator{updateTaskStatusResult: tc.updateResult, updateTaskStatusErr: tc.updateErr}
			srv := httpapi.NewServer(mock, nil, nil)
			ts := httptest.NewServer(srv.Handler())
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/tasks/"+tc.taskID+"/status", strings.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, resp.StatusCode)
			}
			if tc.wantStatus == http.StatusOK {
				var result domain.Task
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatal(err)
				}
				if result.ID != tc.updateResult.ID {
					t.Errorf("task ID: want %q, got %q", tc.updateResult.ID, result.ID)
				}
			}
		})
	}
}

func TestHandleDelegateToNexus(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		instruction string
		delegateErr error
		wantStatus  int
	}{
		{"success", "sess-1", "do something useful", nil, http.StatusOK},
		{"not found", "missing", "", domain.ErrNotFound, http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockOrchestrator{delegateInstruction: tc.instruction, delegateErr: tc.delegateErr}
			srv := httpapi.NewServer(mock, nil, nil)
			ts := httptest.NewServer(srv.Handler())
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/ai-sessions/"+tc.sessionID+"/delegate", http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, resp.StatusCode)
			}
			if tc.wantStatus == http.StatusOK {
				var result map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatal(err)
				}
				if result["instruction"] != tc.instruction {
					t.Errorf("instruction: want %q, got %q", tc.instruction, result["instruction"])
				}
				if result["sessionId"] != tc.sessionID {
					t.Errorf("sessionId: want %q, got %q", tc.sessionID, result["sessionId"])
				}
			}
		})
	}
}

func TestHandleGetSessionTasks(t *testing.T) {
	t.Run("returns tasks for session", func(t *testing.T) {
		tasks := []domain.Task{
			{ID: "t1", AISessionID: "sess-1", Status: domain.StatusProcessing},
			{ID: "t2", AISessionID: "sess-1", Status: domain.StatusCompleted},
			{ID: "t3", AISessionID: "sess-other", Status: domain.StatusQueued},
		}
		mock := &mockOrchestrator{getQueueResult: tasks}
		srv := httpapi.NewServer(mock, nil, nil)
		ts := httptest.NewServer(srv.Handler())
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/ai-sessions/sess-1/tasks")
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
			t.Fatalf("want 2 tasks for sess-1, got %d", len(result))
		}
		for _, task := range result {
			if task.AISessionID != "sess-1" {
				t.Errorf("unexpected task with AISessionID %q", task.AISessionID)
			}
		}
	})

	t.Run("returns empty list when no tasks", func(t *testing.T) {
		mock := &mockOrchestrator{getQueueResult: nil}
		srv := httpapi.NewServer(mock, nil, nil)
		ts := httptest.NewServer(srv.Handler())
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/ai-sessions/sess-empty/tasks")
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
		if len(result) != 0 {
			t.Errorf("want 0 tasks, got %d", len(result))
		}
	})
}

func TestHandlePurgeDisconnectedSessions(t *testing.T) {
	tests := []struct {
		name       string
		purgeCount int
		purgeErr   error
		wantStatus int
		wantCount  int
	}{
		{"success with deletions", 3, nil, http.StatusOK, 3},
		{"success zero deletions", 0, nil, http.StatusOK, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockOrchestrator{purgeCount: tc.purgeCount, purgeErr: tc.purgeErr}
			srv := httpapi.NewServer(mock, nil, nil)
			ts := httptest.NewServer(srv.Handler())
			defer ts.Close()

			req, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/ai-sessions", http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("want %d, got %d", tc.wantStatus, resp.StatusCode)
			}
			var result map[string]int
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatal(err)
			}
			if result["deleted"] != tc.wantCount {
				t.Errorf("deleted: want %d, got %d", tc.wantCount, result["deleted"])
			}
		})
	}
}
