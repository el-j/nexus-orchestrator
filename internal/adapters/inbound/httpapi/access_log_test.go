package httpapi

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// capture runs one request through accessLog and returns what was logged.
func capture(t *testing.T, method, path string, status int, took time.Duration) string {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	h := accessLog(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(took)
		w.WriteHeader(status)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, path, nil))
	return buf.String()
}

func TestAccessLog_SkipsRoutinePollingButKeepsWhatMatters(t *testing.T) {
	for _, tc := range []struct {
		name         string
		method, path string
		status       int
		took         time.Duration
		logged       bool
	}{
		{"a poll that succeeds", "GET", "/api/tasks", 200, 0, false},
		{"a health probe", "GET", "/api/health", 200, 0, false},
		{"a session heartbeat", "POST", "/api/ai-sessions/abc/heartbeat", 204, 0, false},
		{"a change", "POST", "/api/tasks", 201, 0, true},
		{"a delete", "DELETE", "/api/tasks/1", 204, 0, true},
		{"a failed read", "GET", "/api/tasks/x", 404, 0, true},
		{"a server error", "GET", "/api/tasks", 500, 0, true},
		{"a slow read", "GET", "/api/tasks", 200, slowRequest + 50*time.Millisecond, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := capture(t, tc.method, tc.path, tc.status, tc.took)
			if (out != "") != tc.logged {
				t.Fatalf("logged = %v, want %v (%q)", out != "", tc.logged, out)
			}
			if tc.logged && !strings.Contains(out, tc.method+" "+tc.path) {
				t.Errorf("line lacks method and path: %q", out)
			}
		})
	}
}

func TestAccessLog_KeepsTheResponseIntact(t *testing.T) {
	rec := httptest.NewRecorder()
	accessLog(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("body"))
	})).ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Code != 200 || rec.Body.String() != "body" {
		t.Errorf("%d %q", rec.Code, rec.Body.String())
	}
}
