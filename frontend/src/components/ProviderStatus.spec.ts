import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import ProviderStatus from './ProviderStatus.vue';
import type { ProviderConfig, ProviderInfo } from '../types/domain';

const wails = vi.hoisted(() => ({
  listProviderConfigs: vi.fn(),
  addProviderConfig: vi.fn(),
  updateProviderConfig: vi.fn(),
  removeProviderConfig: vi.fn(),
  toastAdd: vi.fn(),
}));
vi.mock('../types/wails', () => ({
  listProviderConfigs: wails.listProviderConfigs,
  addProviderConfig: wails.addProviderConfig,
  updateProviderConfig: wails.updateProviderConfig,
  removeProviderConfig: wails.removeProviderConfig,
}));
vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: wails.toastAdd }) }));

const FormStub = {
  name: 'ProviderConfigForm',
  props: ['modelValue', 'onClose', 'onSave'],
  template: '<div data-testid="form" />',
};
const DialogStub = {
  name: 'AppConfirmDialog',
  props: ['open', 'title', 'message', 'danger'],
  emits: ['confirm', 'cancel'],
  template: '<div data-testid="confirm" :data-open="open">{{ title }}</div>',
};

const info = (o: Partial<ProviderInfo> = {}): ProviderInfo => ({
  name: 'LM Studio',
  active: true,
  activeModel: 'qwen',
  models: [],
  ...o,
});
const cfg = (o: Partial<ProviderConfig> = {}): ProviderConfig => ({
  id: 'c1',
  name: 'Mine',
  kind: 'ollama',
  baseUrl: 'http://x',
  apiKey: '',
  model: '',
  enabled: true,
  createdAt: '',
  updatedAt: '',
  ...o,
});

async function mountStatus(props: Record<string, unknown> = {}) {
  const w = mount(ProviderStatus, {
    props: { providers: [], ...props },
    global: { stubs: { ProviderConfigForm: FormStub, AppConfirmDialog: DialogStub } },
  });
  await flushPromises();
  return w;
}

beforeEach(() => {
  Object.values(wails).forEach((f) => f.mockReset());
  wails.listProviderConfigs.mockResolvedValue([]);
});

describe('ProviderStatus — live providers', () => {
  it('hints how to get started when none are detected', async () => {
    const w = await mountStatus();
    expect(w.text()).toContain('No providers detected');
  });

  it('shows active providers with model and base URL', async () => {
    const w = await mountStatus({ providers: [info({ baseURL: 'http://127.0.0.1:1234' })] });
    expect(w.text()).toContain('LM Studio');
    expect(w.text()).toContain('qwen');
    expect(w.text()).toContain('http://127.0.0.1:1234');
    expect(w.find('.animate-pulse').exists()).toBe(true);
  });

  it('shows the error of an inactive provider, truncated to 80 chars', async () => {
    const long = 'x'.repeat(100);
    const w = await mountStatus({ providers: [info({ active: false, error: long })] });
    expect(w.text()).toContain('x'.repeat(80) + '…');
    expect(w.text()).not.toContain('x'.repeat(81));
    const short = await mountStatus({ providers: [info({ active: false, error: 'down' })] });
    expect(short.text()).toContain('down');
    expect(short.text()).not.toContain('down…');
  });

  it('hides the error of an active provider', async () => {
    const w = await mountStatus({ providers: [info({ error: 'stale' })] });
    expect(w.text()).not.toContain('stale');
  });

  it('calls refresh from the Refresh button', async () => {
    const refresh = vi.fn();
    const w = await mountStatus({ refresh });
    await w
      .findAll('button')
      .find((b) => b.text().includes('Refresh'))!
      .trigger('click');
    expect(refresh).toHaveBeenCalled();
  });

  it('tolerates a missing refresh callback', async () => {
    const w = await mountStatus();
    await w
      .findAll('button')
      .find((b) => b.text().includes('Refresh'))!
      .trigger('click');
    expect(w.exists()).toBe(true);
  });

  it('hides the live section when hideDiscovered is set', async () => {
    const w = await mountStatus({ hideDiscovered: true, providers: [info()] });
    expect(w.text()).not.toContain('Providers:');
    expect(w.text()).toContain('Configured Providers');
  });

  describe('activity-based model state', () => {
    const state = (provider: string, models: string[], active = true) => ({
      provider,
      models,
      lastSeen: '',
      active,
    });

    it('matches lm-studio activity to an LM Studio provider', async () => {
      const w = await mountStatus({
        providers: [info()],
        modelStates: [state('lm-studio', ['a'])],
      });
      expect(w.text()).toContain('a loaded');
    });

    it('counts several loaded models', async () => {
      const w = await mountStatus({
        providers: [info({ name: 'Ollama' })],
        modelStates: [state('ollama', ['a', 'b'], false)],
      });
      expect(w.text()).toContain('2 models loaded');
      expect(w.find('.bg-slate-500').exists()).toBe(true); // idle dot
    });

    it('matches antigravity and generic names', async () => {
      const w = await mountStatus({
        providers: [info({ name: 'Antigravity' }), info({ name: 'Foo-Bar' })],
        modelStates: [state('antigravity', ['ag']), state('foobar', ['fb'])],
      });
      expect(w.text()).toContain('ag loaded');
      expect(w.text()).toContain('fb loaded');
    });

    it('shows nothing for a provider without activity or models', async () => {
      const w = await mountStatus({
        providers: [info({ name: 'Ollama' })],
        modelStates: [state('lm-studio', ['a']), state('ollama', [])],
      });
      expect(w.text()).not.toContain('loaded');
    });
  });
});

describe('ProviderStatus — configured providers', () => {
  it('lists configs and the empty state', async () => {
    expect((await mountStatus()).text()).toContain('No configured providers');
    wails.listProviderConfigs.mockResolvedValue([
      cfg(),
      cfg({ id: 'c2', name: 'Off', enabled: false }),
    ]);
    const w = await mountStatus();
    expect(w.text()).toContain('Mine');
    expect(w.text()).toContain('Off');
    expect(w.find('.bg-slate-600').exists()).toBe(true); // disabled dot
  });

  it('toasts when loading configs fails', async () => {
    wails.listProviderConfigs.mockRejectedValue(new Error('nope'));
    await mountStatus();
    expect(wails.toastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'error', summary: 'Load Configs Failed' }),
    );
  });

  it('opens an empty form to add a provider, then adds and refreshes', async () => {
    const refresh = vi.fn();
    const w = await mountStatus({ refresh });
    await w
      .findAll('button')
      .find((b) => b.text().includes('Add Provider'))!
      .trigger('click');
    const form = w.findComponent(FormStub);
    expect(form.props('modelValue')).toBeNull();

    wails.listProviderConfigs.mockResolvedValue([cfg()]);
    await form.props('onSave')({ name: 'New' });
    await flushPromises();
    expect(wails.addProviderConfig).toHaveBeenCalledWith({ name: 'New' });
    expect(w.findComponent(FormStub).exists()).toBe(false); // closed
    expect(w.text()).toContain('Mine'); // reloaded
    expect(refresh).toHaveBeenCalled();
  });

  it('opens the form pre-filled to edit and merges the changes on save', async () => {
    wails.listProviderConfigs.mockResolvedValue([cfg()]);
    const w = await mountStatus();
    await w.find('button[title="Edit"]').trigger('click');
    const form = w.findComponent(FormStub);
    expect(form.props('modelValue')).toMatchObject({ id: 'c1' });

    await form.props('onSave')({ name: 'Renamed' });
    expect(wails.updateProviderConfig).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'c1', name: 'Renamed', kind: 'ollama' }),
    );
    expect(wails.addProviderConfig).not.toHaveBeenCalled();
  });

  it('closes the form on request', async () => {
    const w = await mountStatus();
    await w
      .findAll('button')
      .find((b) => b.text().includes('Add Provider'))!
      .trigger('click');
    w.findComponent(FormStub).props('onClose')();
    await flushPromises();
    expect(w.findComponent(FormStub).exists()).toBe(false);
  });

  it('asks before deleting, and only deletes on confirm', async () => {
    wails.listProviderConfigs.mockResolvedValue([cfg()]);
    const refresh = vi.fn();
    const w = await mountStatus({ refresh });
    await w.find('button[title="Delete"]').trigger('click');
    const dialog = w.findComponent(DialogStub);
    expect(dialog.props('open')).toBe(true);
    expect(dialog.props('title')).toContain('Mine');
    expect(wails.removeProviderConfig).not.toHaveBeenCalled();

    dialog.vm.$emit('confirm');
    await flushPromises();
    expect(wails.removeProviderConfig).toHaveBeenCalledWith('c1');
    expect(refresh).toHaveBeenCalled();
    expect(w.findComponent(DialogStub).props('open')).toBe(false);
  });

  it('cancelling the dialog deletes nothing', async () => {
    wails.listProviderConfigs.mockResolvedValue([cfg()]);
    const w = await mountStatus();
    await w.find('button[title="Delete"]').trigger('click');
    w.findComponent(DialogStub).vm.$emit('cancel');
    await flushPromises();
    expect(wails.removeProviderConfig).not.toHaveBeenCalled();
    expect(w.findComponent(DialogStub).props('open')).toBe(false);
  });
});
