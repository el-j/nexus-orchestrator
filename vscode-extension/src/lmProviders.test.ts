import { beforeEach, describe, expect, it, vi } from 'vitest';

const models = { list: [] as Array<{ id: string; vendor: string }> };
vi.mock('vscode', () => ({ lm: { selectChatModels: async () => models.list } }));

import { dropLmProviders, isInSync, providerNameFor, syncLmProviders } from './lmProviders';
import type { NexusClient } from './nexusClient';

const bridge = { baseUrl: 'http://127.0.0.1:5000/v1', token: 't0k', close: async () => {} };

function fakeClient(existing: Array<{ name: string; baseURL?: string }>) {
  return {
    getProviders: vi.fn(async () => existing),
    removeProvider: vi.fn(async () => {}),
    registerProvider: vi.fn(async () => {}),
  };
}

describe('lmProviders', () => {
  beforeEach(() => {
    models.list = [
      { id: 'gpt-4o', vendor: 'copilot' },
      { id: 'Claude Sonnet 4.5', vendor: 'copilot' },
      { id: 'gemini-3', vendor: 'antigravity' },
    ];
  });

  it('builds a safe provider name from vendor and model', () => {
    expect(providerNameFor('Copilot', 'Claude Sonnet 4.5')).toBe(
      'vscode-copilot-claude-sonnet-4.5',
    );
  });

  it('registers one openaicompat provider per model, pointing at the bridge', async () => {
    const c = fakeClient([]);
    const names = await syncLmProviders(c as unknown as NexusClient, bridge, []);
    expect(names).toEqual([
      'vscode-copilot-gpt-4o',
      'vscode-copilot-claude-sonnet-4.5',
      'vscode-antigravity-gemini-3',
    ]);
    expect(c.registerProvider).toHaveBeenCalledWith({
      name: 'vscode-copilot-gpt-4o',
      kind: 'openaicompat',
      baseUrl: bridge.baseUrl,
      apiKey: 't0k',
      model: 'gpt-4o',
    });
  });

  it('shares only the listed vendors', async () => {
    const c = fakeClient([]);
    const names = await syncLmProviders(c as unknown as NexusClient, bridge, ['antigravity']);
    expect(names).toEqual(['vscode-antigravity-gemini-3']);
  });

  it('removes stale bridge providers and never touches others', async () => {
    const c = fakeClient([{ name: 'vscode-copilot-old' }, { name: 'LM Studio' }]);
    await syncLmProviders(c as unknown as NexusClient, bridge, []);
    expect(c.removeProvider).toHaveBeenCalledTimes(1);
    expect(c.removeProvider).toHaveBeenCalledWith('vscode-copilot-old');
  });

  it('drops every bridge provider', async () => {
    const c = fakeClient([{ name: 'vscode-a' }, { name: 'ollama' }, { name: 'vscode-b' }]);
    await dropLmProviders(c as unknown as NexusClient);
    expect(c.removeProvider.mock.calls).toEqual([['vscode-a'], ['vscode-b']]);
  });

  describe('isInSync', () => {
    const wanted = [
      'vscode-copilot-gpt-4o',
      'vscode-copilot-claude-sonnet-4.5',
      'vscode-antigravity-gemini-3',
    ];
    const live = (url = bridge.baseUrl) => wanted.map((name) => ({ name, baseURL: url }));

    it('is true when the daemon lists exactly the wanted providers on this bridge', async () => {
      const c = fakeClient([...live(), { name: 'ollama' }]);
      expect(await isInSync(c as unknown as NexusClient, bridge, [])).toBe(true);
    });
    it('is false when one is missing', async () => {
      const c = fakeClient(live().slice(1));
      expect(await isInSync(c as unknown as NexusClient, bridge, [])).toBe(false);
    });
    it('is false when an extra one is listed', async () => {
      const c = fakeClient([...live(), { name: 'vscode-copilot-extra', baseURL: bridge.baseUrl }]);
      expect(await isInSync(c as unknown as NexusClient, bridge, [])).toBe(false);
    });
    it('is false when a provider points at an old bridge', async () => {
      const c = fakeClient(live('http://127.0.0.1:9/v1'));
      expect(await isInSync(c as unknown as NexusClient, bridge, [])).toBe(false);
    });
    it('is false when a listed name is not wanted', async () => {
      const c = fakeClient([...live().slice(1), { name: 'vscode-other', baseURL: bridge.baseUrl }]);
      expect(await isInSync(c as unknown as NexusClient, bridge, [])).toBe(false);
    });
  });
});
