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
import { viewQueueCommand } from './viewQueue';
import { showProvidersCommand } from './index';

const client = (providers: Provider[] | Error) =>
  ({
    getProviders: vi.fn(() =>
      providers instanceof Error ? Promise.reject(providers) : Promise.resolve(providers),
    ),
  }) as unknown as NexusClient;
const p = (o: Partial<Provider> = {}): Provider => ({ name: 'Ollama', active: true, ...o });

beforeEach(() => {
  Object.values(v).forEach((f) => f.mockReset());
  v.getConfiguration.mockReturnValue({ update: v.update });
  v.update.mockResolvedValue(undefined);
});

describe('selectProviderCommand', () => {
  it('reports an unreachable daemon', async () => {
    await selectProviderCommand(client(new Error('down')));
    expect(v.showErrorMessage).toHaveBeenCalledWith(
      expect.stringContaining('Could not reach daemon'),
    );
    expect(v.showQuickPick).not.toHaveBeenCalled();
  });

  it('reports when no provider is active', async () => {
    await selectProviderCommand(client([p({ active: false })]));
    expect(v.showErrorMessage).toHaveBeenCalledWith(
      'No active providers. Check your Nexus daemon.',
    );
  });

  it('offers Auto plus every model of every active provider', async () => {
    v.showQuickPick.mockResolvedValue(undefined);
    await selectProviderCommand(
      client([
        p({ models: ['a', 'b'] }),
        p({ name: 'LM', models: [], activeModel: 'qwen' }),
        p({ name: 'Bare' }),
        p({ name: 'Off', active: false, models: ['x'] }),
      ]),
    );
    const labels = (v.showQuickPick.mock.calls[0][0] as Array<{ label: string }>).map(
      (i) => i.label,
    );
    expect(labels).toEqual([
      '$(zap) Auto (let Nexus choose)',
      'Ollama / a',
      'Ollama / b',
      'LM / qwen',
      'Bare / default',
    ]);
  });

  it('does not touch settings when the picker is dismissed', async () => {
    v.showQuickPick.mockResolvedValue(undefined);
    await selectProviderCommand(client([p()]));
    expect(v.update).not.toHaveBeenCalled();
  });

  it('saves provider and model to the workspace and confirms', async () => {
    v.showQuickPick.mockImplementation(async (items: unknown[]) => items[1]);
    await selectProviderCommand(client([p({ models: ['llama3'] })]));
    expect(v.update).toHaveBeenNthCalledWith(1, 'defaultProvider', 'Ollama', 2);
    expect(v.update).toHaveBeenNthCalledWith(2, 'defaultModel', 'llama3', 2);
    expect(v.showInformationMessage).toHaveBeenCalledWith(
      'Default provider set to: Ollama / llama3',
    );
  });

  it('clears the defaults when Auto is chosen', async () => {
    v.showQuickPick.mockImplementation(async (items: unknown[]) => items[0]);
    await selectProviderCommand(client([p()]));
    expect(v.update).toHaveBeenNthCalledWith(1, 'defaultProvider', '', 2);
    expect(v.showInformationMessage).toHaveBeenCalledWith('Default provider set to: Auto');
  });

  it('explains a settings write failure instead of throwing (e.g. no folder open)', async () => {
    v.showQuickPick.mockImplementation(async (items: unknown[]) => items[0]);
    v.update.mockRejectedValueOnce(new Error('no workspace')).mockRejectedValueOnce('plain');
    await expect(selectProviderCommand(client([p()]))).resolves.toBeUndefined();
    expect(v.showErrorMessage).toHaveBeenLastCalledWith(
      'Nexus: could not save the default provider — no workspace',
    );
    await selectProviderCommand(client([p()]));
    expect(v.showErrorMessage).toHaveBeenLastCalledWith(
      'Nexus: could not save the default provider — plain',
    );
    expect(v.showInformationMessage).not.toHaveBeenCalled();
  });
});

describe('viewQueueCommand', () => {
  it('focuses the task queue view', () => {
    viewQueueCommand();
    expect(v.executeCommand).toHaveBeenCalledWith('nexus.taskQueue.focus');
  });
});

describe('showProvidersCommand', () => {
  it('reports a fetch failure (Error and non-Error)', async () => {
    await showProvidersCommand(client(new Error('down')));
    expect(v.showErrorMessage).toHaveBeenLastCalledWith('Nexus: Failed to fetch providers — down');
    const c = { getProviders: vi.fn().mockRejectedValue('weird') } as unknown as NexusClient;
    await showProvidersCommand(c);
    expect(v.showErrorMessage).toHaveBeenLastCalledWith('Nexus: Failed to fetch providers — weird');
  });

  it('offers to open settings when nothing is configured', async () => {
    v.showInformationMessage.mockResolvedValueOnce('Open Settings');
    await showProvidersCommand(client([]));
    expect(v.executeCommand).toHaveBeenCalledWith('workbench.action.openSettings', 'nexus.');
    v.executeCommand.mockClear();
    v.showInformationMessage.mockResolvedValueOnce(undefined);
    await showProvidersCommand(client([]));
    expect(v.executeCommand).not.toHaveBeenCalled();
  });

  it('lists providers with status, kind, URL, errors and models', async () => {
    v.showQuickPick.mockResolvedValue(undefined);
    await showProvidersCommand(
      client([
        p({
          kind: 'ollama',
          baseURL: 'http://x',
          details: 'd',
          contextLimit: 8192,
          activeModel: 'm',
          models: ['m', 'n'],
        }),
        p({ name: 'Down', active: false, error: 'refused' }),
        p({ name: 'Plain' }),
      ]),
    );
    const items = v.showQuickPick.mock.calls[0][0] as Array<{
      label: string;
      description?: string;
      detail?: string;
    }>;
    expect(items[0].description).toBe('● active · kind: ollama · http://x');
    expect(items[0].detail).toBe('d · Context limit: 8192 · Model: m · Available: m, n');
    expect(items[1].description).toBe('○ inactive');
    expect(items[1].detail).toBe('Error: refused');
    expect(items[2].detail).toBeUndefined();
  });

  it('shows the full details of the chosen provider', async () => {
    v.showQuickPick.mockImplementation(async (items: unknown[]) => items[0]);
    await showProvidersCommand(
      client([
        p({
          kind: 'ollama',
          baseURL: 'http://x',
          activeModel: 'm',
          models: ['m'],
          details: 'd',
          error: 'e',
        }),
      ]),
    );
    expect(v.showInformationMessage).toHaveBeenCalledWith(
      [
        'Ollama — Active',
        'Kind: ollama',
        'Base URL: http://x',
        'Current model: m',
        'Models (1): m',
        'Details: d',
        'Last error: e',
      ].join('\n'),
    );
    v.showQuickPick.mockImplementation(async (items: unknown[]) => items[0]);
    await showProvidersCommand(client([p({ active: false })]));
    expect(v.showInformationMessage).toHaveBeenLastCalledWith('Ollama — Inactive');
  });

  it('does nothing when the picker is dismissed', async () => {
    v.showQuickPick.mockResolvedValue(undefined);
    await showProvidersCommand(client([p()]));
    expect(v.showInformationMessage).not.toHaveBeenCalled();
  });
});
