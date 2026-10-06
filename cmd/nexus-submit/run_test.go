package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeDaemon serves POST /api/tasks (201) and GET /api/tasks/{id}, returning
// each status in statuses in turn (the last one repeats).
func fakeDaemon(t *testing.T, statuses []string, logs string) *httptest.Server {
	t.Helper()
	var polls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/tasks":
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"task_id":"T-1","status":"QUEUED"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/tasks/T-1":
			i := int(polls.Add(1)) - 1
			if i >= len(statuses) {
				i = len(statuses) - 1
			}
			fmt.Fprintf(w, `{"status":%q,"logs":%q}`, statuses[i], logs)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func taskFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "TASK-1.md")
	if err := os.WriteFile(p, []byte("do the thing"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func runCLI(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = run(args, &out, &errb, time.Millisecond)
	return code, out.String(), errb.String()
}

func TestRun_Version(t *testing.T) {
	code, out, _ := runCLI("--version")
	if code != exitOK || !strings.Contains(out, "nexus-submit dev") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestRun_HelpExitsZero(t *testing.T) {
	code, _, errOut := runCLI("-h")
	if code != exitOK || !strings.Contains(errOut, "task-file") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestRun_BadFlagIsFailure(t *testing.T) {
	if code, _, _ := runCLI("--no-such-flag"); code != exitFailure {
		t.Fatalf("code=%d", code)
	}
}

func TestRun_RequiresTaskFile(t *testing.T) {
	code, _, errOut := runCLI()
	if code != exitFailure || !strings.Contains(errOut, "--task-file is required") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestRun_UnreadableTaskFile(t *testing.T) {
	code, _, errOut := runCLI("--task-file", "/no/such/file.md")
	if code != exitFailure || !strings.Contains(errOut, "read task file") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestRun_DaemonUnreachable(t *testing.T) {
	code, _, errOut := runCLI("--task-file", taskFile(t), "--addr", "http://127.0.0.1:1")
	if code != exitFailure || !strings.Contains(errOut, "POST /api/tasks") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestRun_SubmitWithoutWait(t *testing.T) {
	srv := fakeDaemon(t, []string{"QUEUED"}, "")
	code, out, _ := runCLI("--task-file", taskFile(t), "--addr", srv.URL, "--project", "/p",
		"--target", "a.go", "--context", "x.go", "--verify", "go test ./...")
	if code != exitOK {
		t.Fatalf("code=%d", code)
	}
	for _, want := range []string{"submitted: task_id=T-1 status=QUEUED", "track: " + srv.URL + "/api/tasks/T-1", "ui:"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout missing %q:\n%s", want, out)
		}
	}
}

func TestRun_NEXUS_ADDREnvIsTheDefaultAddress(t *testing.T) {
	srv := fakeDaemon(t, []string{"QUEUED"}, "")
	t.Setenv("NEXUS_ADDR", srv.URL)
	if code, out, _ := runCLI("--task-file", taskFile(t)); code != exitOK || !strings.Contains(out, "task_id=T-1") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestRun_WaitExitCodesPerTerminalStatus(t *testing.T) {
	cases := []struct {
		status   string
		wantCode int
		wantOut  string
		wantErr  string
	}{
		{"COMPLETED", exitOK, "status=COMPLETED", ""},
		{"FAILED", exitFailure, "logs: boom", ""},
		{"CANCELLED", exitFailure, "", "Task cancelled"},
		{"TOO_LARGE", exitTooLarge, "", "exceeds model context limit"},
		{"NO_PROVIDER", exitNoProvider, "", "no LLM provider available"},
	}
	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			srv := fakeDaemon(t, []string{"PROCESSING", tc.status}, "boom")
			code, out, errOut := runCLI("--task-file", taskFile(t), "--addr", srv.URL, "--wait", "--timeout", "5s")
			if code != tc.wantCode {
				t.Fatalf("code=%d want %d (stdout=%q stderr=%q)", code, tc.wantCode, out, errOut)
			}
			if !strings.Contains(out, tc.wantOut) || !strings.Contains(errOut, tc.wantErr) {
				t.Errorf("stdout=%q stderr=%q", out, errOut)
			}
		})
	}
}

func TestRun_WaitCompletedWithoutLogsPrintsNoLogsLine(t *testing.T) {
	srv := fakeDaemon(t, []string{"COMPLETED"}, "")
	_, out, _ := runCLI("--task-file", taskFile(t), "--addr", srv.URL, "--wait")
	if strings.Contains(out, "logs:") {
		t.Errorf("unexpected logs line: %q", out)
	}
}

func TestRun_WaitTimesOut(t *testing.T) {
	srv := fakeDaemon(t, []string{"PROCESSING"}, "")
	code, _, errOut := runCLI("--task-file", taskFile(t), "--addr", srv.URL, "--wait", "--timeout", "30ms")
	if code != exitFailure || !strings.Contains(errOut, "timed out") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestWaitForCompletion_ReportsPollErrorsThenTimesOut(t *testing.T) {
	var out, errb bytes.Buffer
	code := waitForCompletion(&out, &errb, "http://127.0.0.1:1", "T", 30*time.Millisecond, time.Millisecond)
	if code != exitFailure || !strings.Contains(errb.String(), "poll error") {
		t.Fatalf("code=%d stderr=%q", code, errb.String())
	}
}

func TestSubmitTask_ErrorBranches(t *testing.T) {
	status := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer status.Close()
	if _, _, err := SubmitTask(status.URL, map[string]interface{}{"a": 1}); err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("non-201: got %v", err)
	}

	badJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, "not json")
	}))
	defer badJSON.Close()
	if _, _, err := SubmitTask(badJSON.URL, map[string]interface{}{}); err == nil || !strings.Contains(err.Error(), "decode response") {
		t.Errorf("bad json: got %v", err)
	}

	if _, _, err := SubmitTask("http://127.0.0.1:1", map[string]interface{}{"ch": make(chan int)}); err == nil || !strings.Contains(err.Error(), "marshal request") {
		t.Errorf("unmarshalable body: got %v", err)
	}
}

func TestBuildRequestBody_DefaultsProjectToCwdAndSkipsEmptyContext(t *testing.T) {
	body, err := BuildRequestBody(taskFile(t), "", "", " a.go , ,b.go ", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	if body["projectPath"] != cwd {
		t.Errorf("projectPath = %v, want cwd %s", body["projectPath"], cwd)
	}
	if got := body["contextFiles"].([]string); len(got) != 2 || got[0] != "a.go" || got[1] != "b.go" {
		t.Errorf("contextFiles = %v", got)
	}
	if _, ok := body["verificationCommand"]; ok {
		t.Error("verificationCommand must be omitted when --verify is empty")
	}
}
