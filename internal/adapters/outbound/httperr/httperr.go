// Package httperr turns the HTTP failures routing cares about (rate limit, exhausted quota, overload)
// into domain.ProviderError, so every LLM adapter reports them the same way.
package httperr

import (
	"io"
	"net/http"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// snippetLimit is how much of an error body is read to tell a quota from a short rate limit.
const snippetLimit = 2048

// FromResponse returns a ProviderError for status 429, 402, 503 or 529, and nil for every other
// status, which the adapter reports as before. It reads the start of the body (at most 2 KiB) only for
// the statuses it classifies, so a successful response is left untouched.
func FromResponse(provider string, resp *http.Response, now time.Time) error {
	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusPaymentRequired, http.StatusServiceUnavailable, 529:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, snippetLimit))
		return domain.ClassifyHTTP(provider, resp.StatusCode, resp.Header.Get("Retry-After"), string(b), now)
	}
	return nil
}
