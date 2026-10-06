// Package httpguard protects the daemon's local HTTP surfaces (REST API and MCP
// server) from requests that a web page in the user's browser could make on the
// user's behalf.
//
// A server bound to 127.0.0.1 is still reachable from any website the user
// visits: a cross-site form or no-cors fetch reaches the handler even though
// CORS stops the page from reading the answer, and DNS rebinding makes a
// hostile origin resolve to 127.0.0.1. Without an API token the daemon would
// execute such requests (submit tasks, register providers, terminate sessions).
// The Policy here closes both paths:
//
//   - Origin: browsers always send an Origin header on cross-origin writes. A
//     request carrying an Origin that is not a local origin is rejected.
//     Non-browser clients (CLI, MCP stdio proxy, curl) send none and are unaffected.
//   - Host: when the server listens on a loopback address only, the Host header
//     must name localhost or an IP literal. A rebinding attacker's own hostname
//     is rejected. Servers deliberately bound to all interfaces (containers,
//     LAN) skip this check; operators there should set NEXUS_API_TOKEN.
//
// Escape hatches: NEXUS_ALLOWED_ORIGINS and NEXUS_ALLOWED_HOSTS take
// comma-separated lists of extra origins (scheme://host[:port]) and host names.
package httpguard

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Policy decides which Origin and Host headers may reach a local server.
type Policy struct {
	checkHost    bool
	extraOrigins map[string]bool
	extraHosts   map[string]bool
}

// New builds a Policy for a server listening on listenAddr (for example
// "127.0.0.1:63987"). Host checking is enabled only when the address is a
// loopback address. Extra allowances are read from the environment.
func New(listenAddr string) *Policy {
	return &Policy{
		checkHost:    isLoopbackAddr(listenAddr),
		extraOrigins: envSet("NEXUS_ALLOWED_ORIGINS"),
		extraHosts:   envSet("NEXUS_ALLOWED_HOSTS"),
	}
}

func envSet(name string) map[string]bool {
	set := map[string]bool{}
	for _, v := range strings.Split(os.Getenv(name), ",") {
		if v = strings.ToLower(strings.TrimSpace(v)); v != "" {
			set[strings.TrimRight(v, "/")] = true
		}
	}
	return set
}

// isLoopbackAddr reports whether addr ("host:port") binds only to the loopback
// interface. An empty host (":8080") or 0.0.0.0 binds every interface.
func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.Trim(strings.ToLower(host), "[]")
	if host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// OriginAllowed reports whether a browser Origin may call the API. An empty
// origin (a non-browser client) is always allowed; the literal "null" (sandboxed
// frames, file:// pages) is not.
func (p *Policy) OriginAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	if p != nil && p.extraOrigins[strings.TrimRight(strings.ToLower(origin), "/")] {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	switch u.Scheme {
	case "http", "https", "wails":
	default:
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "localhost" || host == "wails.localhost" || isLoopbackIP(host)
}

// HostAllowed reports whether a request's Host header is acceptable. It is
// always true when the server is not bound to loopback only.
func (p *Policy) HostAllowed(hostHeader string) bool {
	if p == nil || !p.checkHost {
		return true
	}
	host := strings.ToLower(hostHeader)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if host == "" {
		return false
	}
	if host == "localhost" || isLoopbackIP(host) || p.extraHosts[host] {
		return true
	}
	// Any other IP literal cannot be a rebinding hostname.
	return net.ParseIP(host) != nil
}

func isLoopbackIP(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Middleware rejects requests failing OriginAllowed or HostAllowed with 403.
func (p *Policy) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !p.HostAllowed(r.Host) {
			http.Error(w, `{"error":"forbidden: unexpected Host header"}`, http.StatusForbidden)
			return
		}
		if !p.OriginAllowed(r.Header.Get("Origin")) {
			http.Error(w, `{"error":"forbidden: origin not allowed"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
