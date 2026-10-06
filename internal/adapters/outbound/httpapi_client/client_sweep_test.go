package httpapi_client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
	"nexus-orchestrator/internal/core/domain"
)

type sweepCase struct {
	name    string
	run     func(c *httpapi_client.Client) error
	success int  // the status a healthy daemon answers with
	decodes bool // whether the success response body is decoded
}

func sweepCases() []sweepCase {
	ctx := context.Background()
	return []sweepCase{
		{"SubmitTask", func(c *httpapi_client.Client) error { _, err := c.SubmitTask(domain.Task{}); return err }, 201, true},
		{"GetTask", func(c *httpapi_client.Client) error { _, err := c.GetTask("t"); return err }, 200, true},
		{"GetQueue", func(c *httpapi_client.Client) error { _, err := c.GetQueue(); return err }, 200, true},
		{"GetAllTasks", func(c *httpapi_client.Client) error { _, err := c.GetAllTasks(); return err }, 200, true},
		{"GetQueueForProject", func(c *httpapi_client.Client) error { _, err := c.GetQueueForProject("/p"); return err }, 200, true},
		{"GetTasksForProject", func(c *httpapi_client.Client) error { _, err := c.GetTasksForProject("/p"); return err }, 200, true},
		{"CancelTask", func(c *httpapi_client.Client) error { return c.CancelTask("t") }, 204, false},
		{"CreateDraft", func(c *httpapi_client.Client) error { _, err := c.CreateDraft(domain.Task{}); return err }, 201, true},
		{"GetBacklog", func(c *httpapi_client.Client) error { _, err := c.GetBacklog("/p"); return err }, 200, true},
		{"PromoteTask", func(c *httpapi_client.Client) error { _, err := c.PromoteTask("t"); return err }, 200, true},
		{"UpdateTask", func(c *httpapi_client.Client) error { _, err := c.UpdateTask("t", domain.Task{}); return err }, 200, true},
		{"ClaimTask", func(c *httpapi_client.Client) error { _, err := c.ClaimTask(ctx, "t", "s"); return err }, 200, true},
		{"UpdateTaskStatus", func(c *httpapi_client.Client) error {
			_, err := c.UpdateTaskStatus(ctx, "t", "s", domain.StatusCompleted, "")
			return err
		}, 200, true},
		{"HeartbeatTask", func(c *httpapi_client.Client) error { return c.HeartbeatTask(ctx, "t", "s") }, 204, false},
		{"GetProviders", func(c *httpapi_client.Client) error { _, err := c.GetProviders(); return err }, 200, true},
		{"RegisterCloudProvider", func(c *httpapi_client.Client) error { return c.RegisterCloudProvider(domain.ProviderConfig{}) }, 201, false},
		{"RemoveProvider", func(c *httpapi_client.Client) error { return c.RemoveProvider("p") }, 204, false},
		{"GetProviderModels", func(c *httpapi_client.Client) error { _, err := c.GetProviderModels("p"); return err }, 200, true},
		{"AddProviderConfig", func(c *httpapi_client.Client) error {
			_, err := c.AddProviderConfig(ctx, domain.ProviderConfig{})
			return err
		}, 201, true},
		{"UpdateProviderConfig", func(c *httpapi_client.Client) error {
			_, err := c.UpdateProviderConfig(ctx, domain.ProviderConfig{ID: "p"})
			return err
		}, 200, true},
		{"RemoveProviderConfig", func(c *httpapi_client.Client) error { return c.RemoveProviderConfig(ctx, "p") }, 204, false},
		{"ListProviderConfigs", func(c *httpapi_client.Client) error { _, err := c.ListProviderConfigs(ctx); return err }, 200, true},
		{"GetDiscoveredProviders", func(c *httpapi_client.Client) error { _, err := c.GetDiscoveredProviders(); return err }, 200, true},
		{"TriggerScan", func(c *httpapi_client.Client) error { _, err := c.TriggerScan(ctx); return err }, 200, true},
		{"PromoteProvider", func(c *httpapi_client.Client) error { return c.PromoteProvider(ctx, "p") }, 200, false},
		{"GetRuntimeConfig", func(c *httpapi_client.Client) error { _, err := c.GetRuntimeConfig(ctx); return err }, 200, true},
		{"UpdateRuntimeConfig", func(c *httpapi_client.Client) error {
			_, err := c.UpdateRuntimeConfig(ctx, domain.RuntimeConfigUpdate{})
			return err
		}, 200, true},
		{"RegisterAISession", func(c *httpapi_client.Client) error {
			_, err := c.RegisterAISession(ctx, domain.AISession{})
			return err
		}, 201, true},
		{"ListAISessions", func(c *httpapi_client.Client) error { _, err := c.ListAISessions(ctx); return err }, 200, true},
		{"DeregisterAISession", func(c *httpapi_client.Client) error { return c.DeregisterAISession(ctx, "s") }, 204, false},
		{"TerminateAISession", func(c *httpapi_client.Client) error { return c.TerminateAISession(ctx, "s", true) }, 200, false},
		{"HeartbeatAISession", func(c *httpapi_client.Client) error { return c.HeartbeatAISession(ctx, "s") }, 204, false},
		{"PurgeDisconnectedSessions", func(c *httpapi_client.Client) error { _, err := c.PurgeDisconnectedSessions(ctx); return err }, 200, true},
		{"GetDiscoveredAgents", func(c *httpapi_client.Client) error { _, err := c.GetDiscoveredAgents(ctx); return err }, 200, true},
		{"DelegateToNexus", func(c *httpapi_client.Client) error { _, err := c.DelegateToNexus(ctx, "s"); return err }, 200, true},
		{"GetDiscoveredPlanFiles", func(c *httpapi_client.Client) error {
			_, err := c.GetDiscoveredPlanFiles(ctx, "/p")
			return err
		}, 200, true},
	}
}

// listMethods return JSON arrays; a healthy daemon answers them with [].
var listMethods = map[string]bool{
	"GetQueue": true, "GetAllTasks": true, "GetQueueForProject": true, "GetTasksForProject": true,
	"GetBacklog": true, "GetProviders": true, "GetProviderModels": true, "ListProviderConfigs": true,
	"GetDiscoveredProviders": true, "TriggerScan": true, "ListAISessions": true,
	"GetDiscoveredAgents": true, "GetDiscoveredPlanFiles": true,
}

func fixedServer(status int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func TestEveryClientMethod_FailureModes(t *testing.T) {
	for _, tc := range sweepCases() {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Daemon unreachable.
			dead := fixedServer(200, "{}")
			url := dead.URL
			dead.Close()
			if err := tc.run(httpapi_client.NewClient(url)); err == nil || !strings.Contains(err.Error(), "remote:") {
				t.Errorf("unreachable: want a wrapped remote error, got %v", err)
			}

			// 2. An unparsable base URL fails while building the request.
			if err := tc.run(httpapi_client.NewClient("http://bad host")); err == nil || !strings.Contains(err.Error(), "build request") {
				t.Errorf("bad URL: want a build-request error, got %v", err)
			}

			// 3. A server error carries status and message.
			srv := fixedServer(500, `{"error":"boom"}`)
			err := tc.run(httpapi_client.NewClient(srv.URL))
			srv.Close()
			if err == nil || !strings.Contains(err.Error(), "unexpected status 500") || !strings.Contains(err.Error(), "boom") {
				t.Errorf("500: want status+message, got %v", err)
			}

			// 4. A non-JSON error body still yields a clean status error.
			srv = fixedServer(502, "<html>bad gateway</html>")
			err = tc.run(httpapi_client.NewClient(srv.URL))
			srv.Close()
			if err == nil || !strings.Contains(err.Error(), "unexpected status 502") || strings.Contains(err.Error(), "html") {
				t.Errorf("502: got %v", err)
			}

			// 5. A malformed success body is a decode error where a body is expected.
			if tc.decodes {
				srv = fixedServer(tc.success, "not json")
				err = tc.run(httpapi_client.NewClient(srv.URL))
				srv.Close()
				if err == nil || !strings.Contains(err.Error(), "decode") {
					t.Errorf("malformed body: want a decode error, got %v", err)
				}
			}
		})
	}
}

func TestEveryClientMethod_SucceedsOnHealthyDaemon(t *testing.T) {
	for _, tc := range sweepCases() {
		t.Run(tc.name, func(t *testing.T) {
			body := `{}`
			switch {
			case tc.success == 204:
				body = ""
			case listMethods[tc.name]:
				body = `[]`
			}
			srv := fixedServer(tc.success, body)
			defer srv.Close()
			if err := tc.run(httpapi_client.NewClient(srv.URL)); err != nil {
				t.Errorf("healthy daemon: %v", err)
			}
		})
	}
}

func TestClient_SendsBearerTokenFromEnvironment(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "  secret  ")
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	if _, err := httpapi_client.NewClient(srv.URL).GetQueue(); err != nil {
		t.Fatal(err)
	}
	if got != "Bearer secret" {
		t.Errorf("Authorization = %q", got)
	}
}

func TestClient_NewClientWithHTTPUsesTheProvidedTransport(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`[]`)) }))
	defer srv.Close()
	hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		return http.DefaultTransport.RoundTrip(r)
	})}
	if _, err := httpapi_client.NewClientWithHTTP(srv.URL+"/", hc).GetQueue(); err != nil || !called {
		t.Errorf("custom transport not used: called=%v err=%v", called, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClient_ErrorBodyMessageIsBounded(t *testing.T) {
	huge := `{"error":"` + strings.Repeat("x", 100_000) + `"}`
	srv := fixedServer(500, huge)
	defer srv.Close()
	_, err := httpapi_client.NewClient(srv.URL).GetQueue()
	if err == nil || len(err.Error()) > 8_000 {
		t.Fatalf("error text must stay bounded, got %d bytes", len(err.Error()))
	}
}
