package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

var site = fstest.MapFS{
	"index.html":          {Data: []byte("<!doctype html><html><head><title>Nexus</title></head><body><div id=app></div></body></html>")},
	"favicon.svg":         {Data: []byte("<svg/>")},
	"assets/index-abc.js": {Data: []byte("console.log(1)")},
}

// reply is what a handler answered: the status, headers and body.
type reply struct {
	StatusCode int
	Header     http.Header
}

func get(t *testing.T, h http.Handler, path string) (reply, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return reply{rec.Code, rec.Header()}, rec.Body.String()
}

func TestHandler_ServesTheAppUnderItsPrefixWithABaseTag(t *testing.T) {
	h := New(site, "/app/")
	for _, path := range []string{"/app/", "/app/index.html", "/app/providers", "/app/tasks/123/deep"} {
		res, body := get(t, h, path)
		if res.StatusCode != 200 || !strings.Contains(body, `<base href="/app/">`) || !strings.Contains(body, "<div id=app>") {
			t.Errorf("%s: %d %q", path, res.StatusCode, body)
		}
		if !strings.HasPrefix(res.Header.Get("Content-Type"), "text/html") || res.Header.Get("Cache-Control") != "no-cache" {
			t.Errorf("%s: headers %v", path, res.Header)
		}
		if csp := res.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'self'") || !strings.Contains(csp, "connect-src 'self'") {
			t.Errorf("%s: CSP %q", path, csp)
		}
	}
	// the base tag goes into <head> exactly once
	_, body := get(t, h, "/app/")
	if strings.Count(body, "<base ") != 1 || strings.Index(body, "<base") > strings.Index(body, "<title>") {
		t.Errorf("base tag misplaced: %s", body)
	}
}

func TestHandler_ServesFilesAndNeverFallsBackToTheAppForAMissingFile(t *testing.T) {
	h := New(site, "/app/")
	res, body := get(t, h, "/app/assets/index-abc.js")
	if res.StatusCode != 200 || body != "console.log(1)" || !strings.Contains(res.Header.Get("Cache-Control"), "immutable") {
		t.Errorf("asset: %d %q %v", res.StatusCode, body, res.Header)
	}
	if res, _ := get(t, h, "/app/favicon.svg"); res.StatusCode != 200 || res.Header.Get("Cache-Control") == "" {
		t.Errorf("favicon: %d %v", res.StatusCode, res.Header)
	}
	// A missing asset is a 404, not the app (a browser would try to run HTML as a script).
	for _, path := range []string{"/app/assets/missing.js", "/app/nope.css", "/app/x.map"} {
		if res, _ := get(t, h, path); res.StatusCode != 404 {
			t.Errorf("%s: %d, want 404", path, res.StatusCode)
		}
	}
}

func TestHandler_OnlyAnswersReads(t *testing.T) {
	rec := httptest.NewRecorder()
	New(site, "/app/").ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/app/", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST: %d", rec.Code)
	}
}

func TestEmbedded_IsNilUntilTheFrontendIsBuilt(t *testing.T) {
	// The repository ships dist/ with only a .gitkeep; `make web` fills it.
	if h := Embedded("/app/"); h != nil {
		t.Skip("a built frontend is embedded")
	}
}

func TestNew_WithoutAnIndexIsNil(t *testing.T) {
	if h := New(fstest.MapFS{"x.js": {Data: []byte("1")}}, "/app/"); h != nil {
		t.Error("a site without index.html must not be served")
	}
}
