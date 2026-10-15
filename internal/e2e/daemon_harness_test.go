//go:build e2e

// Real-daemon end-to-end tests. They build the shipped binaries, start the
// daemon as a separate process against a throw-away database and a fake Ollama
// server, and drive it only through its public surfaces (REST, MCP, the stdio
// proxy and the two CLIs). Run with: go test -tags e2e ./internal/e2e/
package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	binDir    string
	buildOnce sync.Once
	buildErr  error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if binDir != "" {
		_ = os.RemoveAll(binDir)
	}
	os.Exit(code)
}

// repoRoot is the module root (the tests run from internal/e2e).
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

// buildBinaries compiles the four shipped commands once per test run.
func buildBinaries(t *testing.T) {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "nexus-e2e-bin-")
		if err != nil {
			buildErr = err
			return
		}
		binDir = dir
		env := os.Environ()
		if os.Getenv("CGO_CFLAGS") == "" {
			env = append(env, "CGO_CFLAGS=-DSQLITE_ENABLE_FTS5")
		}
		if os.Getenv("CGO_LDFLAGS") == "" {
			env = append(env, "CGO_LDFLAGS=-lm")
		}
		for _, name := range []string{"nexus-daemon", "nexus-cli", "nexus-mcp-stdio", "nexus-submit"} {
			out := filepath.Join(dir, name+exeSuffix())
			cmd := exec.Command("go", "build", "-o", out, "./cmd/"+name)
			cmd.Dir = repoRoot(t)
			cmd.Env = env
			if b, err := cmd.CombinedOutput(); err != nil {
				buildErr = fmt.Errorf("build %s: %w\n%s", name, err, b)
				return
			}
		}
	})
	if buildErr != nil {
		t.Fatal(buildErr)
	}
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func bin(name string) string { return filepath.Join(binDir, name+exeSuffix()) }

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

// fakeOllama speaks just enough of the Ollama API for the daemon to treat it as
// a live provider. chat and generate return whatever the test scripts.
type fakeOllama struct {
	*httptest.Server
	mu       sync.Mutex
	chatTurn int
	prompts  []string
	// reply answers both endpoints; generate and chat override it per endpoint.
	reply    func(prompt string) string
	generate func(prompt string) string
	chat     func(turn int, lastUser string) string
	// hold, when set, blocks a streamed chat reply after its first piece until it is closed,
	// so a test can look at the live output while the model is still "writing".
	hold chan struct{}
}

func newFakeOllama(t *testing.T) *fakeOllama {
	t.Helper()
	f := &fakeOllama{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"models":[{"name":"fake-coder"}]}`)
	})
	mux.HandleFunc("/api/show", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"model_info":{"llama.context_length":32768}}`)
	})
	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Prompt string `json:"prompt"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		f.mu.Lock()
		f.prompts = append(f.prompts, req.Prompt)
		fn := f.generate
		if fn == nil {
			fn = f.reply
		}
		f.mu.Unlock()
		out := "```go\npackage main\n```"
		if fn != nil {
			out = fn(req.Prompt)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"response": out})
	})
	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct{ Role, Content string } `json:"messages"`
			Stream   bool                             `json:"stream"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		last := ""
		if n := len(req.Messages); n > 0 {
			last = req.Messages[n-1].Content
		}
		f.mu.Lock()
		f.chatTurn++
		turn, fn, reply := f.chatTurn, f.chat, f.reply
		f.mu.Unlock()
		out := "```go\npackage main\n```"
		switch {
		case fn != nil:
			out = fn(turn, last)
		case reply != nil:
			out = reply(last)
		}
		if !req.Stream {
			_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"content": out}})
			return
		}
		// Like real Ollama: newline-delimited JSON, a few pieces, then a done line.
		enc, flush := json.NewEncoder(w), w.(http.Flusher).Flush
		half := len(out) / 2
		for i, piece := range []string{out[:half], out[half:]} {
			_ = enc.Encode(map[string]any{"message": map[string]string{"content": piece}, "done": false})
			flush()
			if i == 0 && f.hold != nil {
				select {
				case <-f.hold:
				case <-r.Context().Done():
					return
				}
			}
		}
		_ = enc.Encode(map[string]any{"message": map[string]string{"content": ""}, "done": true})
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// daemon is a running nexus-daemon process.
type daemon struct {
	t       *testing.T
	cmd     *exec.Cmd
	http    string // REST base URL
	mcp     string // MCP base URL
	db      string
	home    string
	llmURL  string
	exited  chan error
	logBuf  *bytes.Buffer
	logLock sync.Mutex
}

type daemonOpts struct {
	db, home, httpAddr, mcpAddr string
}

func startDaemon(t *testing.T, llm *fakeOllama, o daemonOpts) *daemon {
	t.Helper()
	buildBinaries(t)
	if o.db == "" {
		o.db = filepath.Join(t.TempDir(), "nexus.db")
	}
	if o.home == "" {
		o.home = t.TempDir()
	}
	if o.httpAddr == "" {
		o.httpAddr = freeAddr(t)
	}
	if o.mcpAddr == "" {
		o.mcpAddr = freeAddr(t)
	}
	d := &daemon{t: t, http: "http://" + o.httpAddr, mcp: "http://" + o.mcpAddr, db: o.db, home: o.home, exited: make(chan error, 1), logBuf: &bytes.Buffer{}}
	cmd := exec.Command(bin("nexus-daemon"))
	cmd.Env = append(os.Environ(),
		"HOME="+o.home, "USERPROFILE="+o.home, "XDG_CONFIG_HOME="+filepath.Join(o.home, ".config"),
		"NEXUS_DB_PATH="+o.db,
		"NEXUS_LISTEN_ADDR="+o.httpAddr,
		"NEXUS_MCP_ADDR="+o.mcpAddr,
		"NEXUS_LMSTUDIO_URL=http://127.0.0.1:1",
		"NEXUS_SCAN_INTERVAL=1h",
	)
	if llm != nil {
		cmd.Env = append(cmd.Env, "NEXUS_OLLAMA_URL="+llm.URL)
		d.llmURL = llm.URL
	} else {
		cmd.Env = append(cmd.Env, "NEXUS_OLLAMA_URL=http://127.0.0.1:1")
	}
	cmd.Stdout, cmd.Stderr = d, d
	if err := cmd.Start(); err != nil {
		t.Fatalf("start daemon: %v", err)
	}
	d.cmd = cmd
	go func() { d.exited <- cmd.Wait() }()
	t.Cleanup(func() { d.kill() })
	return d
}

// Write collects the daemon's output so a failing test can show it.
func (d *daemon) Write(p []byte) (int, error) {
	d.logLock.Lock()
	defer d.logLock.Unlock()
	return d.logBuf.Write(p)
}

func (d *daemon) logs() string {
	d.logLock.Lock()
	defer d.logLock.Unlock()
	return d.logBuf.String()
}

// waitHealthy blocks until /api/health answers or the process dies.
func (d *daemon) waitHealthy() {
	d.t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-d.exited:
			d.exited <- err
			d.t.Fatalf("daemon exited before becoming healthy: %v\n%s", err, d.logs())
		default:
		}
		if resp, err := http.Get(d.http + "/api/health"); err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	d.t.Fatalf("daemon never became healthy\n%s", d.logs())
}

// stop asks the daemon to shut down and returns how the process ended.
func (d *daemon) stop() error {
	d.t.Helper()
	if runtime.GOOS == "windows" {
		_ = d.cmd.Process.Kill()
	} else {
		_ = d.cmd.Process.Signal(sigterm())
	}
	select {
	case err := <-d.exited:
		d.exited <- err
		return err
	case <-time.After(20 * time.Second):
		d.t.Fatalf("daemon ignored SIGTERM\n%s", d.logs())
		return nil
	}
}

func (d *daemon) kill() {
	if d.cmd.ProcessState == nil {
		_ = d.cmd.Process.Kill()
	}
}

// get/post/put/del call the REST API and decode the JSON body into out (if non-nil).
func (d *daemon) do(method, path string, body any, out any) int {
	d.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, d.http+path, rd)
	if err != nil {
		d.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		d.t.Fatalf("%s %s: %v\n%s", method, path, err, d.logs())
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			d.t.Fatalf("%s %s: decode %q: %v", method, path, raw, err)
		}
	}
	return resp.StatusCode
}

type apiTask struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Logs   string `json:"logs"`
}

// waitStatus polls a task until it reaches want, failing early on another terminal state.
func (d *daemon) waitStatus(id string, want string) apiTask {
	d.t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	var t apiTask
	for time.Now().Before(deadline) {
		if code := d.do(http.MethodGet, "/api/tasks/"+id, nil, &t); code != http.StatusOK {
			d.t.Fatalf("get task %s: HTTP %d", id, code)
		}
		if t.Status == want {
			return t
		}
		switch t.Status {
		case "COMPLETED", "FAILED", "CANCELLED", "TOO_LARGE", "NO_PROVIDER":
			d.t.Fatalf("task %s ended %s, want %s\nlogs: %s\n%s", id, t.Status, want, t.Logs, d.logs())
		}
		time.Sleep(150 * time.Millisecond)
	}
	d.t.Fatalf("task %s stuck at %s, want %s\n%s", id, t.Status, want, d.logs())
	return t
}

// rpc posts one JSON-RPC request to the MCP endpoint.
func (d *daemon) rpc(method string, params any) (code int, resp map[string]any) {
	d.t.Helper()
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	r, err := http.Post(d.mcp+"/mcp", "application/json", bytes.NewReader(body))
	if err != nil {
		d.t.Fatalf("mcp %s: %v\n%s", method, err, d.logs())
	}
	defer r.Body.Close()
	raw, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(raw, &resp)
	return r.StatusCode, resp
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func contains(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
