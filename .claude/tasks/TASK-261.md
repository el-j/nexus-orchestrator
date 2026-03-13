# TASK-261 — Wire Antigravity in buildProviders()

**Plan**: PLAN-040  
**Status**: done

Add Antigravity as an always-registered provider in `buildProviders()` in both entry points.
Uses `NEXUS_ANTIGRAVITY_URL` env var (default `http://127.0.0.1:4315/v1`).
