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
	onChunk := func(piece string) {
		pending += piece
		if time.Since(last) >= chunkInterval {
			flush()
		}
	}

	var (
		reply string
		err   error
	)
	if s, ok := llm.(ports.ChatStreamer); ok {
		reply, err = s.ChatStream(ctx, history, onChunk)
	} else {
		reply, err = llm.Chat(ctx, history)
		onChunk(reply)
	}
	flush() // whatever is left, also when the attempt failed
	return reply, err
}
