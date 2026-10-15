package sys_scanner

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// serverPort starts an httptest server and returns its loopback port.
func serverPort(t *testing.T, h http.HandlerFunc) int {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	_, p, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(p)
	return port
}

func newTestScanner() *Scanner { return &Scanner{httpClient: &http.Client{Timeout: 2 * time.Second}} }

func TestProbePort_ReportsOnlyRealModelServers(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		want    bool
		models  []string
	}{
		{"openai-compatible with models", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `{"data":[{"id":"llama-3"},{"id":""},{"id":"qwen"}]}`)
		}, true, []string{"llama-3", "qwen"}},
		{"openai-compatible with no model loaded", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `{"data":[]}`)
		}, true, nil},
		{"ollama tags", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `{"models":[{"name":"llama3:8b"},{"name":""}]}`)
		}, true, []string{"llama3:8b"}},
		{"AirPlay-style 403", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "forbidden", http.StatusForbidden)
		}, false, nil},
		{"dev server 404 page", http.NotFound, false, nil},
		{"HTML 200 page", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, "<html><body>hello</body></html>")
		}, false, nil},
		{"unrelated JSON 200", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `{"status":"ok"}`)
		}, false, nil},
		{"server error", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			port := serverPort(t, tc.handler)
			got, err := newTestScanner().probePort(context.Background(),
				portTarget{name: "LM Studio", endpoint: "/v1/models", port: port, kind: domain.ProviderKindLMStudio})
			if err != nil {
				t.Fatal(err)
			}
			if !tc.want {
				if len(got) != 0 {
					t.Fatalf("a non-LLM service must not be reported as a provider: %+v", got)
				}
				return
			}
			if len(got) != 1 || got[0].Status != domain.DiscoveryStatusReachable || got[0].ID != fmt.Sprintf("port-%d", port) {
				t.Fatalf("got %+v", got)
			}
			if strings.Join(got[0].Models, ",") != strings.Join(tc.models, ",") {
				t.Errorf("models = %v, want %v", got[0].Models, tc.models)
			}
		})
	}
}

func TestProbePort_ClosedPortAndCancelledContext(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	if got, err := newTestScanner().probePort(context.Background(), portTarget{name: "x", endpoint: "/", port: port}); err != nil || got != nil {
		t.Errorf("closed port: %v %v", got, err)
	}
}

func TestProbePort_OllamaAlsoReportsLoadedModels(t *testing.T) {
	port := serverPort(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			fmt.Fprint(w, `{"models":[{"name":"llama3"}]}`)
		case "/api/ps":
			fmt.Fprint(w, `{"models":[{"name":"llama3"},{"name":"","model":"qwen"},{"name":"","model":""}]}`)
		}
	})
	got, err := newTestScanner().probePort(context.Background(),
		portTarget{name: "Ollama", endpoint: "/api/tags", port: port, kind: domain.ProviderKindOllama})
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
	if !got[0].Generating || strings.Join(got[0].ActiveModels, ",") != "llama3,qwen" {
		t.Errorf("loaded models not reported: %+v", got[0])
	}
}

func TestProbeOllamaPS_FailureModes(t *testing.T) {
	s := newTestScanner()
	for name, h := range map[string]http.HandlerFunc{
		"500":      func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "x", 500) },
		"bad json": func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "nope") },
		"idle":     func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `{"models":[]}`) },
	} {
		active, generating := s.probeOllamaPS(context.Background(), serverPort(t, h))
		if len(active) != 0 || generating {
			t.Errorf("%s: active=%v generating=%v", name, active, generating)
		}
	}
	// Nothing listening.
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	if a, g := s.probeOllamaPS(context.Background(), port); len(a) != 0 || g {
		t.Error("unreachable ollama must report nothing")
	}
}

func TestMergeProvidersAndStatusRank(t *testing.T) {
	cli := domain.DiscoveredProvider{ID: "cli-ollama", Name: "Ollama", Status: domain.DiscoveryStatusInstalled, CLIPath: "/usr/bin/ollama"}
	proc := domain.DiscoveredProvider{ID: "proc-ollama", Name: "Ollama", Status: domain.DiscoveryStatusRunning, ProcessName: "ollama"}
	port := domain.DiscoveredProvider{ID: "port-11434", Name: "Ollama", Status: domain.DiscoveryStatusReachable, BaseURL: "http://127.0.0.1:11434", Models: []string{"m"}}

	merged := mergeProviders(mergeProviders(cli, proc), port)
	if merged.Status != domain.DiscoveryStatusReachable || merged.ID != "port-11434" || merged.BaseURL == "" ||
		merged.CLIPath != "/usr/bin/ollama" || merged.ProcessName != "ollama" || len(merged.Models) != 1 {
		t.Errorf("merge lost information: %+v", merged)
	}
	// The weaker status never downgrades, and an existing port id is kept.
	back := mergeProviders(port, cli)
	if back.Status != domain.DiscoveryStatusReachable || back.ID != "port-11434" || back.CLIPath == "" {
		t.Errorf("reverse merge: %+v", back)
	}
	if statusRank(domain.DiscoveryStatusReachable) <= statusRank(domain.DiscoveryStatusRunning) ||
		statusRank(domain.DiscoveryStatusRunning) <= statusRank(domain.DiscoveryStatusInstalled) ||
		statusRank(domain.DiscoveryStatusInstalled) <= statusRank("unknown") {
		t.Error("status ranks must be strictly ordered reachable > running > installed > unknown")
	}
}

func TestParseModels_Shapes(t *testing.T) {
	cases := map[string]string{
		`{"data":[{"id":"a"},{"id":"b"}]}`:                    "a,b",
		`{"models":[{"name":"x"}]}`:                           "x",
		`{"data":[{"id":""}],"models":[{"name":"fallback"}]}`: "fallback",
		`{"data":[]}`: "",
		`not json`:    "",
		``:            "",
	}
	for in, want := range cases {
		if got := strings.Join(parseModels([]byte(in)), ","); got != want {
			t.Errorf("parseModels(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParsePgrepOutput(t *testing.T) {
	cases := []struct {
		out     string
		pattern string
		found   bool
		name    string
		pid     int
	}{
		{"", "Claude", false, "", 0},
		{"  \n", "Claude", false, "", 0},
		{"1234 /Applications/Claude.app/Contents/MacOS/Claude --flag", "Claude", true, "/Applications/Claude.app/Contents/MacOS/Claude", 1234},
		{"99 claude", "Claude", true, "claude", 99},
		{"5678", "Claude", true, "Claude", 0}, // no name column: fall back to the pattern
		// Several matches: only the first record is used (the second PID must not leak into the name).
		{"11 /usr/bin/claude --x\n22 /usr/bin/claude-helper --y\n", "claude", true, "/usr/bin/claude", 11},
	}
	for _, tc := range cases {
		found, name, pid := parsePgrep(tc.out, tc.pattern)
		if found != tc.found || name != tc.name || pid != tc.pid {
			t.Errorf("parsePgrep(%q) = %v %q %d, want %v %q %d", tc.out, found, name, pid, tc.found, tc.name, tc.pid)
		}
	}
}

func TestParseTasklistOutput(t *testing.T) {
	out := "\"System\",\"4\",\"Services\",\"0\",\"148 K\"\r\n" +
		"\"Claude.exe\",\"4321\",\"Console\",\"1\",\"200,000 K\"\r\n" +
		"\"chrome.exe\",\"77\",\"Console\",\"1\",\"90 K\"\r\n"
	found, name, pid := parseTasklist(out, "claude")
	if !found || name != "Claude.exe" || pid != 4321 {
		t.Errorf("got %v %q %d", found, name, pid)
	}
	if found, _, _ := parseTasklist(out, "nothing-here"); found {
		t.Error("an absent process must not be found")
	}
	// A matching line that is not quoted CSV falls back to the pattern name.
	found, name, pid = parseTasklist("Claude running pid 5\n", "claude")
	if !found || name != "claude" || pid != 0 {
		t.Errorf("unquoted line: %v %q %d", found, name, pid)
	}
}
