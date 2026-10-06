package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

func decodeBody[T any](t *testing.T, body string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatalf("response is not the expected JSON shape: %v\n%s", err, body)
	}
	return v
}

// These tests pin the wire contract of the real handlers (they replace tests
// that used to run against a hand-copied router and so never exercised
// the production code).
func TestWireContract_Tasks(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	h := realHandler(newFailOrch(), nil)

	rec := send(h, "POST", "/api/tasks", `{"instruction":"do it","projectPath":"/p"}`)
	got := decodeBody[map[string]string](t, rec.Body.String())
	if rec.Code != 201 || got["task_id"] != "task-1" || got["status"] != "QUEUED" {
		t.Errorf("create: %d %v", rec.Code, got)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content type %q", ct)
	}

	rec = send(h, "POST", "/api/tasks", `{`)
	if rec.Code != 400 || decodeBody[map[string]string](t, rec.Body.String())["error"] == "" {
		t.Errorf("invalid JSON: %d %s", rec.Code, rec.Body)
	}

	rec = send(h, "GET", "/api/tasks", "")
	tasks := decodeBody[[]domain.Task](t, rec.Body.String())
	if rec.Code != 200 || len(tasks) != 1 || tasks[0].ID != "q1" {
		t.Errorf("queue: %d %v", rec.Code, tasks)
	}
	if rec := send(realHandler(emptyOrch(), nil), "GET", "/api/tasks", ""); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("an empty queue must be [] not null: %q", rec.Body)
	}

	rec = send(h, "GET", "/api/tasks/abc", "")
	task := decodeBody[domain.Task](t, rec.Body.String())
	if rec.Code != 200 || task.ID != "abc" {
		t.Errorf("get: %d %+v", rec.Code, task)
	}
	rec = send(realHandler(newFailOrch().failWith("GetTask", domain.ErrNotFound), nil), "GET", "/api/tasks/abc", "")
	if rec.Code != 404 || decodeBody[map[string]string](t, rec.Body.String())["error"] != "task not found" {
		t.Errorf("not found: %d %s", rec.Code, rec.Body)
	}

	rec = send(h, "DELETE", "/api/tasks/abc", "")
	if rec.Code != 204 || rec.Body.Len() != 0 {
		t.Errorf("cancel: %d %q", rec.Code, rec.Body)
	}
	rec = send(realHandler(newFailOrch().failWith("CancelTask", errTestConflict), nil), "DELETE", "/api/tasks/abc", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("cancelling a finished task must be a 409, got %d", rec.Code)
	}
}

func TestWireContract_TaskDurationIsComputed(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	o := &durationOrch{failOrch: newFailOrch()}
	rec := send(realHandler(o, nil), "GET", "/api/tasks/x", "")
	if got := decodeBody[domain.Task](t, rec.Body.String()); got.DurationMs < 4900 || got.DurationMs > 5100 {
		t.Errorf("DurationMs = %d, want about 5000", got.DurationMs)
	}
}

type durationOrch struct{ *failOrch }

func (durationOrch) GetTask(string) (domain.Task, error) {
	now := time.Now()
	return domain.Task{ID: "x", CreatedAt: now.Add(-5 * time.Second), UpdatedAt: now}, nil
}

func TestWireContract_Providers(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	h := realHandler(newFailOrch(), nil)

	if rec := send(h, "GET", "/api/providers", ""); rec.Code != 200 || len(decodeBody[[]map[string]any](t, rec.Body.String())) != 1 {
		t.Errorf("list: %d %s", rec.Code, rec.Body)
	}
	rec := send(h, "POST", "/api/providers", `{"name":"n","kind":"ollama"}`)
	if got := decodeBody[map[string]string](t, rec.Body.String()); rec.Code != 201 || got["name"] != "n" || got["kind"] != "ollama" {
		t.Errorf("register: %d %v", rec.Code, got)
	}
	rec = send(h, "POST", "/api/providers", `{"name":"n"}`)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "name and kind are required") {
		t.Errorf("missing kind: %d %s", rec.Code, rec.Body)
	}
	rec = send(h, "GET", "/api/providers/n/models", "")
	if got := decodeBody[[]string](t, rec.Body.String()); rec.Code != 200 || len(got) != 1 || got[0] != "m1" {
		t.Errorf("models: %d %v", rec.Code, got)
	}
	if rec := send(realHandler(emptyOrch(), nil), "GET", "/api/providers/n/models", ""); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("no models must be [] not null: %q", rec.Body)
	}
	if rec := send(realHandler(emptyOrch(), nil), "GET", "/api/providers", ""); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("no providers must be [] not null: %q", rec.Body)
	}
}

func TestWireContract_HealthAndSecurityHeaders(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	rec := send(realHandler(newFailOrch(), nil), "GET", "/api/health", "")
	got := decodeBody[map[string]string](t, rec.Body.String())
	if rec.Code != 200 || got["status"] != "ok" || got["service"] != "nexus-orchestrator" {
		t.Errorf("health: %d %v", rec.Code, got)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("security headers: %v", rec.Header())
	}
}

func TestWireContract_ProviderConfigsNeverExposeAPIKeys(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	h := realHandler(newFailOrch(), nil)
	for _, tc := range []struct{ method, path, body string }{
		{"GET", "/api/providers/config", ""},
		{"POST", "/api/providers/config", `{"name":"n","kind":"openaicompat","apiKey":"sk-secret-1234"}`},
		{"PUT", "/api/providers/config/pc1", `{"name":"n","kind":"openaicompat"}`},
	} {
		rec := send(h, tc.method, tc.path, tc.body)
		if rec.Code >= 300 {
			t.Fatalf("%s %s: %d", tc.method, tc.path, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "sk-secret-1234") {
			t.Errorf("%s %s leaked the API key: %s", tc.method, tc.path, rec.Body)
		}
		if !strings.Contains(rec.Body.String(), "****1234") {
			t.Errorf("%s %s should show the masked key: %s", tc.method, tc.path, rec.Body)
		}
	}
	// A key of four characters or fewer is fully masked.
	rec := send(realHandler(shortKeyOrch(), nil), "GET", "/api/providers/config", "")
	if !strings.Contains(rec.Body.String(), `"apiKey":"****"`) || strings.Contains(rec.Body.String(), "abcd") {
		t.Errorf("short key: %s", rec.Body)
	}
}

func TestWireContract_ConfigNeverReturnsTokensOnGetOnlyOnExplicitPut(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	o := newFailOrch()
	h := realHandler(o, nil)
	rec := send(h, "GET", "/api/config", "")
	cfg := decodeBody[map[string]any](t, rec.Body.String())
	if _, leaked := cfg["apiToken"]; leaked || cfg["queueCap"] != float64(50) {
		t.Errorf("GET config: %v", cfg)
	}
	// Setting a queue cap alone must not echo any token.
	rec = send(h, "PUT", "/api/config", `{"queueCap":9}`)
	if got := rec.Body.String(); strings.Contains(got, "new-api") || strings.Contains(got, "new-mcp") {
		t.Errorf("tokens echoed without being set: %s", got)
	}
	// Rotating returns the new token exactly once, in the PUT response.
	rec = send(h, "PUT", "/api/config", `{"rotateApiToken":true,"rotateMcpToken":true}`)
	got := decodeBody[map[string]any](t, rec.Body.String())
	if got["apiToken"] != "new-api" || got["mcpToken"] != "new-mcp" || got["apiTokenEnabled"] != true {
		t.Errorf("rotate: %v", got)
	}
	rec = send(h, "PUT", "/api/config", `{"apiToken":"mine"}`)
	if got := decodeBody[map[string]any](t, rec.Body.String()); got["apiToken"] != "new-api" {
		t.Errorf("explicit set returns the stored token: %v", got)
	}
}

var errTestConflict = &conflictErr{}

type conflictErr struct{}

func (*conflictErr) Error() string {
	return "orchestrator: cancel task: cannot cancel task with status COMPLETED"
}

// emptyOrch returns nil/empty collections so handlers must encode them as [].
func emptyOrch() *nilListOrch { return &nilListOrch{failOrch: newFailOrch()} }

type nilListOrch struct{ *failOrch }

func (nilListOrch) GetQueue() ([]domain.Task, error)            { return nil, nil }
func (nilListOrch) GetProviders() ([]ports.ProviderInfo, error) { return nil, nil }
func (nilListOrch) GetProviderModels(string) ([]string, error)  { return nil, nil }

func shortKeyOrch() *shortKeys { return &shortKeys{failOrch: newFailOrch()} }

type shortKeys struct{ *failOrch }

func (shortKeys) ListProviderConfigs(context.Context) ([]domain.ProviderConfig, error) {
	return []domain.ProviderConfig{{ID: "p", Name: "n", APIKey: "abcd"}}, nil
}
