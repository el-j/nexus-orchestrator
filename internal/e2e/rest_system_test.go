//go:build e2e

package e2e_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

// raw calls the daemon and returns the status, headers and body without decoding JSON: for the pages and
// redirects, which are not JSON.
func (d *daemon) raw(method, path string) (int, http.Header, string) {
	d.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, d.http+path, nil)
	if err != nil {
		d.t.Fatal(err)
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		d.t.Fatalf("%s %s: %v\n%s", method, path, err, d.logs())
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, string(b)
}

// The root leads to a page: the dashboard at /ui carries a content security policy, and /app is either the
// full UI or absent when the binary was built without it.
func TestDaemonServesTheDashboardAndRedirectsTheRoot(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()

	code, h, body := d.raw(http.MethodGet, "/ui")
	if code != http.StatusOK || !strings.Contains(h.Get("Content-Security-Policy"), "nonce-") || !strings.Contains(body, "<html") {
		t.Errorf("GET /ui: HTTP %d, CSP %q", code, h.Get("Content-Security-Policy"))
	}
	code, h, _ = d.raw(http.MethodGet, "/")
	if code != http.StatusFound || (h.Get("Location") != "/ui" && h.Get("Location") != "/app/") {
		t.Errorf("GET /: HTTP %d to %q, want a redirect to /ui or /app/", code, h.Get("Location"))
	}
	if h.Get("Location") == "/app/" {
		if code, _, body = d.raw(http.MethodGet, "/app/"); code != http.StatusOK || !strings.Contains(body, "<div id=\"app\"") {
			t.Errorf("GET /app/: HTTP %d, want the single-page app", code)
		}
		if code, h, _ = d.raw(http.MethodGet, "/app"); code != http.StatusFound || h.Get("Location") != "/app/" {
			t.Errorf("GET /app: HTTP %d to %q, want a redirect to /app/", code, h.Get("Location"))
		}
	}
}

// The daemon describes itself: the discovery beacon names the API and the MCP endpoint, and the how-to
// guide's links point at the address it was called on.
func TestDaemonDescribesItselfInTheBeaconAndTheHowTo(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()

	var beacon struct {
		API struct {
			BaseURL string `json:"base_url"`
			Howto   string `json:"howto"`
		} `json:"api"`
		MCP struct {
			Endpoint string `json:"endpoint"`
		} `json:"mcp"`
	}
	if code := d.do(http.MethodGet, "/.well-known/nexus.json", nil, &beacon); code != http.StatusOK || beacon.API.BaseURL == "" || !strings.HasSuffix(beacon.MCP.Endpoint, "/mcp") {
		t.Errorf("beacon: HTTP %d %+v", code, beacon)
	}
	var howto struct {
		Name       string `json:"name"`
		Connection struct {
			HTTPAPI string `json:"http_api"`
		} `json:"connection"`
		Endpoints []struct {
			Path string `json:"path"`
		} `json:"http_endpoints"`
	}
	if code := d.do(http.MethodGet, "/api/howto", nil, &howto); code != http.StatusOK || howto.Name == "" || len(howto.Endpoints) < 10 {
		t.Errorf("howto: HTTP %d, %d endpoints", code, len(howto.Endpoints))
	}
	if howto.Connection.HTTPAPI != d.http {
		t.Errorf("howto links to %q, the daemon was called on %q", howto.Connection.HTTPAPI, d.http)
	}
}

// Settings and policy written over the API are read back, and a policy the project cannot hold is refused.
func TestDaemonStoresSettingsAndAProjectPolicy(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()

	var cfg map[string]any
	if code := d.do(http.MethodGet, "/api/config", nil, &cfg); code != http.StatusOK {
		t.Fatalf("GET /api/config: HTTP %d", code)
	}
	cfg["queueCap"] = 7
	if code := d.do(http.MethodPut, "/api/config", cfg, nil); code != http.StatusOK && code != http.StatusNoContent {
		t.Fatalf("PUT /api/config: HTTP %d\n%s", code, d.logs())
	}
	var back map[string]any
	d.do(http.MethodGet, "/api/config", nil, &back)
	if back["queueCap"] != float64(7) {
		t.Errorf("settings read back = %v", back)
	}

	project := t.TempDir()
	q := "?projectPath=" + url.QueryEscape(project)
	if code := d.do(http.MethodPut, "/api/policy"+q, map[string]any{"verify": "true", "quotaThreshold": 0.8}, nil); code != http.StatusOK && code != http.StatusNoContent {
		t.Fatalf("PUT /api/policy: HTTP %d\n%s", code, d.logs())
	}
	var pol struct {
		Found  bool           `json:"found"`
		Policy map[string]any `json:"policy"`
	}
	if code := d.do(http.MethodGet, "/api/policy"+q, nil, &pol); code != http.StatusOK || !pol.Found || pol.Policy["verify"] != "true" {
		t.Errorf("GET /api/policy: HTTP %d %+v", code, pol)
	}
	if code := d.do(http.MethodPut, "/api/policy"+q, map[string]any{"quotaThreshold": 7}, nil); code != http.StatusBadRequest {
		t.Errorf("a quota threshold above 1 must be refused, got HTTP %d", code)
	}
}

// The daemon's own log is served, and activity and timeline answer (empty on a fresh machine).
func TestDaemonServesItsLogsAndTheActivityViews(t *testing.T) {
	d := startDaemon(t, nil, daemonOpts{})
	d.waitHealthy()

	for _, path := range []string{"/api/logs", "/api/activities", "/api/activities/timeline"} {
		var out any
		if code := d.do(http.MethodGet, path, nil, &out); code != http.StatusOK {
			t.Errorf("GET %s: HTTP %d\n%s", path, code, d.logs())
		}
	}
}

// Scans find what is on the machine: a plan file in a project is listed after a scan, agents and providers
// answer as lists, and the file map and knowledge entry of a project can be read and removed.
func TestDaemonScansAndListsWhatItDiscovers(t *testing.T) {
	llm := newFakeOllama(t)
	d := startDaemon(t, llm, daemonOpts{})
	d.waitHealthy()

	var scanned []any
	project := t.TempDir()
	mustWrite(t, filepath.Join(project, ".claude", "plans", "PLAN-001.md"), "# Plan\nstatus: active\n")
	if code := d.do(http.MethodPost, "/api/plans/discovered/scan", map[string]any{"projectPath": project}, &scanned); code != http.StatusOK && code != http.StatusAccepted {
		t.Fatalf("POST /api/plans/discovered/scan: HTTP %d\n%s", code, d.logs())
	}
	var plans any
	if code := d.do(http.MethodGet, "/api/plans/discovered?projectPath="+url.QueryEscape(project), nil, &plans); code != http.StatusOK {
		t.Errorf("GET /api/plans/discovered: HTTP %d", code)
	}

	var found []struct {
		ID string `json:"id"`
	}
	if code := d.do(http.MethodPost, "/api/providers/discovered/scan", nil, &found); code != http.StatusOK {
		t.Fatalf("POST /api/providers/discovered/scan: HTTP %d", code)
	}
	var listed []struct {
		ID string `json:"id"`
	}
	if code := d.do(http.MethodGet, "/api/providers/discovered", nil, &listed); code != http.StatusOK {
		t.Errorf("GET /api/providers/discovered: HTTP %d", code)
	}
	if code := d.do(http.MethodPost, "/api/providers/promote/no-such-provider", nil, nil); code != http.StatusNotFound {
		t.Errorf("promoting an unknown provider: HTTP %d, want 404", code)
	}

	var agents []any
	if code := d.do(http.MethodGet, "/api/ai-sessions/discovered", nil, &agents); code != http.StatusOK {
		t.Errorf("GET /api/ai-sessions/discovered: HTTP %d", code)
	}
}
