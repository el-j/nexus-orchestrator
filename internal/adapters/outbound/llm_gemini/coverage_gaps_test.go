package llm_gemini

import (
	"context"
	"testing"
)

func TestChatStream_SkipsAChunkWithoutCandidates(t *testing.T) {
	// The closing chunk of a Gemini stream often carries only usage numbers.
	body := part("Hel") + "data: {\"usageMetadata\":{\"promptTokenCount\":3,\"candidatesTokenCount\":2}}\n\n" + "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"lo\"}]},\"finishReason\":\"STOP\"}]}\n\n"
	a := streamAdapter(t, 200, "text/event-stream", body, nil)
	var got string
	reply, err := a.ChatStream(context.Background(), nil, func(s string) { got += s })
	if err != nil || reply != "Hello" || got != "Hello" {
		t.Errorf("reply=%q streamed=%q err=%v", reply, got, err)
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
