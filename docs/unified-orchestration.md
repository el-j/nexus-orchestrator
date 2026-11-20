# Unified Autonomous Orchestration & Adaptive Harness

`nexus orchestrate` provides a single, unified pipeline command combining dynamic RAG context retrieval, model-tier context tailoring, cost-to-capability affinity routing, test-driven self-healing execution, and continuous project memory digestion.

---

## The 5-Pillar Architecture

```
                       +-------------------------------+
                       |  Spec File / Prompt Input     |
                       +---------------+---------------+
                                       |
                                       v
                       +-------------------------------+
                       |  1. Dynamic RAG Synthesis     |  <-- Tailors context by model tier
                       |     (Frontier / Standard /    |      (Invariant, Arch, Avoid rules)
                       |      Local compact headers)   |      Saves 40-70% prompt tokens
                       +---------------+---------------+
                                       |
                                       v
                       +-------------------------------+
                       |  2. Affinity Matching Router  |  <-- Complexity estimation (O(1)-O(N))
                       |     (Cost vs Capability Score |      Headroom pre-flight checks
                       |      & Session Affinity)      |      External agent matching
                       +---------------+---------------+
                                       |
                                       v
                       +-------------------------------+
                       |  3. Test-Driven Execution     |  <-- Configurable turn budget
                       |     (Adaptive TDD Harness &   |      Quality gates (Permissive/Strict)
                       |      AST Diagnostic Filter)   |      Deterministic model simulator
                       +---------------+---------------+
                                       |
                                       v
                       +-------------------------------+
                       |  4. Continuous Project Memory |  <-- Ingests outcomes, conventions,
                       |     (SQLite FTS5 Knowledge)   |      file maps, and pitfalls
                       +-------------------------------+
```

---

## 1. Dynamic RAG Synthesis & Model-Tier Context Tailoring

Different language models require different context densities:

- **Frontier Models** (`claude-3-5-sonnet`, `gpt-4o`): Receive concise rule invariants (`* INVARIANT: ...`), architecture boundaries, and critical pitfalls. Detailed prose is pruned to save high-value tokens.
- **Standard Models** (`gemini-1.5-flash`, `deepseek-chat`): Receive balanced architectural notes, targeted conventions, and relevant file paths.
- **Local Models** (`qwen2.5-coder`, `llama3.2`): Receive compact, prioritized guidelines with minimal prose overhead, preserving small context windows.

Context synthesis automatically deducts redundant token overhead while preserving exact architectural constraints.

---

## 2. Cost-to-Capability Affinity Routing

The affinity engine evaluates tasks before dispatching:

- **Task Complexity Estimation**: Scans instruction for high-impact keywords (`refactor`, `migrate`, `concurrency`, `deadlock`, `race`) and verification rigor to classify complexity as `Trivial`, `Standard`, or `High`.
- **Context Headroom Pre-flight**: Verifies that the provider's active model has sufficient context capacity for the prompt plus output tokens before invocation.
- **Affinity Scoring**: Ranks models by matching role capability, complexity tier, and token cost.
- **External AI Session Affinity**: Prefers reusing active, warm external AI sessions (Claude Code, Cursor, Antigravity) with the same project and role, preventing cold-start overhead.

---

## 3. Test-Driven Adaptive Harness & AST Diagnostics

The autonomous harness wraps code generation in a rigorous feedback loop:

- **Verification Command Ladder**: Runs commands such as `go test ./...` or `npm test` after code changes.
- **Quality Gates**:
  - `QualityGatePermissive`: Accepts task completion if verification passes.
  - `QualityGateStrict`: Enforces zero lint warnings, zero uncovered branches, and strict pass criteria.
- **AST Diagnostic Filter**: When tests fail, raw compiler and linter outputs are parsed and distilled to relevant line-numbered diagnostics, saving up to 80% token overhead across self-healing retry turns.
- **Simulation Mode**: Supports deterministic in-memory simulation (`ModelSimulator`, `ScriptedSimulator`) for fast CI/CD validation without external API dependencies.

---

## 4. Continuous Autonomous Project Memory

When tasks complete or fail, outcomes are automatically digested into the project's knowledge base:

- **Convention Mining**: Detects patterns like "use errors.Is", "table-driven tests", or "strict permissions" and records them as `convention` knowledge entries.
- **Pitfall Avoidance**: Records failures and root causes as `pitfall` knowledge entries to prevent subsequent agents from repeating identical errors.
- **File Map Tracking**: Updates structural project relationships in SQLite FTS5 for rapid multi-turn lookup.

---

## 5. CLI Usage & Pipeline Command

Execute the complete end-to-end pipeline with `nexus orchestrate`:

### From an Inline Instruction

```sh
nexus orchestrate "implement rate limiting middleware" \
  --project . \
  --role coder \
  --target internal/adapters/inbound/http/ratelimit.go \
  --verify "go test -v ./internal/adapters/inbound/http/" \
  --max-turns 5
```

### From a Specification File

```sh
nexus orchestrate specs/auth-feature.md \
  --project /path/to/project \
  --model claude-3-5-sonnet \
  --role architect
```

### Output Formats

Standard human-readable progress:

```text
[1/4] RAG Retrieval: tailored for frontier tier (saved 320 tokens)
[2/4] Affinity Routing: role=coder, complexity=standard, target=internal/adapters/inbound/http/ratelimit.go
[3/4] Task Dispatched: ID=task-8f92b, status=QUEUED
[4/4] Knowledge Digested: 2 entries ingested into project memory
Pipeline Finished: status=COMPLETED, elapsed=1.45s
```

Structured JSON telemetry (`--json`):

```json
{
  "task_id": "task-8f92b",
  "status": "COMPLETED",
  "prompt_tokens": 850,
  "rag_tokens_saved": 320,
  "model_tier": "frontier",
  "assigned_model": "claude-3-5-sonnet",
  "turns_executed": 1,
  "duration": "1.45s",
  "learnings_saved": 2,
  "suggested_files": ["internal/adapters/inbound/http/server.go"]
}
```
