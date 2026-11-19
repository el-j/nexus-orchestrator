import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { NexusClient, Provider } from '../nexusClient';

const v = vi.hoisted(() => ({
  showQuickPick: vi.fn(),
  showInformationMessage: vi.fn(),
  showErrorMessage: vi.fn(),
  executeCommand: vi.fn(),
  update: vi.fn(),
  getConfiguration: vi.fn(),
}));
vi.mock('vscode', () => ({
  ConfigurationTarget: { Workspace: 2 },
  window: {
    showQuickPick: v.showQuickPick,
    showInformationMessage: v.showInformationMessage,
    showErrorMessage: v.showErrorMessage,
  },
  workspace: { getConfiguration: v.getConfiguration },
  commands: { executeCommand: v.executeCommand },
}));
vi.mock('./submitTask', () => ({ sendCurrentContextCommand: vi.fn(), submitTaskCommand: vi.fn() }));
vi.mock('./delegateToNexus', () => ({ delegateToNexusCommand: vi.fn() }));

import { selectProviderCommand } from './selectProvider';
import { showProvidersCommand } from './index';

const until = new Date(Date.now() + 3600_000).toISOString();
const providers: Provider[] = [
  {
    name: 'Claude',
    active: true,
    models: ['sonnet'],
    rateLimited: true,
    rateLimitedUntil: until,
    cooldownReason: 'rate limited (429)',
  },
  {
    name: 'Copilot',
    active: true,
    models: ['auto'],
    quota: { window: 'm', status: 's', utilization: 0.95 },
  },
  { name: 'Ollama', active: true, models: ['qwen'] },
];
const client = { getProviders: vi.fn(() => Promise.resolve(providers)) } as unknown as NexusClient;

beforeEach(() => {
  Object.values(v).forEach((f) => f.mockReset());
  v.getConfiguration.mockReturnValue({ update: v.update });
});

describe('provider pickers show limits', () => {
  it('the default-provider picker marks a provider that cools down or whose plan is nearly used up', async () => {
    v.showQuickPick.mockResolvedValue(undefined);
    await selectProviderCommand(client);
    const items = v.showQuickPick.mock.calls[0][0] as Array<{ label: string; detail?: string }>;
    const byLabel = Object.fromEntries(items.map((i) => [i.label, i.detail]));
    expect(byLabel['Claude / sonnet']).toMatch(/cooling down until \d{2}:\d{2}/);
    expect(byLabel['Copilot / auto']).toBe('plan 95% used');
    expect(byLabel['Ollama / qwen']).toBeUndefined();
  });

  it('the provider list shows the same limits next to the status', async () => {
    v.showQuickPick.mockResolvedValue(undefined);
    await showProvidersCommand(client);
    const items = v.showQuickPick.mock.calls[0][0] as Array<{ label: string; description: string }>;
    const byLabel = Object.fromEntries(items.map((i) => [i.label, i.description]));
    expect(byLabel.Claude).toMatch(/● active · .*cooling down until \d{2}:\d{2}/);
    expect(byLabel.Copilot).toContain('plan 95% used');
    expect(byLabel.Ollama).not.toContain('cooling');
  });
});
