---
id: TASK-489
plan: PLAN-063
status: done
wave: 3
priority: 2
---

# TASK-489: Add `workspaceOrchView` and `workspaceScanner` unit tests

## Problem

`vscode-extension/src/workspaceOrchView.ts` `getChildren()` tree-building logic is completely untested. `vscode-extension/src/workspaceScanner.ts` `parseOrchestratorFile()` is untested — malformed or missing fields in `orchestrator.json` cause silent failures that surface as empty tree views with no user feedback.

## Checklist

- [ ] Create `vscode-extension/src/test/workspaceOrchView.test.ts`; mock `vscode.workspace.workspaceFolders` and `workspaceScanner.scan()` using vitest/jest mocks
- [ ] Test case: empty workspace (no folders) → `getChildren()` returns `[]`
- [ ] Test case: folder with valid `orchestrator.json` → correct tree items with task counts
- [ ] Test case: folder with malformed `orchestrator.json` (missing `counters`, bad JSON) → `parseOrchestratorFile()` returns a safe default object and does not throw
- [ ] Test case: multi-folder workspace — both folders appear as root tree items; task counts are disambiguated per folder
- [ ] Create `vscode-extension/src/test/workspaceScanner.test.ts`; use `fs` mocks or temp files to simulate various `orchestrator.json` shapes
- [ ] Test `parseOrchestratorFile()` with: missing file, empty file, valid file, file with extra unknown fields (should be ignored), file where `counters.nextTaskId` is a string not a number

## Files to change

- `vscode-extension/src/test/workspaceOrchView.test.ts` (new)
- `vscode-extension/src/test/workspaceScanner.test.ts` (new)
- `vscode-extension/src/workspaceScanner.ts` (harden `parseOrchestratorFile` if needed)

## Acceptance criteria

- [ ] `npm test` in `vscode-extension/` passes all new tests
- [ ] `parseOrchestratorFile()` never throws on any file content — always returns a typed default
- [ ] Coverage for `workspaceOrchView.getChildren` and `workspaceScanner.parseOrchestratorFile` reaches >= 80 %
