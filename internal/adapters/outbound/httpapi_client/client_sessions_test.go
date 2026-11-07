package httpapi_client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
	"nexus-orchestrator/internal/core/domain"
)

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

func TestClient_GetDiscoveredAgents(t *testing.T) {
	want := []domain.DiscoveredAgent{{ID: "agent-1", Name: "copilot", IsRunning: true}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/ai-sessions/discovered" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.GetDiscoveredAgents(context.Background())
	if err != nil {
		t.Fatalf("GetDiscoveredAgents: %v", err)
	}
	if len(got) != 1 || got[0].Name != "copilot" {
		t.Errorf("unexpected agents: %+v", got)
	}

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).GetDiscoveredAgents(context.Background()); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_DelegateToNexus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		switch r.URL.Path {
		case "/api/ai-sessions/sess-1/delegate":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"instruction": "do work"}) //nolint:errcheck
		case "/api/ai-sessions/unknown/delegate":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	inst, err := c.DelegateToNexus(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("DelegateToNexus: %v", err)
	}
	if inst != "do work" {
		t.Errorf("expected instruction 'do work', got %q", inst)
	}

	if _, err := c.DelegateToNexus(context.Background(), "unknown"); err == nil {
		t.Error("expected not-found error, got nil")
	}
	if _, err := c.DelegateToNexus(context.Background(), "err"); err == nil {
		t.Error("expected error on 500, got nil")
	}
}
