# Switching to another provider when a limit is reached

Nexus keeps working when one subscription runs out: the task moves to the next provider, the limited one is
left alone until its reset, and if every provider is limited the task waits and resumes by itself. This page
is a run of exactly that with real providers, so you can repeat it. It was run on 2026-10-17 with the daemon
built from `develop` (v0.10.0 plus PLAN-074), a logged-in `claude` and a local model in LM Studio.

## Set up (nothing here touches your own daemon or database)

```sh
make build   # or: go build -o /tmp/nexus-daemon ./cmd/nexus-daemon && go build -o /tmp/nexus-cli ./cmd/nexus-cli
NEXUS_DB_PATH=/tmp/nexus-demo.db NEXUS_LISTEN_ADDR=127.0.0.1:64987 NEXUS_MCP_ADDR=127.0.0.1:64988 /tmp/nexus-daemon &
export NEXUS_ADDR=http://127.0.0.1:64987
lms load <a small model> --context-length 8192     # LM Studio is picked up by itself on :1234
curl -X POST $NEXUS_ADDR/api/providers/config -H 'Content-Type: application/json' -d '{
  "name": "Claude (subscription)", "kind": "cli",
  "baseUrl": "claude -p --output-format stream-json --model haiku", "model": "haiku", "enabled": true }'
```

Put a `.nexus/policy.json` in the project, for example `{"quotaThreshold": 0.85}`, and look at the order the
providers would be tried in:

```
$ nexus-cli routing --project /path/to/project --role architect
  #  Provider                Model          Rung  State
  1  Claude (subscription)   haiku             -  ready
  2  LM Studio               nexus-local       -  ready
```

## What happened

1. **A task for the architect role** ran on Claude (`ranBy: Claude (subscription)`). `nexus-cli usage` then showed
   the call: 2 input, 293 output, 3 933 cached tokens, $0.0008. The provider list showed the plan reading
   `{"window":"five_hour","status":"allowed","utilization":-1,...}`: **a plan far from its limit reports no
   share used** (`-1`); the share appears once the client starts warning. So `quotaThreshold` acts only
   when the client reports one, and the reliable switch is the refusal below.
2. **A second Claude provider whose limit was used up.** A real refusal cannot be produced on demand, so a stand-in
   `claude` script printed what the client prints on a refusal (a `rate_limit_event` with status `rejected`,
   `seven_day`, a reset time, and an error result with HTTP status 429). **This is a simulation; the refusal
   shape is modelled on the success fields and has not been captured from a real refusal.** A task pinned to that provider:

   ```
   provider Claude (limit simulated) failed: seven_day limit reached, resets Sat 09:30
   [failover: primary provider failed, executed via LM Studio]
   verification passed ... completed via LM Studio
   ```

   The file was written by the local model and the verification command passed.

3. **The next task** logged `Claude (limit simulated) is cooling down until 09:30:46 (...); skipped` and went
   to the next provider without asking the limited one again. `nexus-cli routing` showed the state:
   `cooling down until Sat 09:30 (...)`. A task that is cancelled while it runs is aborted (`204`, then `CANCELLED`).
4. **LM Studio showed up in the usage ledger** only after its adapter reported tokens (it did not before this
   run: the ledger listed Claude only). Now: 235 input, 97 output for one small task.

## What to expect and what not

- With **several providers**, a limit moves work to the next one at once, and `nexus-cli routing` and
  `GET /api/providers` show who is cooling down and until when.
- With **one provider**, or when **all are limited**, the task is `WAITING` until the earliest reset and then
  queued again by the daemon (see [token-economy.md](token-economy.md) for the other levers).
- A small local model is slower and weaker: the third task of this run (a plain function) kept the model
  generating for more than eight minutes before it was cancelled, and a task in a project with earlier
  conversation includes that history. Put a **model ladder** in the policy so the local model is the cheap
  first rung and Claude the later one, or the other way round, as you prefer.
- Subscriptions are for one person on one machine; Nexus runs the official `claude` program and never touches
  the login (see [subscription-bridges.md](subscription-bridges.md)).
