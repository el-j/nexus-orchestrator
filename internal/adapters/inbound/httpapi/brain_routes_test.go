package httpapi_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

type brainRoute struct {
	name    string
	method  string
	path    string
	body    string
	ok      int
	setErr  func(m *mockBrainService, err error)
	invalid []string // request bodies / paths (for GET) that must yield 400
	notFnd  int      // status when the service reports domain.ErrNotFound (0 = 500)
}

func brainRoutes() []brainRoute {
	return []brainRoute{
		{"ingest", "POST", "/api/brain/ingest", `{"projectPath":"/p","filePath":"/p/CLAUDE.md"}`, 200,
			func(m *mockBrainService, e error) { m.ingestFromFileErr = e }, []string{`{`, `{"projectPath":"/p"}`, `{"filePath":"/f"}`}, 0},
		{"status", "GET", "/api/brain/status?projectPath=/p", "", 200,
			func(m *mockBrainService, e error) { m.retStatusErr = e }, []string{"/api/brain/status"}, 0},
		{"context", "POST", "/api/brain/context", `{"projectPath":"/p"}`, 200,
			func(m *mockBrainService, e error) { m.retContextErr = e }, []string{`{`, `{}`}, 0},
		{"focused context", "POST", "/api/brain/focused-context", `{"projectPath":"/p","question":"q"}`, 200,
			func(m *mockBrainService, e error) { m.retFocusedContextErr = e }, []string{`{`, `{"projectPath":"/p"}`, `{"question":"q"}`}, 0},
		{"search", "GET", "/api/brain/search?projectPath=/p&q=x", "", 200,
			func(m *mockBrainService, e error) { m.retSearchErr = e }, []string{"/api/brain/search?projectPath=/p", "/api/brain/search?q=x"}, 0},
		{"init", "POST", "/api/brain/init", `{"projectPath":"/p"}`, 200,
			func(m *mockBrainService, e error) { m.retInitErr = e }, []string{`{`, `{}`}, 0},
		{"list", "GET", "/api/brain/knowledge?projectPath=/p&kind=learning", "", 200,
			func(m *mockBrainService, e error) { m.retListErr = e }, []string{"/api/brain/knowledge"}, 0},
		{"delete", "DELETE", "/api/brain/knowledge/k1", "", 204,
			func(m *mockBrainService, e error) { m.retDeleteErr = e }, nil, 404},
		{"file map", "GET", "/api/brain/file-map?projectPath=/p&focusArea=x", "", 200,
			func(m *mockBrainService, e error) { m.retFileMapErr = e }, []string{"/api/brain/file-map"}, 0},
		{"onboarding", "GET", "/api/brain/onboarding?projectPath=/p&maxTokens=100", "", 200,
			func(m *mockBrainService, e error) { m.retOnboardingErr = e }, []string{"/api/brain/onboarding"}, 0},
	}
}

func TestBrainRoutes(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	for _, rt := range brainRoutes() {
		t.Run(rt.name, func(t *testing.T) {
			// success
			rec := send(realHandler(newFailOrch(), &mockBrainService{}), rt.method, rt.path, rt.body)
			if rec.Code != rt.ok {
				t.Fatalf("success: %d %s", rec.Code, rec.Body)
			}
			// brain not configured
			rec = send(realHandler(newFailOrch(), nil), rt.method, rt.path, rt.body)
			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("no brain service: %d", rec.Code)
			}
			// validation
			for _, bad := range rt.invalid {
				var rec2 = rec
				if rt.method == "GET" {
					rec2 = send(realHandler(newFailOrch(), &mockBrainService{}), rt.method, bad, "")
				} else {
					rec2 = send(realHandler(newFailOrch(), &mockBrainService{}), rt.method, rt.path, bad)
				}
				if rec2.Code != http.StatusBadRequest {
					t.Errorf("invalid input %q: %d, want 400", bad, rec2.Code)
				}
			}
			// service failure
			m := &mockBrainService{}
			rt.setErr(m, errors.New("boom"))
			if rec = send(realHandler(newFailOrch(), m), rt.method, rt.path, rt.body); rec.Code != http.StatusInternalServerError {
				t.Errorf("service failure: %d", rec.Code)
			}
			// not found
			if rt.notFnd != 0 {
				m = &mockBrainService{}
				rt.setErr(m, domain.ErrNotFound)
				if rec = send(realHandler(newFailOrch(), m), rt.method, rt.path, rt.body); rec.Code != rt.notFnd {
					t.Errorf("not found: %d, want %d", rec.Code, rt.notFnd)
				}
			}
		})
	}
}

func TestBrainRoutes_DefaultsAndEmptyResults(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	h := realHandler(newFailOrch(), &mockBrainService{})

	// A search limit that is not a positive integer falls back to the default.
	for _, q := range []string{"&limit=abc", "&limit=-3", "&limit=0", "&limit=9"} {
		if rec := send(h, "GET", "/api/brain/search?projectPath=/p&q=x"+q, ""); rec.Code != 200 {
			t.Errorf("limit %q: %d", q, rec.Code)
		}
	}
	// Likewise maxTokens on onboarding and context.
	for _, q := range []string{"&maxTokens=abc", "&maxTokens=-1", ""} {
		rec := send(h, "GET", "/api/brain/onboarding?projectPath=/p"+q, "")
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"maxTokens":800`) {
			t.Errorf("maxTokens %q: %d %s", q, rec.Code, rec.Body)
		}
	}
	// Empty results are encoded as [] / {"filePaths":[]}, never null.
	if rec := send(h, "GET", "/api/brain/knowledge?projectPath=/p", ""); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("list: %q", rec.Body)
	}
	if rec := send(h, "GET", "/api/brain/file-map?projectPath=/p", ""); !strings.Contains(rec.Body.String(), `"filePaths":[]`) {
		t.Errorf("file map: %q", rec.Body)
	}
	if rec := send(h, "POST", "/api/brain/context", `{"projectPath":"/p","maxTokens":-5}`); rec.Code != 200 {
		t.Errorf("context with a non-positive budget: %d", rec.Code)
	}
}
