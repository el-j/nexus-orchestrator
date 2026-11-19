---
id: TASK-629
planId: PLAN-076
title: 'Probe providers in parallel and without holding the cache lock'
role: backend
status: done
createdAt: 2026-10-31
---

# TASK-629 — Provider probes blocked routing

`DiscoveryService.cachedHealth` held `cacheMu` while it pinged a provider and read its models, and
`ListProviders`, `DetectActive` and `FindForModel` probed one provider after another. Every 30 s (the
cache TTL) the first caller therefore waited for the sum of all probes (0.95 s cold with 20 providers
and `agy models`; up to 10 s per unreachable host) and every other caller, including task routing,
waited behind the lock. Now the probe runs outside the lock, callers of the same provider share the
probe in flight, and all stale providers are probed at once (`probeAll`); `DetectActive` returns at
the first healthy provider in order. Tests: `discovery_parallel_test.go` (the in-flight test fails on the old code).
