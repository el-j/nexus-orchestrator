package llm_openaicompat

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdapter_ReportsABaseURLThatCannotFormARequest(t *testing.T) {
	a := NewAdapter("local", "http://bad\x7fhost", "", "m")
	if a.Ping(context.Background()) {
		t.Error("an unusable URL is not a live provider")
	}
	if _, err := a.GetAvailableModels(context.Background()); err == nil {
		t.Error("an unusable URL must be an error")
	}
}

func TestGetAvailableModels_ReportsAModelListThatIsNotJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "<html>not the models</html>")
	}))
	defer srv.Close()
	if _, err := NewAdapter("local", srv.URL, "", "m").GetAvailableModels(context.Background()); err == nil {
		t.Error("an undecodable list must be an error")
	}
}

func TestFirstNonEmpty_IsEmptyWhenEveryValueIs(t *testing.T) {
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("got %q", got)
	}
	if got := firstNonEmpty("", "b", "c"); got != "b" {
		t.Errorf("got %q", got)
	}
}
