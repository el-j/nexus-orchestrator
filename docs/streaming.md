# Live output while a task runs

While a model writes its reply, the daemon publishes it on the event stream (`GET /api/events`,
server-sent events) next to the usual task status events. Subscribe once and you see the text as
it is generated instead of waiting for the task to finish.

## Events

| `type`         | Fields                  | Meaning                                                                       |
| -------------- | ----------------------- | ----------------------------------------------------------------------------- |
| `task.restart` | `taskId`                | A new attempt starts: throw away the text you collected for this task so far. |
| `task.chunk`   | `taskId`, `text`, `seq` | The next piece of the reply. `seq` counts from 1 within an attempt.           |

A restart is sent before **every** attempt: the first try, a retry, the next provider in the
fallback chain after a failure, and each turn of an agentic task. So a client needs only one rule:
on `task.restart` clear the buffer, on `task.chunk` append `text`.

## Guarantees and limits

- The final result of a task is unchanged: it is the complete reply, written and verified as
  before. Live text is not stored anywhere; only a subscriber that was connected sees it.
- A subscriber that reads too slowly may miss events (the hub drops instead of blocking). A gap in
  `seq` tells you so; the finished task still holds the full result.
- Pieces that arrive within 50 ms of each other are joined into one chunk.
- A provider that cannot stream sends its whole reply as one chunk after it finishes.
- Providers that stream: OpenAI-compatible endpoints (OpenAI, the VS Code model bridge and other
  servers speaking `stream: true`) and Ollama. Anthropic, Gemini and LM Studio still send one chunk
  when the reply is complete.
- A stream that ends without its terminator (`[DONE]` / `done`) counts as a failure, so the next
  provider takes over and a `task.restart` precedes its text.
