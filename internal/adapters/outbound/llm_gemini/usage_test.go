package llm_gemini

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var _ ports.UsageReporter = (*Adapter)(nil)

const usageMeta = `"usageMetadata":{"promptTokenCount":40,"candidatesTokenCount":10,"cachedContentTokenCount":15},"modelVersion":"gemini-x-002"`

func TestUsage_NothingBeforeTheFirstCall(t *testing.T) {
	if _, ok := NewAdapter("k", "m").LastUsage(); ok {
		t.Error("no usage yet")
	}
}

func TestUsage_FromAGenerateContentReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"candidates":[{"content":{"parts":[{"text":"hi"}]}}],`+usageMeta+`}`)
	}))
	defer srv.Close()
	a := NewAdapter("k", "gemini-x", srv.URL)
	if _, err := a.GenerateCode(context.Background(), "q"); err != nil {
		t.Fatal(err)
	}
	u, ok := a.LastUsage()
	want := domain.Usage{Model: "gemini-x-002", InputTokens: 40, OutputTokens: 10, CachedTokens: 15}
	if !ok || u != want {
		t.Errorf("usage %+v %v; want %+v", u, ok, want)
	}
}

func TestUsage_FromAStreamTakesTheLastChunk(t *testing.T) {
	body := "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"he\"}]}}],\"usageMetadata\":{\"promptTokenCount\":40,\"candidatesTokenCount\":1}}\n\n" +
		"data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"llo\"}]},\"finishReason\":\"STOP\"}]," + usageMeta + "}\n\n"
	a := streamAdapter(t, 200, "text/event-stream", body, nil)
	if _, out, err := stream(a); err != nil || out != "hello" {
		t.Fatalf("%q %v", out, err)
	}
	u, ok := a.LastUsage()
	if !ok || u.InputTokens != 40 || u.OutputTokens != 10 || u.CachedTokens != 15 || u.Model != "gemini-x-002" {
		t.Errorf("usage %+v %v", u, ok)
	}
}

func TestUsage_AStreamServedAsOneJSONBodyStillReportsUsage(t *testing.T) {
	a := streamAdapter(t, 200, "application/json", `{"candidates":[{"content":{"parts":[{"text":"x"}]}}],`+usageMeta+`}`, nil)
	if _, _, err := stream(a); err != nil {
		t.Fatal(err)
	}
	if u, ok := a.LastUsage(); !ok || u.InputTokens != 40 {
		t.Errorf("%+v %v", u, ok)
	}
}

func TestUsage_NoMetadataLeavesTheConfiguredModel(t *testing.T) {
	a := streamAdapter(t, 200, "text/event-stream", "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"x\"}]},\"finishReason\":\"STOP\"}]}\n\n", nil)
	if _, _, err := stream(a); err != nil {
		t.Fatal(err)
	}
	if u, ok := a.LastUsage(); !ok || u.InputTokens != 0 || u.Model == "" {
		t.Errorf("%+v %v", u, ok)
	}
}

func TestUsage_AChunkThatOnlyNamesTheModelKeepsTheCounts(t *testing.T) {
	body := "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"a\"}]}}],\"usageMetadata\":{\"promptTokenCount\":7,\"candidatesTokenCount\":1}}\n\n" +
		"data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"b\"}]},\"finishReason\":\"STOP\"}],\"modelVersion\":\"gemini-y\"}\n\n"
	a := streamAdapter(t, 200, "text/event-stream", body, nil)
	if _, _, err := stream(a); err != nil {
		t.Fatal(err)
	}
	if u, _ := a.LastUsage(); u.InputTokens != 7 || u.Model != "gemini-y" {
		t.Errorf("%+v", u)
	}
}
