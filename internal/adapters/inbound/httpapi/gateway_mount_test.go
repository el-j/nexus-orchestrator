package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
)

// The OpenAI-compatible routes of a mounted gateway are served on the API's own port, behind the same
// token as /api; without a gateway the paths do not exist.
func TestGatewayMountedUnderV1(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "tok")
	inner := http.NewServeMux()
	inner.HandleFunc("/v1/models", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("models")) })
	srv := httpapi.NewServer(&mockOrchestrator{}, nil, nil).WithGateway(inner)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	get := func(path, auth string) (int, string) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b := make([]byte, 16)
		n, _ := resp.Body.Read(b)
		return resp.StatusCode, string(b[:n])
	}
	if code, _ := get("/v1/models", ""); code != http.StatusUnauthorized {
		t.Errorf("without a token: HTTP %d, want 401", code)
	}
	if code, _ := get("/v1/models", "Bearer wrong"); code != http.StatusUnauthorized {
		t.Errorf("with a wrong token: HTTP %d, want 401", code)
	}
	if code, body := get("/v1/models", "Bearer tok"); code != http.StatusOK || body != "models" {
		t.Errorf("with the token: HTTP %d %q", code, body)
	}

	bare := httptest.NewServer(httpapi.NewServer(&mockOrchestrator{}, nil, nil).Handler())
	t.Cleanup(bare.Close)
	req, _ := http.NewRequest(http.MethodGet, bare.URL+"/v1/models", nil)
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("without a gateway: HTTP %d, want 404", resp.StatusCode)
	}
}
