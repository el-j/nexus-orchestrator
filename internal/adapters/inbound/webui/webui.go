// Package webui serves the Vue frontend from the daemon, so that "make start" gives the full UI
// at http://localhost:63987/app/ without the desktop app. `make web` builds the frontend into
// dist/ (embedded at compile time); with only the placeholder there, Embedded returns nil and the
// daemon keeps its small built-in dashboard at /ui.
package webui

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var dist embed.FS

// Embedded returns the handler for the frontend built into this binary, serving it under prefix
// (for example "/app/"), or nil when the binary was built without the frontend.
func Embedded(prefix string) http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil
	}
	return New(sub, prefix)
}

// csp lets the single-page app load its own scripts and styles and talk to this daemon; nothing
// else is allowed to load, except the web fonts the UI's stylesheet asks Google Fonts for.
// Inline styles are needed by the UI's component library.
const csp = "default-src 'self'; connect-src 'self'; img-src 'self' data:; " +
	"font-src 'self' data: https://fonts.gstatic.com; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; frame-ancestors 'none'"

// New serves the single-page app in site under prefix (which starts and ends with "/"): files
// that exist are served as they are, any other path without a file extension gets index.html
// (client-side routes, deep links), and a missing file is a 404. index.html gets <base href=prefix>
// so the relative asset paths of the build resolve whatever route the page is on. It returns nil
// when site has no index.html.
func New(site fs.FS, prefix string) http.Handler {
	index, err := fs.ReadFile(site, "index.html")
	if err != nil {
		return nil
	}
	index = bytes.Replace(index, []byte("<head>"), []byte(`<head><base href="`+prefix+`">`), 1)
	files := http.StripPrefix(strings.TrimSuffix(prefix, "/"), http.FileServerFS(site))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rel := strings.TrimPrefix(path.Clean("/"+strings.TrimPrefix(r.URL.Path, strings.TrimSuffix(prefix, "/"))), "/")
		if rel != "" && rel != "index.html" {
			if _, err := fs.Stat(site, rel); err == nil {
				if strings.HasPrefix(rel, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // hashed file names
				} else {
					w.Header().Set("Cache-Control", "public, max-age=3600")
				}
				files.ServeHTTP(w, r)
				return
			}
			if path.Ext(rel) != "" {
				http.NotFound(w, r)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		_, _ = w.Write(index)
	})
}
