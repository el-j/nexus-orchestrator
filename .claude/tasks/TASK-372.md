# TASK-372: Plan scanner — Enrich nexus-kind summary with project stats from orchestrator.json

**Plan:** PLAN-054 | **Wave:** 1 | **Status:** done | **Role:** backend

## Goal

When the plan scanner discovers `orchestrator.json` (kind=nexus), parse it and populate
the `summary` field with structured project stats: plan count, task count, completion ratio.
This enables the Plans view Project Brain card to show meaningful data.

## Context

- File: `internal/adapters/outbound/sys_scanner/plan_scanner.go`
- When `classifyFile` identifies `orchestrator.json` in `.claude/`, it calls `extractSummary`
  which reads the first ~200 printable chars — this ends up being just the raw JSON start
- `domain.DiscoveredPlanFile.summary` is a `string` — we can put structured text there
- `orchestrator.json` structure: `{ "plans": { "PLAN-NNN": { "status": "completed"|... } }, "tasks": {...}, "counters": { "nextPlanId": N, "nextTaskId": N } }`

## Implementation

### Add a dedicated `summarizeOrchestratorJSON` function:

```go
func summarizeOrchestratorJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var doc struct {
		Counters struct {
			NextPlanID int `json:"nextPlanId"`
			NextTaskID int `json:"nextTaskId"`
		} `json:"counters"`
		Plans map[string]struct {
			Status string `json:"status"`
		} `json:"plans"`
		UpdatedAt string `json:"updatedAt"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return ""
	}
	totalPlans := len(doc.Plans)
	completedPlans := 0
	for _, p := range doc.Plans {
		if p.Status == "completed" || p.Status == "done" {
			completedPlans++
		}
	}
	totalTasks := doc.Counters.NextTaskID - 1
	return fmt.Sprintf("plans: %d (%d completed) · tasks: %d · updated: %s",
		totalPlans, completedPlans, totalTasks, doc.UpdatedAt)
}
```

### Call it from `classifyFile` for the nexus case:

```go
case name == "orchestrator.json" && strings.HasSuffix(filepath.Dir(absPath), ".claude"):
    summary = summarizeOrchestratorJSON(absPath)
    return domain.PlanFileKind("nexus"), "json", summary, true
```

### Dependencies

- Add `"encoding/json"` import if not present
- No new domain types or DB schema changes

## Expected output

`summary` field for orchestrator.json becomes:
`"plans: 52 (42 completed) · tasks: 363 · updated: 2026-03-28T18:10:00Z"`

## Status

done
