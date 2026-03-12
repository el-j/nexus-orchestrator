# PLAN-026: Production Hardening & GUI Completeness

**Status:** active  
**Created:** 2026-03-12  
**Goal:** Close the last gaps in the v0.9.x feature set — orchestrator robustness, GUI completeness (History + Settings views), and README refresh.

---

## Background

A stale-task audit (2026-03-12) of PLAN-002 and PLAN-021 found that 19 of 22 "todo" tasks were already fully implemented in the codebase. Three genuine gaps remained:

| Stale task | Gap | PLAN-026 task |
|-----------|-----|--------------|
| TASK-014 | No retry limits, no queue cap, no path normalization | TASK-189 |
| TASK-021 | No HistoryView.vue — completed tasks not visible in GUI | TASK-190 |
| TASK-022 | No SettingsView.vue — no GUI settings page | TASK-191 |

Additionally, the README `Features` section is outdated (still describes v0.7 capabilities). This is remedied by TASK-192.

---

## Wave Breakdown

### Wave 1 — Backend (unblocked)
| Task | Title | Role | Effort |
|------|-------|------|--------|
| TASK-189 | Orchestrator hardening: startup recovery, path normalization, queue cap, retry limit | backend | M |

### Wave 2 — Frontend (TASK-190 unblocked; TASK-191 depends on TASK-189)
| Task | Title | Role | Effort | Depends On |
|------|-------|------|--------|-----------|
| TASK-190 | GUI: Task History view | frontend | M | — |
| TASK-191 | GUI: Settings view | frontend | M | TASK-189 |

### Wave 3 — Docs (unblocked, can run in parallel with Wave 1)
| Task | Title | Role | Effort |
|------|-------|------|--------|
| TASK-192 | README v0.9.x refresh | devops | S |

---

## Success Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` passes
- [ ] A task that is `PROCESSING` at daemon startup is re-queued automatically (TASK-189)
- [ ] The GUI has 7 views: Dashboard, Backlog, Providers, Discovery, AI Sessions, **History**, **Settings** (TASK-190, TASK-191)
- [ ] README Features section covers all v0.9.x capabilities with no PLAN-002 task references (TASK-192)
- [ ] All 4 tasks reach `status: done` in `orchestrator.json`

---

## Out of Scope

- Removing the empty `internal/adapters/outbound/fs_writeback/` directory (cosmetic; writeback is handled by `fs_writer`)  
- Full systray OS-level integration (tracked separately; `tray.go` scaffold is intentionally a no-op pending main-thread coordination)
- Any new LLM provider adapters
