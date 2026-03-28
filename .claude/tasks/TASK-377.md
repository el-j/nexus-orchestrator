# TASK-377: Fix WorkspaceOrchView — LiveTasksGroupNode filter by project path

**Plan:** PLAN-055 | **Wave:** 2 | **Status:** done | **Role:** extension

## Problem

`WorkspaceOrchView` builds a tree of workspace folders and shows "Live Tasks" under each folder. But `LiveTasksGroupNode` receives all tasks from `getAllTasks()` and creates task items globally, NOT filtered by the `folderPath` of that tree node. The count "Live Tasks (N)" is therefore global.

## Files to Edit

- `vscode-extension/src/workspaceOrchView.ts` — filter tasks in `LiveTasksGroupNode`

## Investigation Needed

Read `vscode-extension/src/workspaceOrchView.ts` before implementing to understand:

- Where `getAllTasks()` is called
- Where `LiveTasksGroupNode` is constructed with tasks
- What `folderPath` variable holds

## Implementation

Find the code that passes tasks to `LiveTasksGroupNode` and add a filter:

```typescript
// BEFORE (wrong — global tasks):
const liveNode = new LiveTasksGroupNode(allTasks, folder.uri.fsPath);

// AFTER (correct — filtered to this folder):
const folderTasks = allTasks.filter((t) => t.projectPath === folder.uri.fsPath);
const liveNode = new LiveTasksGroupNode(folderTasks, folder.uri.fsPath);
```

Similarly, inside `LiveTasksGroupNode.getChildren()`:

```typescript
getChildren(): vscode.TreeItem[] {
  // tasks are already filtered to this folder, no further filter needed
  return this.tasks
    .slice(0, 10)  // keep cap to avoid UI overload
    .map(t => new TaskTreeItem(t));
}
```

## Verification

- `npm test` in vscode-extension directory passes
- When workspace A has tasks `/a/task1`, `/a/task2` and workspace B has `/b/task3`:
  - Folder A node shows "Live Tasks (2)"
  - Folder B node shows "Live Tasks (1)"

## Status

done
