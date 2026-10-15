package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
)

// With the frontend built into the daemon, the full UI is served under /app/ and the root goes
// there; without it the root keeps pointing at the built-in dashboard.
func TestFullUIIsServedUnderAppWhenAvailable(t *testing.T) {
	ui := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("UI:" + r.URL.Path)) })
	h := httpapi.NewServer(&spyOrch{}, nil, nil).WithUI(ui).Handler()

	for _, tc := range []struct{ path, want string }{{"/", "/app/"}, {"/app", "/app/"}} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != http.StatusFound || rec.Header().Get("Location") != tc.want {
			t.Errorf("%s: %d -> %q, want a redirect to %q", tc.path, rec.Code, rec.Header().Get("Location"), tc.want)
		}
	}
	for _, p := range []string{"/app/", "/app/providers", "/app/assets/x.js"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Body.String() != "UI:"+p {
			t.Errorf("%s: %q", p, rec.Body.String())
		}
	}
	// the built-in dashboard stays reachable as the light alternative, and the API is untouched
	if rec := httptest.NewRecorder(); func() bool { h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui", nil)); return rec.Code == 200 }() == false {
		t.Error("/ui must still work")
	}
}

func TestWithoutTheFullUITheRootStillGoesToTheDashboard(t *testing.T) {
	h := httpapi.NewServer(&spyOrch{}, nil, nil).Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("Location") != "/ui" {
		t.Errorf("root -> %q", rec.Header().Get("Location"))
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app/", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("/app/ without a UI: %d", rec.Code)
	}
}
