---
id: PLAN-074
title: 'Quota-aware routing, a model ladder and a usage ledger (from the v0.10.0 audit)'
status: completed
createdAt: 2026-10-16
tasks:
  [
    TASK-581,
    TASK-582,
    TASK-583,
    TASK-584,
    TASK-585,
    TASK-586,
    TASK-587,
    TASK-588,
    TASK-589,
    TASK-590,
    TASK-591,
    TASK-592,
    TASK-593,
    TASK-594,
    TASK-595,
    TASK-596,
    TASK-597,
    TASK-598,
    TASK-599,
    TASK-600,
    TASK-601,
    TASK-602,
    TASK-603,
    TASK-604,
    TASK-605,
    TASK-606,
    TASK-607,
  ]
---

# PLAN-074 — Do not run out of quota; spend fewer tokens

Source: [docs/audit-v0.10.0.md](../../docs/audit-v0.10.0.md). Goal: when a provider's quota runs out the
work continues on the next model or waits for the reset, and expensive models are asked only for what
needs them, with the tokens measured. One task = one small change, tests first, each ends with its
documentation (REST, MCP, CLI, surface matrix, changelog).

## A. Quota and rate limits

| Id  | Change                                                                                                                                                        |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A1  | Typed provider errors (`RateLimited{RetryAfter}`, `QuotaExhausted{ResetAt}`, `Overloaded`, `Auth`, `Network`); every adapter maps its HTTP status and headers |
| A2  | Per-provider/model cooldown store (until time), set from A1, cleared on success; `ProviderInfo.RateLimited` becomes true and shows `until`                    |
| A3  | The chain skips cooled-down candidates; the engine logs why                                                                                                   |
| A4  | New task state `WAITING` (all candidates cooled down): resume time in the task, REST, SSE, MCP, CLI, UI and extension; resumes by itself                      |
| A5  | e2e: a fake provider answers 429 with `Retry-After`; the task fails over, then waits, then completes                                                          |

## B. Model ladder

| Id  | Change                                                                                                                                  |
| --- | --------------------------------------------------------------------------------------------------------------------------------------- |
| B1  | Policy `ladder` per role: an ordered list of `provider/model` (wildcards allowed); validated at load                                    |
| B2  | Routing uses the ladder: start at the role's rung, step down on A-errors or when the task is small, step up after a failed verification |
| B3  | UI and CLI: show and edit the ladder; docs in `rulebased-workflow.md`                                                                   |

## C. Usage ledger

| Id  | Change                                                                                                  |
| --- | ------------------------------------------------------------------------------------------------------- |
| C1  | Adapters return tokens in/out/cached from the response `usage` (streaming included); port gains `Usage` |
| C2  | Store per task, provider, model, role; migration; `GET /api/usage`, MCP `get_usage`, `nexus-cli usage`  |
| C3  | Mission Control card and extension status: today's tokens per provider, estimate vs. limit              |
| C4  | Budget per provider and day in the policy; reaching it acts like a cooldown (A2)                        |

## D. Fewer tokens

| Id  | Change                                                                                                                         |
| --- | ------------------------------------------------------------------------------------------------------------------------------ |
| D1  | Context by relevance: FTS-ranked brain entries and file outlines instead of whole `ContextFiles`, with a per-task token budget |
| D2  | History capped by tokens; older turns summarized by a cheap model                                                              |
| D3  | Anthropic prompt caching for the stable prefix (rules, briefing); keep the prefix byte-stable                                  |
| D4  | Run the verification command before asking a frontier model again; send it only the failing output                             |
| D5  | Report: tokens saved by D1-D4 against the v0.10.0 behaviour on the same tasks (measured with C)                                |

## E. Close the audit

| Id  | Change                                                                                                                   |
| --- | ------------------------------------------------------------------------------------------------------------------------ |
| E1  | Go coverage to 100 % (247 statements), gate to 100; `-Wl,-no_warn_duplicate_libraries` for the cgo link warning on macOS |
| E2  | e2e: `nexus connect`, plan folders, delegation, discovered plans, real-daemon extension flow                             |
| E3  | Mutation results recorded per package (adapters, Vue) with a list of accepted equivalent mutants                         |
| E4  | Repository: archive the two historical audits, fix the 41 → 42 tool count, close or archive the ~230 stale `todo` tasks  |
| E5  | `httpapi_client.IngestKnowledge`: implement or remove from the port                                                      |

Order: E1/E4 first (cheap, clean base), then A → B → C → D. D5 decides whether D2/D3 stay.

## Task map

Wave 1 TASK-581..585 (typed errors, cooldown, Claude driver, usage port and ledger), wave 2 586..588 (quota routing, WAITING, e2e), wave 3 589..591 (ladder), wave 4 592..593 (usage UI, budgets), wave 5 594..598 (fewer tokens), wave 6 599..602 (vendor drivers and status; 599 needs `agy` installed), wave 7 603..607 (audit gaps). Design for the vendor bridges: [docs/subscription-bridges.md](../../docs/subscription-bridges.md).
