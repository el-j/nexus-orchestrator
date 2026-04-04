# TASK-290 — TS tests for AgentDetector + delegation command

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** TypeScript · tests (vitest)  
**Depends on:** TASK-282, TASK-284  

## Objective

Unit tests for the two new TypeScript classes using vitest + the existing VS Code mock infrastructure.

## `vscode-extension/src/agentDetector.test.ts`

### Setup

Use the pattern from existing test files (mock `vscode` via `__mocks__`; mock `NexusClient` via `vi.mock`).

```typescript
import { vi, describe, it, expect, beforeEach } from 'vitest';
vi.mock('vscode');
vi.mock('./nexusClient');
```

### Test 1 — Detects Cline via extension API

```typescript
it('registers Cline session when saoudrizwan.claude-dev is installed', async () => {
  vi.mocked(vscode.extensions.getExtension).mockImplementation((id: string) =>
    id === 'saoudrizwan.claude-dev' ? { id } as any : undefined
  );
  const client = new NexusClient('http://localhost');
  vi.mocked(client.registerSession).mockResolvedValue({ id: 'sess-1', ...} as AISession);

  const detector = new AgentDetector(client, fakeContext());
  await detector.detectAll();

  expect(client.registerSession).toHaveBeenCalledWith(
    expect.objectContaining({ agentName: 'Cline', source: 'vscode-discovered' })
  );
});
```

### Test 2 — Detects Claude CLI via filesystem

```typescript
it('registers Claude CLI when ~/.claude/settings.json exists with apiKey', async () => {
  vi.spyOn(fs.promises, 'readFile').mockImplementation(async (p: any) => {
    if (String(p).endsWith('settings.json')) return JSON.stringify({ apiKey: 'sk-ant-test' });
    throw Object.assign(new Error(), { code: 'ENOENT' });
  });
  vi.spyOn(fs.promises, 'stat').mockRejectedValue(Object.assign(new Error(), { code: 'ENOENT' }));

  await detector.detectAll();
  expect(client.registerSession).toHaveBeenCalledWith(
    expect.objectContaining({ agentName: 'Claude CLI' })
  );
});
```

### Test 3 — Deregisters on disappearance

```typescript
it('deregisters session when agent disappears between ticks', async () => {
  // Tick 1: Cline present → registerSession called
  vi.mocked(vscode.extensions.getExtension).mockReturnValueOnce({ id: 'saoudrizwan.claude-dev' } as any);
  await detector.detectAll();
  const sessionId = 'sess-cline';
  // Manually set internal map (or trigger via mocked registerSession return)

  // Tick 2: Cline absent
  vi.mocked(vscode.extensions.getExtension).mockReturnValue(undefined);
  await detector.detectAll();

  expect(client.deregisterSession).toHaveBeenCalledWith(sessionId);
});
```

### Test 4 — Does not throw on network failure

```typescript
it('handles registerSession network failure silently', async () => {
  vi.mocked(vscode.extensions.getExtension).mockReturnValue({ id: 'saoudrizwan.claude-dev' } as any);
  vi.mocked(client.registerSession).mockRejectedValue(new Error('ECONNREFUSED'));

  await expect(detector.detectAll()).resolves.not.toThrow();
});
```

## `vscode-extension/src/commands/delegateToNexus.test.ts`

### Test 1 — Copilot path opens chat

```typescript
it('opens Copilot Chat for GitHub Copilot session', async () => {
  const session: AISession = { id: 's1', agentName: 'GitHub Copilot', source: 'vscode', status: 'active', lastActivity: '' };
  vi.mocked(client.delegateSession).mockResolvedValue({ instruction: 'You are now...', sessionId: 's1' });

  await delegateToNexusCommand(client, session);

  expect(vscode.commands.executeCommand).toHaveBeenCalledWith(
    'workbench.action.chat.open',
    expect.objectContaining({ query: expect.stringContaining('nexusOrchestrator') })
  );
});
```

### Test 2 — CLI path writes .nexus-delegate.md

```typescript
it('writes .nexus-delegate.md for Cline session', async () => {
  const session: AISession = { id: 's2', agentName: 'Cline', source: 'vscode-discovered', status: 'active',
    projectPath: '/home/user/myproject', lastActivity: '' };
  vi.mocked(client.delegateSession).mockResolvedValue({ instruction: 'You are now...', sessionId: 's2' });

  await delegateToNexusCommand(client, session);

  expect(vscode.workspace.fs.writeFile).toHaveBeenCalledWith(
    expect.objectContaining({ fsPath: expect.stringContaining('.nexus-delegate.md') }),
    expect.any(Uint8Array)
  );
});
```

### Test 3 — Handles delegateSession failure

```typescript
it('shows error notification when delegateSession fails', async () => {
  vi.mocked(client.delegateSession).mockRejectedValue(new Error('session not found'));
  await delegateToNexusCommand(client, { id: 's3', agentName: 'Cline', source: 'vscode', status: 'active', lastActivity: '' });
  expect(vscode.window.showErrorMessage).toHaveBeenCalled();
});
```

## Run command

```sh
cd vscode-extension && npm test
```

## Acceptance Criteria

- All 7 tests pass
- Tests run in isolation (no real filesystem or network access)
- Test coverage for `agentDetector.ts` ≥ 80% (tracked in `coverage/lcov-report/`)
