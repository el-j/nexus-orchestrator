package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
)

// newHowtoServer creates a real httpapi.Server wired to the full Handler() router.
func newHowtoServer() *httptest.Server {
	hub := httpapi.NewHub()
	srv := httpapi.NewServer(&mockOrchestrator{}, nil, hub)
	return httptest.NewServer(srv.Handler())
}

// ---------------------------------------------------------------------------
// GET /api/howto
// ---------------------------------------------------------------------------

func TestHowto_ReturnsJSON(t *testing.T) {
	ts := newHowtoServer()
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/api/howto")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected application/json, got %q", ct)
	}
}

func TestHowto_StructureIsComplete(t *testing.T) {
	ts := newHowtoServer()
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/api/howto")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	var doc map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, f := range []string{"name", "version", "description", "quick_start", "connection", "ai_workflows", "http_endpoints", "examples"} {
		if _, ok := doc[f]; !ok {
			t.Errorf("missing field %q", f)
		}
	}
	if doc["name"] != "nexusOrchestrator" {
		t.Errorf("name = %v, want nexusOrchestrator", doc["name"])
	}
}

func TestHowto_ConnectionURLsUseLiveHost(t *testing.T) {
	ts := newHowtoServer()
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/api/howto")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	var doc struct {
		Connection struct {
			HTTPAPI   string `json:"http_api"`
			Dashboard string `json:"dashboard"`
			Discovery string `json:"discovery"`
		} `json:"connection"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(doc.Connection.HTTPAPI, ts.URL) {
		t.Errorf("http_api %q does not start with %q", doc.Connection.HTTPAPI, ts.URL)
	}
	if !strings.HasSuffix(doc.Connection.Dashboard, "/ui") {
		t.Errorf("dashboard %q should end in /ui", doc.Connection.Dashboard)
	}
	if !strings.HasSuffix(doc.Connection.Discovery, "/.well-known/nexus.json") {
		t.Errorf("discovery %q should end in /.well-known/nexus.json", doc.Connection.Discovery)
	}
}

func TestHowto_EndpointsDocumentsSelf(t *testing.T) {
	ts := newHowtoServer()
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/api/howto")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	var doc struct {
		Endpoints []struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"http_endpoints"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(doc.Endpoints) == 0 {
		t.Fatal("http_endpoints must not be empty")
	}
	found := false
	for _, ep := range doc.Endpoints {
		if ep.Method == "GET" && ep.Path == "/api/howto" {
			found = true
			break
		}
	}
	if !found {
		t.Error("howto should document itself in http_endpoints")
	}
}

// ---------------------------------------------------------------------------
// GET /.well-known/nexus.json
// ---------------------------------------------------------------------------

func TestWellKnownNexus_ReturnsJSON(t *testing.T) {
	ts := newHowtoServer()
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/.well-known/nexus.json")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected application/json, got %q", ct)
	}
}

func TestWellKnownNexus_DocStructure(t *testing.T) {
	ts := newHowtoServer()
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/.well-known/nexus.json")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	var doc map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, f := range []string{"schema_version", "name", "description", "api", "mcp", "capabilities"} {
		if _, ok := doc[f]; !ok {
			t.Errorf("missing field %q", f)
		}
	}
	if doc["name"] != "nexusOrchestrator" {
		t.Errorf("name = %v, want nexusOrchestrator", doc["name"])
	}
	if doc["schema_version"] != "1" {
		t.Errorf("schema_version = %v, want 1", doc["schema_version"])
	}
}

func TestWellKnownNexus_APILinksUseLiveHost(t *testing.T) {
	ts := newHowtoServer()
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/.well-known/nexus.json")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	var doc struct {
		API struct {
			BaseURL string `json:"base_url"`
			HowTo   string `json:"howto"`
			Health  string `json:"health"`
		} `json:"api"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(doc.API.BaseURL, ts.URL) {
		t.Errorf("api.base_url %q does not start with %q", doc.API.BaseURL, ts.URL)
	}
	if !strings.HasSuffix(doc.API.HowTo, "/api/howto") {
		t.Errorf("api.howto %q should end in /api/howto", doc.API.HowTo)
	}
	if !strings.HasSuffix(doc.API.Health, "/api/health") {
		t.Errorf("api.health %q should end in /api/health", doc.API.Health)
	}
}

func TestWellKnownNexus_CapabilitiesContainTaskQueue(t *testing.T) {
	ts := newHowtoServer()
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/.well-known/nexus.json")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	var doc struct {
		Capabilities []string `json:"capabilities"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, c := range doc.Capabilities {
		if c == "task-queue" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("capabilities should contain task-queue, got %v", doc.Capabilities)
	}
}

// ---------------------------------------------------------------------------
// MCP endpoint advertised by /.well-known/nexus.json
// ---------------------------------------------------------------------------

func mcpEndpointFor(t *testing.T, mcpAddr string) string {
	t.Helper()
	srv := httpapi.NewServer(&mockOrchestrator{}, nil, httpapi.NewHub()).WithMCPAddr(mcpAddr)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/.well-known/nexus.json")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	var doc struct {
		MCP struct {
			Endpoint string `json:"endpoint"`
		} `json:"mcp"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return doc.MCP.Endpoint
}

func TestWellKnownNexus_AdvertisesConfiguredMCPEndpoint(t *testing.T) {
	if got := mcpEndpointFor(t, "10.1.2.3:7000"); got != "http://10.1.2.3:7000/mcp" {
		t.Errorf("endpoint = %q, want http://10.1.2.3:7000/mcp", got)
	}
}

func TestWellKnownNexus_HostlessMCPAddrUsesRequestHost(t *testing.T) {
	// httptest serves on 127.0.0.1, so a wildcard bind must be reported on that host.
	for _, addr := range []string{":63988", "0.0.0.0:63988", "[::]:63988"} {
		if got := mcpEndpointFor(t, addr); got != "http://127.0.0.1:63988/mcp" {
			t.Errorf("addr %q: endpoint = %q, want http://127.0.0.1:63988/mcp", addr, got)
		}
	}
}

func TestWellKnownNexus_OmitsEndpointWhenUnknownOrInvalid(t *testing.T) {
	for _, addr := range []string{"", "not-an-address"} {
		if got := mcpEndpointFor(t, addr); got != "" {
			t.Errorf("addr %q: endpoint = %q, want it omitted", addr, got)
		}
	}
}
