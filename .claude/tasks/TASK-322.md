# TASK-322: VS Code extension — modelId in registerSession + display

**Plan:** PLAN-048
**Role:** vscode
**Dependencies:** TASK-315

## Goal

When the VS Code extension registers an AI session with nexusOrchestrator, include the active Copilot model ID so it shows up in the "Registered Sessions" view with a model badge.

## Changes

### `vscode-extension/src/sessionMonitor.ts`

In `detectAndRegister()`, after selecting Copilot models, read the first model's ID:

```ts
const models = await vscode.lm.selectChatModels({ vendor: 'copilot' });
if (models.length === 0) {
  /* existing retry logic */ return;
}

const modelId = models[0]?.id ?? models[0]?.name ?? '';

await this.client.registerSession({
  agentName: 'GitHub Copilot',
  source: 'vscode',
  projectPath: workspacePath,
  externalId: `${vscode.env.machineId}:${workspacePath}`,
  modelId, // NEW
});
```

### `vscode-extension/src/nexusClient.ts`

Add `modelId?: string` to the `RegisterSessionPayload` type / interface (or the inline object type used in `registerSession`).

### `vscode-extension/src/agentDetector.ts`

When building `AgentInfo` for detected extensions, include the model if available:

- For `detectVSCodeExtensions`: if the extension is GitHub Copilot and LM models are available, include `modelId: models[0]?.id`
- For `detectActiveLMParticipants`: include `modelId: participant.id` or `participant.name`

### Build + package

After changes:

```bash
cd vscode-extension && npm run compile 2>&1
```

Verify compilation succeeds. **Do NOT rebuild the .vsix** — that is done in release tasks.

## Acceptance Criteria

- [ ] `registerSession` call includes `modelId` when Copilot model is available
- [ ] `RegisterSessionPayload` type includes `modelId?: string`
- [ ] `AgentInfo` type has `modelId?: string`
- [ ] `npm run compile` in `vscode-extension/` succeeds with no TypeScript errors
- [ ] Go backend already accepts `modelId` in POST body (handled by TASK-318)
