package llm_lmstudio

import (
	"context"
	"testing"
)

func TestAdapter_ReportsABaseURLThatCannotFormARequest(t *testing.T) {
	a := NewLMStudioAdapter("http://bad\x7fhost")
	if _, err := a.GenerateCode(context.Background(), "x"); err == nil {
		t.Error("an unusable URL must be an error")
	}
}
