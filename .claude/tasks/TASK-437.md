---
id: TASK-437
plan: PLAN-058
status: done
---

# TASK-437: ProvidersView filtering (search + status)

**Plan:** PLAN-058 (Wave 3)

## Problem

- No way to filter providers or AI coding tools
- Many cards with no filtering capability

## Solution

1. Add search input for providers (filters by name, baseURL)
2. Add status filter: All, Active, Unreachable
3. Add search input for AI tools (filters by name, kind)
4. Add status filter for AI tools: All, Running, Installed, Detected

## Files

- `frontend/src/views/ProvidersView.vue`

## Acceptance

- Search filters providers and AI tools in real-time
- Status filters work independently for each section
