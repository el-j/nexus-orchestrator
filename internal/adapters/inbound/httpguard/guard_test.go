package httpguard

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOriginAllowed(t *testing.T) {
	t.Setenv("NEXUS_ALLOWED_ORIGINS", "")
	p := New("127.0.0.1:63987")
	allowed := []string{
		"", "http://localhost", "http://localhost:5173", "https://localhost:8443",
		"http://127.0.0.1:63987", "http://[::1]:3000", "wails://wails.localhost",
		"http://wails.localhost", "HTTP://LOCALHOST:1",
	}
	for _, o := range allowed {
		if !p.OriginAllowed(o) {
			t.Errorf("origin %q should be allowed", o)
		}
	}
	denied := []string{
		"https://evil.example", "http://localhost.evil.example", "http://127.0.0.1.evil.example",
		"null", "file://", "ftp://localhost", "http://", "not a url", "http://192.168.1.5:80",
		"javascript:alert(1)", "http://localhost@evil.example",
	}
	for _, o := range denied {
		if p.OriginAllowed(o) {
			t.Errorf("origin %q must be rejected", o)
		}
	}
}

func TestOriginAllowed_EnvironmentExtras(t *testing.T) {
	t.Setenv("NEXUS_ALLOWED_ORIGINS", " https://dash.example.com/ , http://tool.internal:9000 ")
	p := New("127.0.0.1:1")
	for _, o := range []string{"https://dash.example.com", "http://tool.internal:9000", "HTTPS://DASH.EXAMPLE.COM/"} {
		if !p.OriginAllowed(o) {
			t.Errorf("configured origin %q should be allowed", o)
		}
	}
	if p.OriginAllowed("https://other.example.com") {
		t.Error("unlisted origin must stay rejected")
	}
}

func TestHostAllowed_LoopbackBindingRejectsRebindingHostnames(t *testing.T) {
	t.Setenv("NEXUS_ALLOWED_HOSTS", "nexus.lan, Dev-Box")
	p := New("127.0.0.1:63987")
	for _, h := range []string{"localhost", "localhost:63987", "127.0.0.1", "127.0.0.1:63987", "[::1]:63987", "::1", "10.0.0.5:63987", "nexus.lan", "dev-box:80", "LOCALHOST"} {
		if !p.HostAllowed(h) {
			t.Errorf("host %q should be allowed", h)
		}
	}
	for _, h := range []string{"attacker.example", "attacker.example:63987", "127.0.0.1.nip.io", "localhost.evil.example", ""} {
		if p.HostAllowed(h) {
			t.Errorf("host %q must be rejected on a loopback-bound server", h)
		}
	}
}

func TestHostCheckIsSkippedWhenBoundToAllInterfaces(t *testing.T) {
	t.Setenv("NEXUS_ALLOWED_HOSTS", "")
	for _, addr := range []string{"0.0.0.0:63987", ":63987", "192.168.1.10:80", "[::]:80"} {
		if !New(addr).HostAllowed("nexus-service:63987") {
			t.Errorf("addr %q: container/LAN hostnames must pass when the operator exposed the server", addr)
		}
	}
	// But origins are always checked.
	if New("0.0.0.0:1").OriginAllowed("https://evil.example") {
		t.Error("foreign origins are rejected regardless of the bind address")
	}
	for _, addr := range []string{"localhost:1", "[::1]:1", "127.0.0.2:1", "localhost"} {
		if New(addr).HostAllowed("attacker.example") {
			t.Errorf("addr %q is loopback: hostname hosts must be rejected", addr)
		}
	}
}

func TestMiddleware(t *testing.T) {
	t.Setenv("NEXUS_ALLOWED_ORIGINS", "")
	t.Setenv("NEXUS_ALLOWED_HOSTS", "")
	reached := 0
	h := New("127.0.0.1:63987").Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached++
		w.WriteHeader(http.StatusNoContent)
	}))
	do := func(host, origin string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/tasks", nil)
		req.Host = host
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if c := do("127.0.0.1:63987", ""); c != http.StatusNoContent {
		t.Errorf("plain local client: %d", c)
	}
	if c := do("localhost:63987", "http://localhost:5173"); c != http.StatusNoContent {
		t.Errorf("local browser app: %d", c)
	}
	if c := do("127.0.0.1:63987", "https://evil.example"); c != http.StatusForbidden {
		t.Errorf("drive-by cross-site POST must be forbidden, got %d", c)
	}
	if c := do("attacker.example:63987", "http://attacker.example:63987"); c != http.StatusForbidden {
		t.Errorf("DNS rebinding must be forbidden, got %d", c)
	}
	if c := do("attacker.example:63987", ""); c != http.StatusForbidden {
		t.Errorf("rebinding without an Origin header (same-origin GET) must be forbidden, got %d", c)
	}
	if reached != 2 {
		t.Errorf("handler reached %d times, want 2", reached)
	}
}

func TestNilPolicyAllowsEverything(t *testing.T) {
	var p *Policy
	if !p.HostAllowed("anything") || !p.OriginAllowed("") {
		t.Error("a nil policy must not block")
	}
	if p.OriginAllowed("https://evil.example") {
		t.Error("even a nil policy must not accept a foreign origin")
	}
}
