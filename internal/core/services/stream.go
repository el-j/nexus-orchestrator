package services

import (
	"context"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// chunkInterval is the shortest gap between two chunk events of one attempt. Pieces that arrive
// faster are joined, so a fast model does not flood the event hub; the first piece goes out at once.
const chunkInterval = 50 * time.Millisecond

// publish sends a stream event if a broadcaster is configured.
func (o *OrchestratorService) publish(ev ports.TaskEvent) {
	o.mu.Lock()
	b := o.broadcaster
	o.mu.Unlock()
	if b != nil {
		b.Broadcast(ev)
	}
}

// liveChat is llm.Chat with the reply published while it is generated: a restart event opens
// the attempt, then chunk events follow. A client that cannot stream sends its whole reply as
// one chunk. The returned reply and error are those of the client, so callers treat it like Chat.
func (o *OrchestratorService) liveChat(ctx context.Context, taskID string, llm ports.LLMClient, history []domain.Message) (string, error) {
	return o.live(taskID, func(onChunk func(string)) (string, error) {
		if s, ok := llm.(ports.ChatStreamer); ok {
			return s.ChatStream(ctx, history, onChunk)
		}
		reply, err := llm.Chat(ctx, history)
		onChunk(reply)
		return reply, err
	})
}

// liveGenerate is llm.GenerateCode published the same way, for the single-prompt path used
// when no session store is configured. A client that can stream receives the prompt as one
// user message; any other goes through GenerateCode and sends its whole reply as one chunk.
func (o *OrchestratorService) liveGenerate(ctx context.Context, taskID string, llm ports.LLMClient, prompt string) (string, error) {
	return o.live(taskID, func(onChunk func(string)) (string, error) {
		if s, ok := llm.(ports.ChatStreamer); ok {
			return s.ChatStream(ctx, []domain.Message{{Role: domain.RoleUser, Content: prompt}}, onChunk)
		}
		reply, err := llm.GenerateCode(ctx, prompt)
		onChunk(reply)
		return reply, err
	})
}

// live opens an attempt with a restart event, runs generate and publishes the pieces it hands to
// its callback, at most one chunk event per chunkInterval. Whatever is left is flushed at the end,
// also when generate failed.
func (o *OrchestratorService) live(taskID string, generate func(onChunk func(string)) (string, error)) (string, error) {
	o.publish(ports.TaskEvent{Type: ports.EventTaskRestart, TaskID: taskID})
	var (
		seq     int
		pending string
		last    time.Time
	)
	flush := func() {
		if pending == "" {
			return
		}
		seq++
		o.publish(ports.TaskEvent{Type: ports.EventTaskChunk, TaskID: taskID, Text: pending, Seq: seq})
		pending, last = "", time.Now()
	}
	reply, err := generate(func(piece string) {
		pending += piece
		if time.Since(last) >= chunkInterval {
			flush()
		}
	})
	flush()
	return reply, err
}
