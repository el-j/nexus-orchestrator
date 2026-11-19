//go:build e2e

package e2e_test

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

// A session that registers can be kept alive, listed with the tasks it claimed, and terminated; a task it
// claimed is kept from going stale by its heartbeat.
func TestDaemonKeepsASessionAndItsClaimedTaskAlive(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()
	project := t.TempDir()

	var session struct {
		ID string `json:"id"`
	}
	if code := d.do(http.MethodPost, "/api/ai-sessions", map[string]any{"agentName": "keepalive", "source": "http", "projectPath": project}, &session); code != http.StatusCreated {
		t.Fatalf("register: HTTP %d\n%s", code, d.logs())
	}
	if code := d.do(http.MethodPost, "/api/ai-sessions/"+session.ID+"/heartbeat", nil, nil); code != http.StatusNoContent && code != http.StatusOK {
		t.Errorf("session heartbeat: HTTP %d", code)
	}

	var draft map[string]string
	d.do(http.MethodPost, "/api/tasks/draft", map[string]any{"projectPath": project, "instruction": "work by hand", "targetFile": "x.txt"}, &draft)
	id := draft["id"]
	d.do(http.MethodPost, "/api/tasks/"+id+"/promote", nil, nil)
	d.waitStatus(id, "NO_PROVIDER") // the daemon has no model; the agent takes the task from there
	if code := d.do(http.MethodPost, "/api/tasks/"+id+"/claim", map[string]any{"sessionId": session.ID}, nil); code != http.StatusOK && code != http.StatusNoContent {
		t.Fatalf("claim: HTTP %d\n%s", code, d.logs())
	}
	if code := d.do(http.MethodPost, "/api/tasks/"+id+"/heartbeat", map[string]any{"sessionId": session.ID}, nil); code != http.StatusNoContent && code != http.StatusOK {
		t.Errorf("task heartbeat: HTTP %d\n%s", code, d.logs())
	}
	var claimed []struct {
		ID string `json:"id"`
	}
	if code := d.do(http.MethodGet, "/api/ai-sessions/"+session.ID+"/tasks", nil, &claimed); code != http.StatusOK || len(claimed) != 1 || claimed[0].ID != id {
		t.Errorf("the session's tasks: HTTP %d %+v", code, claimed)
	}
	if code := d.do(http.MethodPost, "/api/ai-sessions/"+session.ID+"/terminate", map[string]any{"force": false}, nil); code != http.StatusNoContent && code != http.StatusOK {
		t.Errorf("terminate: HTTP %d\n%s", code, d.logs())
	}
}

// A stored provider configuration can be changed and removed, and a provider's models can be listed.
func TestDaemonEditsAndRemovesAStoredProviderConfiguration(t *testing.T) {
	llm := newFakeOllama(t)
	d := startDaemon(t, llm, daemonOpts{})
	d.waitHealthy()

	var created struct {
		ID string `json:"id"`
	}
	cfg := map[string]any{"name": "e2e-ollama", "kind": "ollama", "baseUrl": llm.URL, "model": "m1", "enabled": true}
	if code := d.do(http.MethodPost, "/api/providers/config", cfg, &created); code != http.StatusCreated || created.ID == "" {
		t.Fatalf("add config: HTTP %d %+v\n%s", code, created, d.logs())
	}
	cfg["model"] = "m2"
	if code := d.do(http.MethodPut, "/api/providers/config/"+created.ID, cfg, nil); code != http.StatusOK && code != http.StatusNoContent {
		t.Fatalf("update config: HTTP %d\n%s", code, d.logs())
	}
	var configs []struct {
		ID    string `json:"id"`
		Model string `json:"model"`
	}
	d.do(http.MethodGet, "/api/providers/config", nil, &configs)
	if len(configs) != 1 || configs[0].Model != "m2" {
		t.Errorf("stored configurations = %+v", configs)
	}

	var providers []struct {
		Name string `json:"name"`
	}
	d.do(http.MethodGet, "/api/providers", nil, &providers)
	var ollama string
	for _, p := range providers {
		if strings.Contains(strings.ToLower(p.Name), "ollama") {
			ollama = p.Name
		}
	}
	if ollama == "" {
		t.Fatalf("no Ollama provider is registered although the fake Ollama answers: %+v", providers)
	}
	var models []string
	if code := d.do(http.MethodGet, "/api/providers/"+url.PathEscape(ollama)+"/models", nil, &models); code != http.StatusOK || len(models) == 0 {
		t.Errorf("models of %q: HTTP %d %v", ollama, code, models)
	}

	if code := d.do(http.MethodDelete, "/api/providers/config/"+created.ID, nil, nil); code != http.StatusNoContent && code != http.StatusOK {
		t.Errorf("remove config: HTTP %d", code)
	}
	if code := d.do(http.MethodDelete, "/api/providers/config/"+created.ID, nil, nil); code != http.StatusNotFound {
		t.Errorf("removing a configuration twice: HTTP %d, want 404", code)
	}
}

// A knowledge entry stored for a project is listed, appears in the file map when it is one, and is removed
// by id.
func TestDaemonStoresReadsAndDeletesAKnowledgeEntry(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()
	project := t.TempDir()
	mustWrite(t, filepath.Join(project, "main.go"), "package main\n")

	var stored struct {
		ID string `json:"id"`
	}
	entry := map[string]any{"projectPath": project, "kind": "learning", "topic": "e2e lesson", "content": "always test the route"}
	if code := d.do(http.MethodPost, "/api/brain/knowledge", entry, &stored); code != http.StatusCreated || stored.ID == "" {
		t.Fatalf("store: HTTP %d %+v\n%s", code, stored, d.logs())
	}
	var fileMap struct {
		Content string `json:"content"`
	}
	if code := d.do(http.MethodGet, "/api/brain/file-map?projectPath="+url.QueryEscape(project), nil, &fileMap); code != http.StatusOK {
		t.Errorf("file map: HTTP %d\n%s", code, d.logs())
	}
	if code := d.do(http.MethodDelete, "/api/brain/knowledge/"+stored.ID, nil, nil); code != http.StatusNoContent && code != http.StatusOK {
		t.Fatalf("delete: HTTP %d", code)
	}
	var entries []struct {
		Topic string `json:"topic"`
	}
	d.do(http.MethodGet, "/api/brain/knowledge?projectPath="+url.QueryEscape(project)+"&kind=learning", nil, &entries)
	for _, e := range entries {
		if strings.Contains(e.Topic, "e2e lesson") {
			t.Errorf("the deleted entry is still listed: %+v", entries)
		}
	}
}
