package httperr

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

func resp(t *testing.T, status int, retryAfter, body string) *http.Response {
	t.Helper()
	h := http.Header{}
	if retryAfter != "" {
		h.Set("Retry-After", retryAfter)
	}
	r := &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(body))}
	t.Cleanup(func() { _ = r.Body.Close() })
	return r
}

func TestFromResponse(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	r429 := resp(t, 429, "20", "")
	defer r429.Body.Close()
	pe := FromResponse("anthropic", r429, now)
	got, ok := domain.AsProviderError(pe)
	if !ok || got.Kind != domain.ErrKindRateLimited || !got.ResetAt.Equal(now.Add(20*time.Second)) || pe.Error() != "anthropic: rate limited (429)" {
		t.Fatalf("429: %v", pe)
	}
	rq := resp(t, 429, "", `{"error":{"status":"RESOURCE_EXHAUSTED"}}`)
	defer rq.Body.Close()
	if q, _ := domain.AsProviderError(FromResponse("g", rq, now)); q == nil || q.Kind != domain.ErrKindQuotaExhausted {
		t.Errorf("quota body: %v", q)
	}
	for _, status := range []int{402, 503, 529} {
		r := resp(t, status, "", "")
		if FromResponse("p", r, now) == nil {
			t.Errorf("%d must be classified", status)
		}
		r.Body.Close()
	}
	for _, status := range []int{200, 400, 401, 403, 404, 500, 502} {
		r := resp(t, status, "", "")
		if FromResponse("p", r, now) != nil {
			t.Errorf("%d must be left to the caller", status)
		}
		r.Body.Close()
	}
}

func TestSuccessBodyIsNotConsumed(t *testing.T) {
	r := resp(t, 200, "", "payload")
	defer r.Body.Close()
	if FromResponse("p", r, time.Now()) != nil {
		t.Fatal("200 is not an error")
	}
	if b, _ := io.ReadAll(r.Body); string(b) != "payload" {
		t.Errorf("body was read: %q", b)
	}
}

func TestOnlyTheStartOfALongErrorBodyIsRead(t *testing.T) {
	big := &countingBody{r: strings.NewReader(strings.Repeat("a", 10000))}
	r := &http.Response{StatusCode: 429, Header: http.Header{}, Body: big}
	t.Cleanup(func() { _ = r.Body.Close() })
	if FromResponse("p", r, time.Now()) == nil {
		t.Fatal("429 must be classified")
	}
	if big.n > 4096 {
		t.Errorf("read %d bytes", big.n)
	}
}

type countingBody struct {
	r io.Reader
	n int
}

func (c *countingBody) Read(p []byte) (int, error) { n, err := c.r.Read(p); c.n += n; return n, err }
func (c *countingBody) Close() error               { return nil }
