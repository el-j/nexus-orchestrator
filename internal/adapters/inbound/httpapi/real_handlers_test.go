package httpapi_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/adapters/inbound/httpguard"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// spyOrch wraps mockOrchestrator, counting every state-changing call and letting
// a test force one error onto every operation that can fail.
type spyOrch struct {
	mockOrchestrator
	calls atomic.Int32
	fail  error // when non-nil, every fallible operation returns it

	registerSession func(domain.AISession) (domain.AISession, error)
	listSessions    func() ([]domain.AISession, error)
	deregister      func(string) error
	agents          func() ([]domain.DiscoveredAgent, error)
	planFiles       func(string) ([]domain.DiscoveredPlanFile, error)
	scan            func() ([]domain.DiscoveredProvider, error)
	discovered      func() ([]domain.DiscoveredProvider, error)
	updateCfg       func(domain.RuntimeConfigUpdate) (domain.RuntimeConfig, error)
	getCfg          func() (domain.RuntimeConfig, error)
}

func (s *spyOrch) SubmitTask(t domain.Task) (string, error) {
	s.calls.Add(1)
	if s.fail != nil {
		return "", s.fail
	}
	return "task-1", nil
}
func (s *spyOrch) RegisterAISession(_ context.Context, a domain.AISession) (domain.AISession, error) {
	s.calls.Add(1)
	if s.registerSession != nil {
		return s.registerSession(a)
	}
	return a, nil
}
func (s *spyOrch) TerminateAISession(_ context.Context, id string, _ bool) error {
	s.calls.Add(1)
	return s.fail
}
func (s *spyOrch) ListAISessions(context.Context) ([]domain.AISession, error) {
	if s.listSessions != nil {
		return s.listSessions()
	}
	return nil, s.fail
}
func (s *spyOrch) DeregisterAISession(_ context.Context, id string) error {
	if s.deregister != nil {
		return s.deregister(id)
	}
	return s.fail
}
func (s *spyOrch) GetDiscoveredAgents(context.Context) ([]domain.DiscoveredAgent, error) {
	if s.agents != nil {
		return s.agents()
	}
	return nil, s.fail
}
func (s *spyOrch) GetDiscoveredPlanFiles(_ context.Context, p string) ([]domain.DiscoveredPlanFile, error) {
	if s.planFiles != nil {
		return s.planFiles(p)
	}
	return nil, s.fail
}
func (s *spyOrch) TriggerScan(context.Context) ([]domain.DiscoveredProvider, error) {
	if s.scan != nil {
		return s.scan()
	}
	return nil, s.fail
}
func (s *spyOrch) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	if s.discovered != nil {
		return s.discovered()
	}
	return nil, s.fail
}
func (s *spyOrch) GetRuntimeConfig(ctx context.Context) (domain.RuntimeConfig, error) {
	if s.getCfg != nil {
		return s.getCfg()
	}
	return s.mockOrchestrator.GetRuntimeConfig(ctx)
}
func (s *spyOrch) UpdateRuntimeConfig(ctx context.Context, u domain.RuntimeConfigUpdate) (domain.RuntimeConfig, error) {
	if s.updateCfg != nil {
		return s.updateCfg(u)
	}
	return s.mockOrchestrator.UpdateRuntimeConfig(ctx, u)
}

var _ ports.Orchestrator = (*spyOrch)(nil)

func realHandler(orch ports.Orchestrator, brain ports.BrainService) http.Handler {
	return httpapi.NewServer(orch, brain, httpapi.NewHub()).Handler()
}

func send(h http.Handler, method, path, body string, headers ...string) *httptest.ResponseRecorder {
	var rdr *bytes.Reader
	if body != "" {
		rdr = bytes.NewReader([]byte(body))
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.ContentLength = int64(len(body))
	}
	for i := 0; i+1 < len(headers); i += 2 {
		if strings.EqualFold(headers[i], "Host") {
			req.Host = headers[i+1]
			continue
		}
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// ── security ────────────────────────────────────────────────────────────────

func TestDriveByCrossSiteWritesNeverReachTheOrchestrator(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	t.Setenv("NEXUS_ALLOWED_ORIGINS", "")
	spy := &spyOrch{}
	h := realHandler(spy, nil)

	// A hostile page can send a "simple" cross-site POST (text/plain, no preflight).
	rec := send(h, http.MethodPost, "/api/tasks", `{"instruction":"rm -rf","projectPath":"/"}`,
		"Origin", "https://evil.example", "Content-Type", "text/plain")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site POST: status %d, want 403", rec.Code)
	}
	for _, p := range []string{
		"/api/ai-sessions",
	} {
		rec = send(h, http.MethodPost, p, `{"agentName":"x","source":"http","pid":1}`, "Origin", "http://attacker.test")
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: status %d, want 403", p, rec.Code)
		}
	}
	rec = send(h, http.MethodPost, "/api/ai-sessions/s1/terminate", `{"force":true}`, "Origin", "null")
	if rec.Code != http.StatusForbidden {
		t.Errorf("terminate from the null origin: %d", rec.Code)
	}
	if spy.calls.Load() != 0 {
		t.Fatalf("the orchestrator was invoked %d times by forbidden requests", spy.calls.Load())
	}

	// The same call from a local origin and from a non-browser client works.
	for _, hdrs := range [][]string{{"Origin", "http://localhost:5173"}, {"Origin", "wails://wails.localhost"}, {}} {
		if rec = send(h, http.MethodPost, "/api/tasks", `{"instruction":"ok","projectPath":"/p"}`, hdrs...); rec.Code != http.StatusCreated {
			t.Errorf("legitimate caller %v: status %d body %s", hdrs, rec.Code, rec.Body)
		}
	}
}

func TestDNSRebindingIsRejectedOnLoopbackBoundServers(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	t.Setenv("NEXUS_ALLOWED_HOSTS", "")
	spy := &spyOrch{}
	h := httpapi.NewServer(spy, nil, nil).WithGuard(httpguard.New("127.0.0.1:63987")).Handler()

	if rec := send(h, http.MethodGet, "/api/health", "", "Host", "attacker.example:63987"); rec.Code != http.StatusForbidden {
		t.Errorf("rebinding hostname: %d", rec.Code)
	}
	if rec := send(h, http.MethodGet, "/api/health", "", "Host", "127.0.0.1:63987"); rec.Code != http.StatusOK {
		t.Errorf("loopback host: %d", rec.Code)
	}
	// A server the operator bound to every interface (containers, LAN) keeps working by hostname.
	open := httpapi.NewServer(spy, nil, nil).WithGuard(httpguard.New("0.0.0.0:63987")).Handler()
	if rec := send(open, http.MethodGet, "/api/health", "", "Host", "nexus-service:63987"); rec.Code != http.StatusOK {
		t.Errorf("container hostname on an all-interfaces bind: %d", rec.Code)
	}
}

func TestBearerTokenAuthentication(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "s3cret-token")
	h := realHandler(&spyOrch{}, nil)

	for _, tc := range []struct {
		name string
		path string
		auth string
		want int
	}{
		{"health is public", "/api/health", "", 200},
		{"howto is public", "/api/howto", "", 200},
		{"well-known is public", "/.well-known/nexus.json", "", 200},
		{"ui is public", "/ui", "", 200},
		{"no header", "/api/tasks", "", 401},
		{"wrong scheme", "/api/tasks", "Basic s3cret-token", 401},
		{"wrong token", "/api/tasks", "Bearer wrong-token", 401},
		{"prefix of the token", "/api/tasks", "Bearer s3cret", 401},
		{"token with extra suffix", "/api/tasks", "Bearer s3cret-token-and-more", 401},
		{"correct token", "/api/tasks", "Bearer s3cret-token", 200},
		{"correct token with padding", "/api/tasks", "Bearer   s3cret-token  ", 200},
	} {
		var hdrs []string
		if tc.auth != "" {
			hdrs = []string{"Authorization", tc.auth}
		}
		if rec := send(h, http.MethodGet, tc.path, "", hdrs...); rec.Code != tc.want {
			t.Errorf("%s: status %d, want %d", tc.name, rec.Code, tc.want)
		}
	}
}

func TestTokenFromRuntimeConfigIsHonouredWhenEnvIsUnset(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	spy := &spyOrch{getCfg: func() (domain.RuntimeConfig, error) { return domain.RuntimeConfig{APIToken: "stored"}, nil }}
	h := realHandler(spy, nil)
	if rec := send(h, http.MethodGet, "/api/tasks", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("stored token must be enforced: %d", rec.Code)
	}
	if rec := send(h, http.MethodGet, "/api/tasks", "", "Authorization", "Bearer stored"); rec.Code != http.StatusOK {
		t.Errorf("stored token accepted: %d", rec.Code)
	}
	// A failing config read must not lock everyone out or crash.
	failing := &spyOrch{getCfg: func() (domain.RuntimeConfig, error) { return domain.RuntimeConfig{}, errors.New("db down") }}
	if rec := send(realHandler(failing, nil), http.MethodGet, "/api/tasks", ""); rec.Code != http.StatusOK {
		t.Errorf("config read failure: %d", rec.Code)
	}
}

func TestCORSHeadersAndPreflight(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	h := realHandler(&spyOrch{}, nil)
	rec := send(h, http.MethodOptions, "/api/tasks", "", "Origin", "http://localhost:5173")
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Errorf("preflight from a local app: %d %v", rec.Code, rec.Header())
	}
	rec = send(h, http.MethodGet, "/api/health", "")
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("no origin, no CORS headers")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("security headers missing: %v", rec.Header())
	}
	if rec := send(h, http.MethodGet, "/", ""); rec.Code != http.StatusFound || rec.Header().Get("Location") != "/ui" {
		t.Errorf("root redirect: %d %v", rec.Code, rec.Header())
	}
	big := strings.Repeat("x", 2<<20)
	if rec := send(h, http.MethodPost, "/api/tasks", `{"instruction":"`+big+`"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("an oversized body must be rejected (1 MiB cap), got %d", rec.Code)
	}
}

// ── contract fixes ──────────────────────────────────────────────────────────

func TestTaskHeartbeat_AcceptsCanonicalAndLegacySessionField(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	h := realHandler(&spyOrch{}, nil)
	for name, body := range map[string]string{
		"sessionId (what the Go client and /claim use)": `{"sessionId":"s1"}`,
		"session_id (legacy / MCP naming)":              `{"session_id":"s1"}`,
	} {
		if rec := send(h, http.MethodPost, "/api/tasks/t1/heartbeat", body); rec.Code != http.StatusNoContent {
			t.Errorf("%s: status %d body %s", name, rec.Code, rec.Body)
		}
	}
	for _, body := range []string{`{}`, `{"sessionId":""}`, `not json`} {
		if rec := send(h, http.MethodPost, "/api/tasks/t1/heartbeat", body); rec.Code != http.StatusBadRequest {
			t.Errorf("body %q: status %d, want 400", body, rec.Code)
		}
	}
}

func TestTaskHeartbeat_ErrorMapping(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	for _, tc := range []struct {
		err  error
		want int
	}{
		{fmt.Errorf("heartbeat task: %w", domain.ErrNotFound), 404},
		{errors.New("heartbeat task: task not processing"), 409},
		{errors.New("heartbeat task: not claimed by session s1"), 409},
		{errors.New("heartbeat task: status changed"), 409},
		{errors.New("disk exploded"), 500},
	} {
		mock := &mockOrchestrator{heartbeatTaskErr: tc.err}
		if rec := send(realHandler(mock, nil), http.MethodPost, "/api/tasks/t/heartbeat", `{"sessionId":"s"}`); rec.Code != tc.want {
			t.Errorf("%v: status %d, want %d", tc.err, rec.Code, tc.want)
		}
	}
}

func TestFullQueueIsA429WithRetryAfter_NotA500(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	full := fmt.Errorf("orchestrator: submit task: %w", domain.ErrQueueFull)

	rec := send(realHandler(&mockOrchestrator{submitTaskErr: full}, nil), http.MethodPost, "/api/tasks", `{"instruction":"x"}`)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Errorf("submit: status %d, Retry-After %q", rec.Code, rec.Header().Get("Retry-After"))
	}
	rec = send(realHandler(&mockOrchestrator{promoteTaskErr: full}, nil), http.MethodPost, "/api/tasks/t/promote", "")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Errorf("promote: status %d", rec.Code)
	}
}

func TestMaskedAPIKeyIsNeverTheStoredValueAfterUpdate(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	mock := &mockOrchestrator{}
	h := realHandler(mock, nil)
	rec := send(h, http.MethodPut, "/api/providers/config/p1", `{"name":"n","kind":"openaicompat","apiKey":"****abcd"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	// The response never echoes a real key (the mock returns what it was given, masked again).
	if strings.Contains(rec.Body.String(), "sk-") {
		t.Errorf("unexpected key material in response: %s", rec.Body)
	}
}
