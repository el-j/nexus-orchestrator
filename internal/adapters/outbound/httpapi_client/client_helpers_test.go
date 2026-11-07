package httpapi_client_test

import (
	"net/http"
	"testing"
)

func assertContentType(t *testing.T, r *http.Request) {
	t.Helper()
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

type countingTransport struct {
	calls *int
	base  http.RoundTripper
}

func (tr *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	*tr.calls++
	base := tr.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}
