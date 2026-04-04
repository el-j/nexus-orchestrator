---
id: TASK-491
plan: PLAN-063
status: done
wave: 4
priority: 2
---

# TASK-491: Fix `github-action` `resolveAgents` - rate-limit error handling

## Problem

`github-action/src/agents.ts` `fetchCategoryIndex()` silently returns `[]` on a GitHub API 403 or 429 (rate-limit), or when the API returns an HTML error page instead of JSON. Downstream callers report `Agent not found` which is a misleading error that causes confused support tickets.

## Checklist

- [ ] In `fetchCategoryIndex()`, check `response.status` after `fetch()`: if `403` or `429`, throw a `RateLimitError` (custom class extending `Error`) with message `'GitHub API rate limited (HTTP ${status}). Retry after ${retryAfter}s.'` — read `Retry-After` header when present
- [ ] If the response `Content-Type` is not `application/json` or the body is not a JSON array, throw a `ParseError` with the first 200 characters of the raw response body included so callers can diagnose HTML error pages
- [ ] Wrap `fetchCategoryIndex()` call site in a retry loop with exponential backoff: base delay 2 s, multiplier 2x, max 3 attempts, max delay 30 s; only retry on `RateLimitError`
- [ ] Export `RateLimitError` and `ParseError` from `agents.ts` so tests can assert on them
- [ ] Add tests in `github-action/__tests__/agents.test.ts`: - fetch returns 429 -> `RateLimitError` thrown after 3 retries - fetch returns 200 with HTML -> `ParseError` thrown immediately (no retry) - fetch returns 200 with valid JSON array -> returns parsed agents
- [ ] Update the action's error output step to include a check for `RateLimitError` and set `core.setFailed('GitHub API rate limit exceeded. Try again later.')` instead of a generic agent-not-found message

## Files to change

- `github-action/src/agents.ts`
- `github-action/__tests__/agents.test.ts`
- `github-action/src/index.ts` or wherever the action error handler lives

## Acceptance criteria

- [ ] A mocked 429 response causes the action to retry 3 times then fail with `'GitHub API rate limit exceeded'`
- [ ] A mocked HTML response causes the action to fail immediately with `ParseError` and the response preview in the message
- [ ] All existing `agents.test.ts` tests continue to pass
