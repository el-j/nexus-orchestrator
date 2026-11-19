# nexus-router

`nexus-router` is a standalone LLM Gateway and OpenAI-compatible HTTP reverse proxy.

It manages local and cloud model providers, fallback ladders, provider rate-limit cooldowns, and direct streaming without requiring the full orchestrator daemon.

---

## Standalone OpenAI-Compatible Proxy Server

Start a local proxy server that serves `/v1/chat/completions`:

```sh
nexus-router serve --port 8080 --default-role coder
```

### Supported Endpoints

- `POST /v1/chat/completions`: Standard OpenAI chat completions endpoint (supports `stream: true` SSE).
- `GET /v1/models`: Returns list of discovered models across all available providers.
- `GET /health`: Returns gateway readiness status and discovered provider counts.

### Using with External Tools (curl, Python, LangChain)

```sh
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "auto",
    "messages": [
      {"role": "user", "content": "Explain Go concurrency patterns"}
    ]
  }'
```

```python
from openai import OpenAI

client = OpenAI(base_url="http://localhost:8080/v1", api_key="not-needed")
response = client.chat.completions.create(
    model="auto",
    messages=[{"role": "user", "content": "Hello from Python!"}],
)
print(response.choices[0].message.content)
```

---

## Direct CLI Prompt Execution

Execute prompts directly against configured provider ladders from the terminal:

```sh
# Generate response using default role ladder
nexus-router prompt "Write a Go unit test for a LRU cache"

# Target a specific role profile
nexus-router prompt "Refactor this architecture" --role architect
```

---

## Provider Discovery & Status

Inspect all locally discovered providers (Ollama, LM Studio, Claude CLI, Copilot) and configured cloud providers (OpenAI, Anthropic, Gemini):

```sh
nexus-router providers
```
