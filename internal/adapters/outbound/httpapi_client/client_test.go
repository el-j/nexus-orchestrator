package httpapi_client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

func assertContentType(t *testing.T, r *http.Request) {
	t.Helper()
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

func TestClient_SubmitAndGetTask(t *testing.T) {
	want := domain.Task{ID: "task-1", Instruction: "write hello world", Status: domain.StatusQueued}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/tasks":
			assertContentType(t, r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"task_id": want.ID}) //nolint:errcheck
		case r.Method == http.MethodGet && r.URL.Path == "/api/tasks/"+want.ID:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(want) //nolint:errcheck
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := httpapi_client.NewClient(srv.URL)
	id, err := c.SubmitTask(want)
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}
	if id != want.ID {
		t.Errorf("expected task id %q, got %q", want.ID, id)
	}
	got, err := c.GetTask(want.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.ID != want.ID || got.Instruction != want.Instruction {
		t.Errorf("unexpected task: %+v", got)
	}
}

func TestClient_AddProviderConfig(t *testing.T) {
	cfg := domain.ProviderConfig{ID: "prov-1", Name: "OpenAI", Kind: "openai"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/providers/config" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}
		assertContentType(t, r)
		var received domain.ProviderConfig
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if received.Name != cfg.Name {
			t.Errorf("expected name %q, got %q", cfg.Name, received.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(cfg) //nolint:errcheck
	}))
	defer srv.Close()
	c := httpapi_client.NewClient(srv.URL)
	created, err := c.AddProviderConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("AddProviderConfig: %v", err)
	}
	if created.ID != cfg.ID {
		t.Errorf("expected ID %q, got %q", cfg.ID, created.ID)
	}
}

func TestClient_RemoveProviderConfig(t *testing.T) {
	const id = "prov-delete-1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/providers/config/"+id {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c := httpapi_client.NewClient(srv.URL)
	if err := c.RemoveProviderConfig(context.Background(), id); err != nil {
		t.Fatalf("RemoveProviderConfig: %v", err)
	}
}

func TestClient_CreateDraft(t *testing.T) {
	task := domain.Task{Instruction: "draft task", Status: domain.StatusDraft}
	const draftID = "draft-42"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/tasks/draft" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}
		assertContentType(t, r)
		var received domain.Task
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": draftID}) //nolint:errcheck
	}))
	defer srv.Close()
	c := httpapi_client.NewClient(srv.URL)
	got, err := c.CreateDraft(task)
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if got != draftID {
		t.Errorf("expected draft id %q, got %q", draftID, got)
	}
}

func TestClient_PromoteTask(t *testing.T) {
	const taskID = "task-99"
	want := ports.PromoteResult{Promoted: true}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/tasks/"+taskID+"/promote" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()
	c := httpapi_client.NewClient(srv.URL)
	result, err := c.PromoteTask(taskID)
	if err != nil {
		t.Fatalf("PromoteTask: %v", err)
	}
	if result.Promoted != want.Promoted {
		t.Errorf("expected Promoted %v, got %v", want.Promoted, result.Promoted)
	}
}

func TestClient_RegisterAISession(t *testing.T) {
	sess := domain.AISession{ID: "sess-10", AgentName: "copilot", ProjectPath: "/project"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/ai-sessions" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}
		assertContentType(t, r)
		var received domain.AISession
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if received.AgentName != sess.AgentName {
			t.Errorf("expected agentName %q, got %q", sess.AgentName, received.AgentName)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sess) //nolint:errcheck
	}))
	defer srv.Close()
	c := httpapi_client.NewClient(srv.URL)
	created, err := c.RegisterAISession(context.Background(), sess)
	if err != nil {
		t.Fatalf("RegisterAISession: %v", err)
	}
	if created.ID != sess.ID {
		t.Errorf("expected session ID %q, got %q", sess.ID, created.ID)
	}
}

func TestClient_ClaimTask(t *testing.T) {
	const taskID = "task-claim-1"
	const sessionID = "sess-claim-1"
	want := domain.Task{ID: taskID, Status: domain.StatusProcessing}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/tasks/"+taskID+"/claim" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}
		assertContentType(t, r)
		var body struct {
			SessionID string `json:"sessionId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if body.SessionID != sessionID {
			t.Errorf("expected sessionId %q, got %q", sessionID, body.SessionID)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()
	c := httpapi_client.NewClient(srv.URL)
	got, err := c.ClaimTask(context.Background(), taskID, sessionID)
	if err != nil {
		t.Fatalf("ClaimTask: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected task ID %q, got %q", want.ID, got.ID)
	}
}

func TestClient_RequestRoutingAndContentTypeBehavior(t *testing.T) {
	type endpointCase struct {
		name          string
		method        string
		path          string
		expectsJSONCT bool
		invoke        func(c *httpapi_client.Client) error
	}

	tests := []endpointCase{
		{
			name:          "claim task sets json content type",
			method:        http.MethodPost,
			path:          "/api/tasks/task-route-1/claim",
			expectsJSONCT: true,
			invoke: func(c *httpapi_client.Client) error {
				_, err := c.ClaimTask(context.Background(), "task-route-1", "sess-route-1")
				return err
			},
		},
		{
			name:          "heartbeat task sets json content type",
			method:        http.MethodPost,
			path:          "/api/tasks/task-route-2/heartbeat",
			expectsJSONCT: true,
			invoke: func(c *httpapi_client.Client) error {
				return c.HeartbeatTask(context.Background(), "task-route-2", "sess-route-2")
			},
		},
		{
			name:          "update task status sets json content type",
			method:        http.MethodPut,
			path:          "/api/tasks/task-route-3/status",
			expectsJSONCT: true,
			invoke: func(c *httpapi_client.Client) error {
				_, err := c.UpdateTaskStatus(context.Background(), "task-route-3", "sess-route-3", domain.StatusCompleted, "done")
				return err
			},
		},
		{
			name:          "terminate session sets json content type",
			method:        http.MethodPost,
			path:          "/api/ai-sessions/sess-route-4/terminate",
			expectsJSONCT: true,
			invoke: func(c *httpapi_client.Client) error {
				return c.TerminateAISession(context.Background(), "sess-route-4", true)
			},
		},
		{
			name:          "heartbeat ai session uses no json content type",
			method:        http.MethodPost,
			path:          "/api/ai-sessions/sess-route-5/heartbeat",
			expectsJSONCT: false,
			invoke: func(c *httpapi_client.Client) error {
				return c.HeartbeatAISession(context.Background(), "sess-route-5")
			},
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method {
					http.Error(w, "wrong method", http.StatusMethodNotAllowed)
					return
				}
				if r.URL.Path != tc.path {
					http.Error(w, "wrong path", http.StatusNotFound)
					return
				}

				ct := r.Header.Get("Content-Type")
				if tc.expectsJSONCT {
					if ct != "application/json" {
						t.Errorf("expected Content-Type application/json, got %q", ct)
					}
				} else if ct != "" {
					t.Errorf("expected empty Content-Type, got %q", ct)
				}

				switch tc.path {
				case "/api/tasks/task-route-1/claim", "/api/tasks/task-route-3/status":
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(domain.Task{ID: "ok", Status: domain.StatusProcessing}) //nolint:errcheck
				case "/api/tasks/task-route-2/heartbeat", "/api/ai-sessions/sess-route-5/heartbeat":
					w.WriteHeader(http.StatusNoContent)
				case "/api/ai-sessions/sess-route-4/terminate":
					w.WriteHeader(http.StatusOK)
				default:
					http.Error(w, "unhandled test endpoint", http.StatusInternalServerError)
				}
			}))
			defer srv.Close()

			c := httpapi_client.NewClient(srv.URL)
			if err := tc.invoke(c); err != nil {
				t.Fatalf("invoke: %v", err)
			}
		})
	}
}

func TestClient_UseInjectedTransport(t *testing.T) {
	var count int
	transport := &countingTransport{calls: &count}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]domain.Task{}) //nolint:errcheck
	}))
	defer srv.Close()
	c := httpapi_client.NewClientWithHTTP(srv.URL, &http.Client{Transport: transport})
	_, err := c.GetQueue()
	if err != nil {
		t.Fatalf("GetQueue: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 request via injected transport, got %d", count)
	}
}

type countingTransport struct {
	calls *int
	base  http.RoundTripper
}

func (tr *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	*tr.calls++
	base := tr.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// TestClient_GetTask_ErrorPaths covers 404 and malformed-JSON responses.
func TestClient_GetTask_ErrorPaths(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "404 not found",
			statusCode: http.StatusNotFound,
			body:       `{"error":"not found"}`,
			wantErr:    true,
		},
		{
			name:       "malformed JSON response",
			statusCode: http.StatusOK,
			body:       `not-json{`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			c := httpapi_client.NewClient(srv.URL)
			_, err := c.GetTask("any-id")
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestClient_CancelTask_Scenarios covers the 204 success and 404 error paths.
func TestClient_CancelTask_Scenarios(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "204 no content",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "404 task not found",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			const taskID = "cancel-test-id"
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete || r.URL.Path != "/api/tasks/"+taskID {
					http.Error(w, "wrong route", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(tc.statusCode)
			}))
			defer srv.Close()

			c := httpapi_client.NewClient(srv.URL)
			err := c.CancelTask(taskID)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestClient_GetAllTasks_OK(t *testing.T) {
	want := []domain.Task{
		{ID: "t1", Instruction: "first", Status: domain.StatusQueued},
		{ID: "t2", Instruction: "second", Status: domain.StatusCompleted},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/tasks/all" {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.GetAllTasks()
	if err != nil {
		t.Fatalf("GetAllTasks: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d tasks, got %d", len(want), len(got))
	}
	if got[0].ID != want[0].ID || got[1].ID != want[1].ID {
		t.Errorf("unexpected tasks: %+v", got)
	}

	// non-200 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).GetAllTasks(); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_GetBacklog_OK(t *testing.T) {
	const projectPath = "/my/project"
	want := []domain.Task{
		{ID: "b1", Instruction: "backlog task", Status: domain.StatusDraft},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/tasks/backlog" {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		if got := r.URL.Query().Get("project"); got != projectPath {
			t.Errorf("expected query param project=%q, got %q", projectPath, got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.GetBacklog(projectPath)
	if err != nil {
		t.Fatalf("GetBacklog: %v", err)
	}
	if len(got) != 1 || got[0].ID != want[0].ID {
		t.Errorf("unexpected backlog: %+v", got)
	}

	// non-200 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).GetBacklog(""); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_UpdateTask_OK(t *testing.T) {
	const taskID = "task-upd-1"
	updates := domain.Task{Instruction: "updated instruction", Status: domain.StatusQueued}
	returned := domain.Task{ID: taskID, Instruction: "updated instruction", Status: domain.StatusQueued}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/tasks/"+taskID {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		assertContentType(t, r)
		var received domain.Task
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if received.Instruction != updates.Instruction {
			t.Errorf("expected instruction %q, got %q", updates.Instruction, received.Instruction)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(returned) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.UpdateTask(taskID, updates)
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if got.ID != returned.ID {
		t.Errorf("expected ID %q, got %q", returned.ID, got.ID)
	}

	// non-200 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).UpdateTask("x", domain.Task{}); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_ListAISessions_OK(t *testing.T) {
	want := []domain.AISession{
		{ID: "s1", AgentName: "claude", Status: domain.SessionStatusActive},
		{ID: "s2", AgentName: "cursor", Status: domain.SessionStatusIdle},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/ai-sessions" {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.ListAISessions(context.Background())
	if err != nil {
		t.Fatalf("ListAISessions: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d sessions, got %d", len(want), len(got))
	}
	if got[0].ID != want[0].ID || got[1].ID != want[1].ID {
		t.Errorf("unexpected sessions: %+v", got)
	}

	// non-200 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).ListAISessions(context.Background()); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_DeregisterAISession_OK(t *testing.T) {
	const sessID = "sess-dereg-1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/ai-sessions/"+sessID {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	if err := c.DeregisterAISession(context.Background(), sessID); err != nil {
		t.Fatalf("DeregisterAISession: %v", err)
	}

	// non-204 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv2.Close()
	if err := httpapi_client.NewClient(srv2.URL).DeregisterAISession(context.Background(), sessID); err == nil {
		t.Error("expected error on non-204, got nil")
	}
}

func TestClient_PurgeDisconnectedSessions_OK(t *testing.T) {
	const deletedCount = 3
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/ai-sessions" {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"deleted": deletedCount}) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	n, err := c.PurgeDisconnectedSessions(context.Background())
	if err != nil {
		t.Fatalf("PurgeDisconnectedSessions: %v", err)
	}
	if n != deletedCount {
		t.Errorf("expected deleted=%d, got %d", deletedCount, n)
	}

	// malformed JSON returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).PurgeDisconnectedSessions(context.Background()); err == nil {
		t.Error("expected error on malformed JSON, got nil")
	}
}

func TestClient_GetDiscoveredProviders_OK(t *testing.T) {
	want := []domain.DiscoveredProvider{
		{ID: "dp-1", Name: "Ollama", Kind: domain.ProviderKindOllama},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/providers/discovered" {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.GetDiscoveredProviders()
	if err != nil {
		t.Fatalf("GetDiscoveredProviders: %v", err)
	}
	if len(got) != 1 || got[0].ID != want[0].ID {
		t.Errorf("unexpected providers: %+v", got)
	}

	// non-200 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).GetDiscoveredProviders(); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_TriggerScan_OK(t *testing.T) {
	want := []domain.DiscoveredProvider{
		{ID: "dp-scan-1", Name: "LM Studio", Kind: domain.ProviderKindLMStudio},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/providers/discovered/scan" {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.TriggerScan(context.Background())
	if err != nil {
		t.Fatalf("TriggerScan: %v", err)
	}
	if len(got) != 1 || got[0].ID != want[0].ID {
		t.Errorf("unexpected providers from scan: %+v", got)
	}

	// non-200/204 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).TriggerScan(context.Background()); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_GetRuntimeConfig_OK(t *testing.T) {
	want := domain.RuntimeConfig{QueueCap: 50}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/config" {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"queueCap": want.QueueCap}) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.GetRuntimeConfig(context.Background())
	if err != nil {
		t.Fatalf("GetRuntimeConfig: %v", err)
	}
	if got.QueueCap != want.QueueCap {
		t.Errorf("expected QueueCap %d, got %d", want.QueueCap, got.QueueCap)
	}

	// non-200 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).GetRuntimeConfig(context.Background()); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_UpdateRuntimeConfig_OK(t *testing.T) {
	queueCap := 25
	update := domain.RuntimeConfigUpdate{QueueCap: &queueCap}
	wantResp := domain.RuntimeConfig{QueueCap: 25, APIToken: "tok-abc"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/config" {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		assertContentType(t, r)
		var received domain.RuntimeConfigUpdate
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if received.QueueCap == nil || *received.QueueCap != queueCap {
			t.Errorf("expected queueCap %d in body", queueCap)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
			"queueCap": wantResp.QueueCap,
			"apiToken": wantResp.APIToken,
		})
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.UpdateRuntimeConfig(context.Background(), update)
	if err != nil {
		t.Fatalf("UpdateRuntimeConfig: %v", err)
	}
	if got.QueueCap != wantResp.QueueCap {
		t.Errorf("expected QueueCap %d, got %d", wantResp.QueueCap, got.QueueCap)
	}
	if got.APIToken != wantResp.APIToken {
		t.Errorf("expected APIToken %q, got %q", wantResp.APIToken, got.APIToken)
	}

	// non-200 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).UpdateRuntimeConfig(context.Background(), update); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}
