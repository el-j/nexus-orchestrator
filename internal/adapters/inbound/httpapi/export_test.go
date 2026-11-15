package httpapi

import (
	"context"
	"net/http"
	"time"
)

// RunHTTP exposes the server loop to the external tests.
func RunHTTP(ctx context.Context, addr string, h http.Handler, grace time.Duration) error {
	return runHTTP(ctx, addr, h, grace)
}
