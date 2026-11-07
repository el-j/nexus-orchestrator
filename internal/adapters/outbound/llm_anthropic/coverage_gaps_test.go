package llm_anthropic

import "testing"

func TestFirstNonEmpty_IsEmptyWhenEveryValueIs(t *testing.T) {
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("got %q", got)
	}
	if got := firstNonEmpty("", "b", "c"); got != "b" {
		t.Errorf("got %q", got)
	}
}
