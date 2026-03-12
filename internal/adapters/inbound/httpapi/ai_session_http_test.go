package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/adapters/outbound/fs_writer"
	"nexus-orchestrator/internal/adapters/outbound/repo_sqlite"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

// aiSessionTestLLM is a minimal LLM stub needed to construct DiscoveryService.
// AI session tests do not submit tasks, so no LLM method is actually invoked.
type aiSessionTestLLM struct{}

func (m *aiSessionTestLLM) Ping() bool                              { return true }
func (m *aiSessionTestLLM) ProviderName() string                    { return "test" }
func (m *aiSessionTestLLM) ActiveModel() string                     { return "test-model" }
func (m *aiSessionTestLLM) BaseURL() string                         { return "" }
func (m *aiSessionTestLLM) GetAvailableModels() ([]string, error)   { return []string{"test-model"}, nil }
func (m *aiSessionTestLLM) ContextLimit() int                       { return 0 }
func (m *aiSessionTestLLM) GenerateCode(_ string) (string, error)   { return "", nil }
func (m *aiSessionTestLLM) Chat(_ []domain.Message) (string, error) { return "", nil }

// newAISessionStack wires a real OrchestratorService with in-memory SQLite and
// returns an httptest.Server exposing the full httpapi.Server handler tree.
// The returned cleanup function stops the orchestrator, closes the DB, and
// shuts down the httptest.Server.
func newAISessionStack(t *testing.T) (*httptest.Server, func()) {
	t.Helper()

	repo, err := repo_sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("open :memory: db: %v", err)
	}

	sessionRepo := repo_sqlite.NewSessionRepo(repo)
	aiSessionRepo := repo_sqlite.NewAISessionRepo(repo)
	writer := fs_writer.New()
	discovery := services.NewDiscoveryService(&aiSessionTestLLM{})
	orch := services.NewOrchestrator(discovery, repo, writer, sessionRepo)
	orch.SetAISessionRepo(aiSessionRepo)

	hub := httpapi.NewHub()
	orch.SetBroadcaster(hub)

	srv := httptest.NewServer(httpapi.NewServer(orch, hub).Handler())
	return srv, func() {
		srv.Close()
		orch.Stop()
		_ = repo.Close()
	}
}

func TestAISessionHTTP(t *testing.T) {
	t.Parallel()

	// Validation cases — each spins up its own server and runs in parallel.
	validationCases := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:       "missing_agentName_returns_400",
			body:       map[string]interface{}{"source": "mcp"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid_source_returns_400",
			body:       map[string]interface{}{"agentName": "Claude Desktop", "source": "invalid"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range validationCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv, cleanup := newAISessionStack(t)
			defer cleanup()

			body, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("marshal request body: %v", err)
			}
			resp, err := http.Post(srv.URL+"/api/ai-sessions", "application/json", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("POST /api/ai-sessions: %v", err)
			}
			resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, resp.StatusCode)
			}
		})
	}

	// Lifecycle — steps must be sequential and share a single server instance.
	t.Run("lifecycle", func(t *testing.T) {
		srv, cleanup := newAISessionStack(t)
		defer cleanup()

		// Step 3: POST valid session → 201 + JSON body with "id" field.
		reqBody, _ := json.Marshal(map[string]interface{}{
			"agentName": "Claude Desktop",
			"source":    "mcp",
		})
		resp, err := http.Post(srv.URL+"/api/ai-sessions", "application/json", bytes.NewReader(reqBody))
		if err != nil {
			t.Fatalf("POST /api/ai-sessions: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			resp.Body.Close()
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
		var created domain.AISession
		if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
			resp.Body.Close()
			t.Fatalf("decode create response: %v", err)
		}
		resp.Body.Close()
		if created.ID == "" {
			t.Fatal("expected non-empty id in 201 response")
		}

		// Step 4: GET /api/ai-sessions → array containing exactly 1 session.
		resp, err = http.Get(srv.URL + "/api/ai-sessions")
		if err != nil {
			t.Fatalf("GET /api/ai-sessions: %v", err)
		}
		var sessions []domain.AISession
		if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
			resp.Body.Close()
			t.Fatalf("decode sessions list: %v", err)
		}
		resp.Body.Close()
		if len(sessions) != 1 {
			t.Fatalf("expected 1 session after register, got %d", len(sessions))
		}

		// Step 5: DELETE /api/ai-sessions/{id} with valid id → 204.
		req, err := http.NewRequestWithContext(context.Background(),
			http.MethodDelete, srv.URL+"/api/ai-sessions/"+created.ID, nil)
		if err != nil {
			t.Fatalf("build DELETE request: %v", err)
		}
		deleteResp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("DELETE /api/ai-sessions/%s: %v", created.ID, err)
		}
		deleteResp.Body.Close()
		if deleteResp.StatusCode != http.StatusNoContent {
			t.Errorf("expected 204, got %d", deleteResp.StatusCode)
		}

		// Step 6: GET /api/ai-sessions → session status is "disconnected".
		resp, err = http.Get(srv.URL + "/api/ai-sessions")
		if err != nil {
			t.Fatalf("GET /api/ai-sessions after deregister: %v", err)
		}
		var sessionsAfter []domain.AISession
		if err := json.NewDecoder(resp.Body).Decode(&sessionsAfter); err != nil {
			resp.Body.Close()
			t.Fatalf("decode sessions list after deregister: %v", err)
		}
		resp.Body.Close()
		if len(sessionsAfter) != 1 {
			t.Fatalf("expected 1 session after deregister, got %d", len(sessionsAfter))
		}
		if sessionsAfter[0].Status != domain.SessionStatusDisconnected {
			t.Errorf("expected status %q after deregister, got %q",
				domain.SessionStatusDisconnected, sessionsAfter[0].Status)
		}

		// Step 7: DELETE /api/ai-sessions/nonexistent-id → 404.
		req, err = http.NewRequestWithContext(context.Background(),
			http.MethodDelete, srv.URL+"/api/ai-sessions/nonexistent-id-xyz", nil)
		if err != nil {
			t.Fatalf("build DELETE nonexistent request: %v", err)
		}
		notFoundResp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("DELETE /api/ai-sessions/nonexistent-id-xyz: %v", err)
		}
		notFoundResp.Body.Close()
		if notFoundResp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 for nonexistent session, got %d", notFoundResp.StatusCode)
		}
	})
}
