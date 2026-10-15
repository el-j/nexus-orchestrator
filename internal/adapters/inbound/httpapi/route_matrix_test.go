package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

// route describes one endpoint and how it must respond when the orchestrator
// operation behind it succeeds, fails generically, or reports "not found".
type route struct {
	name     string
	method   string
	path     string
	body     string
	op       string // failOrch operation that backs the route
	ok       int    // status on success
	fail     int    // status on a generic failure (0 = the route never surfaces the failure)
	notFound int    // status when the operation reports domain.ErrNotFound (0 = same as fail)
	badBody  string // a body the route must reject with 400 (empty = no body validation)
}

func routes() []route {
	return []route{
		// ── tasks ──
		{"create task", "POST", "/api/tasks", `{"instruction":"x"}`, "SubmitTask", 201, 500, 0, `{`},
		{"list queue", "GET", "/api/tasks", "", "GetQueue", 200, 500, 0, ""},
		{"list queue by project", "GET", "/api/tasks?projectPath=/p", "", "GetQueueForProject", 200, 500, 0, ""},
		{"list all", "GET", "/api/tasks/all", "", "GetAllTasks", 200, 500, 0, ""},
		{"list all by project", "GET", "/api/tasks/all?projectPath=/p", "", "GetTasksForProject", 200, 500, 0, ""},
		{"get task", "GET", "/api/tasks/t1", "", "GetTask", 200, 500, 404, ""},
		{"cancel task", "DELETE", "/api/tasks/t1", "", "CancelTask", 204, 500, 404, ""},
		{"create draft", "POST", "/api/tasks/draft", `{"instruction":"x","projectPath":"/p"}`, "CreateDraft", 201, 400, 400, `{`},
		{"backlog", "GET", "/api/tasks/backlog?project=/p", "", "GetBacklog", 200, 500, 0, ""},
		{"promote task", "POST", "/api/tasks/t1/promote", "", "PromoteTask", 200, 400, 404, ""},
		{"update task", "PUT", "/api/tasks/t1", `{"instruction":"y"}`, "UpdateTask", 200, 400, 404, `{`},
		{"claim task", "POST", "/api/tasks/t1/claim", `{"sessionId":"s1"}`, "ClaimTask", 200, 500, 404, `{}`},
		{"update status", "PUT", "/api/tasks/t1/status", `{"sessionId":"s1","status":"COMPLETED"}`, "UpdateTaskStatus", 200, 500, 404, `{"sessionId":"s1","status":"RUNNING"}`},
		{"heartbeat task", "POST", "/api/tasks/t1/heartbeat", `{"sessionId":"s1"}`, "HeartbeatTask", 204, 500, 404, `{}`},
		{"session tasks", "GET", "/api/ai-sessions/sess-1/tasks", "", "GetAllTasks", 200, 500, 0, ""},
		// ── providers ──
		{"list providers", "GET", "/api/providers", "", "GetProviders", 200, 500, 0, ""},
		{"register provider", "POST", "/api/providers", `{"name":"n","kind":"ollama"}`, "RegisterCloudProvider", 201, 422, 409, `{"name":"n"}`},
		{"remove provider", "DELETE", "/api/providers/n", "", "RemoveProvider", 204, 500, 404, ""},
		{"provider models", "GET", "/api/providers/n/models", "", "GetProviderModels", 200, 500, 404, ""},
		{"add provider config", "POST", "/api/providers/config", `{"name":"n","kind":"openaicompat","apiKey":"sk-secret-1234"}`, "AddProviderConfig", 201, 500, 0, `{"kind":"x"}`},
		{"list provider configs", "GET", "/api/providers/config", "", "ListProviderConfigs", 200, 500, 0, ""},
		{"update provider config", "PUT", "/api/providers/config/pc1", `{"name":"n","kind":"ollama"}`, "UpdateProviderConfig", 200, 500, 404, `{`},
		{"remove provider config", "DELETE", "/api/providers/config/pc1", "", "RemoveProviderConfig", 204, 500, 404, ""},
		{"discovered providers", "GET", "/api/providers/discovered", "", "GetDiscoveredProviders", 200, 500, 0, ""},
		{"scan", "POST", "/api/providers/discovered/scan", "", "TriggerScan", 200, 500, 0, ""},
		{"promote provider", "POST", "/api/providers/promote/d1", "", "PromoteProvider", 204, 400, 404, ""},
		// ── AI sessions ──
		{"register session", "POST", "/api/ai-sessions", `{"agentName":"a","source":"http"}`, "RegisterAISession", 201, 500, 0, `{`},
		{"list sessions", "GET", "/api/ai-sessions", "", "ListAISessions", 200, 500, 0, ""},
		{"purge sessions", "DELETE", "/api/ai-sessions", "", "PurgeDisconnectedSessions", 200, 500, 0, ""},
		{"deregister session", "DELETE", "/api/ai-sessions/s1", "", "DeregisterAISession", 204, 500, 404, ""},
		{"heartbeat session", "POST", "/api/ai-sessions/s1/heartbeat", "", "HeartbeatAISession", 204, 500, 404, ""},
		{"terminate session", "POST", "/api/ai-sessions/s1/terminate", `{"force":true}`, "TerminateAISession", 204, 500, 404, `{`},
		{"delegate", "POST", "/api/ai-sessions/s1/delegate", "", "DelegateToNexus", 200, 500, 404, ""},
		{"discovered agents", "GET", "/api/ai-sessions/discovered", "", "GetDiscoveredAgents", 200, 200, 0, ""},
		{"plan files", "GET", "/api/plans/discovered?projectPath=/p", "", "GetDiscoveredPlanFiles", 200, 500, 0, ""},
		{"scan plan files", "POST", "/api/plans/discovered/scan?projectPath=/p", "", "GetDiscoveredPlanFiles", 200, 500, 0, ""},
		// ── runtime config ──
		{"get config", "GET", "/api/config", "", "GetRuntimeConfig", 200, 500, 0, ""},
		{"put config", "PUT", "/api/config", `{"queueCap":9}`, "UpdateRuntimeConfig", 200, 400, 0, `{`},
	}
}

func TestEveryRoute_SuccessAndFailureMapping(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	t.Setenv("NEXUS_ALLOWED_ORIGINS", "")
	for _, rt := range routes() {
		t.Run(rt.name, func(t *testing.T) {
			// success
			ok := newFailOrch()
			rec := send(realHandler(ok, nil), rt.method, rt.path, rt.body)
			if rec.Code != rt.ok {
				t.Fatalf("success: status %d, want %d (body %s)", rec.Code, rt.ok, rec.Body)
			}
			if rt.ok != 204 && !json.Valid(rec.Body.Bytes()) {
				t.Errorf("success body is not JSON: %q", rec.Body)
			}

			// generic failure: never leaks the internal message, always valid JSON error
			bad := newFailOrch().failWith(rt.op, errors.New("secret internal detail: /var/db/x"))
			rec = send(realHandler(bad, nil), rt.method, rt.path, rt.body)
			if rec.Code != rt.fail {
				t.Errorf("failure: status %d, want %d (body %s)", rec.Code, rt.fail, rec.Body)
			}
			if rt.fail >= 500 && strings.Contains(rec.Body.String(), "secret internal detail") {
				t.Errorf("a 5xx response must not leak internal error text: %s", rec.Body)
			}

			// not found
			nf := newFailOrch().failWith(rt.op, domain.ErrNotFound)
			rec = send(realHandler(nf, nil), rt.method, rt.path, rt.body)
			want := rt.notFound
			if want == 0 {
				want = rt.fail
			}
			if rec.Code != want {
				t.Errorf("not found: status %d, want %d (body %s)", rec.Code, want, rec.Body)
			}

			// body validation
			if rt.badBody != "" {
				rec = send(realHandler(newFailOrch(), nil), rt.method, rt.path, rt.badBody)
				if rec.Code != http.StatusBadRequest {
					t.Errorf("bad body %q: status %d, want 400", rt.badBody, rec.Code)
				}
			}
		})
	}
}
