package httpapi_client

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// A body that cannot be marshalled is reported before any request is made.
func TestSend_MarshalFailure(t *testing.T) {
	c := NewClient("http://127.0.0.1:1")
	err := c.send(context.Background(), call{op: "demo", method: http.MethodPost, path: "/x", body: make(chan int)}, nil)
	if err == nil || !strings.Contains(err.Error(), "marshal request") {
		t.Fatalf("got %v", err)
	}
}
