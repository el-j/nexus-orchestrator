import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { ref } from 'vue';
import ProvidersView from './ProvidersView.vue';
import type { DiscoveredAgent, ProviderConfig, ProviderInfo } from '../types/domain';

const h = vi.hoisted(() => ({
  route: { query: {} as Record<string, unknown> },
  refresh: vi.fn(),
  discoveryRefresh: vi.fn(),
  scanNow: vi.fn(),
  state: {
    providers: [] as unknown[],
    discovered: [] as unknown[],
    discoveryLoading: false,
    scanning: false,
  },
  api: {
    listProviderConfigs: vi.fn(),
    addProviderConfig: vi.fn(),
    updateProviderConfig: vi.fn(),
    removeProviderConfig: vi.fn(),
  },
}));
vi.mock('vue-router', () => ({ useRoute: () => h.route }));
vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl: () => Promise.resolve('http://d'),
}));
vi.mock('../composables/useProviders', () => ({
  useProviders: () => ({ providers: ref(h.state.providers), refresh: h.refresh }),
}));
vi.mock('../composables/useDiscovery', () => ({
  useDiscovery: () => ({
    discovered: ref(h.state.discovered),
    loading: ref(h.state.discoveryLoading),
    scanning: ref(h.state.scanning),
    refresh: h.discoveryRefresh,
    scanNow: h.scanNow,
  }),
}));
vi.mock('../types/wails', () => h.api);

const Status = {
  name: 'ProviderStatus',
  props: ['providers', 'refresh', 'hideDiscovered'],
  template: '<div class="status" />',
};
const Panel = {
  name: 'DiscoveredProvidersPanel',
  props: ['providers', 'loading', 'scanning'],
  emits: ['scan', 'promote'],
  template: '<div class="panel" />',
};
const Form = {
  name: 'ProviderConfigForm',
  props: ['modelValue', 'onClose', 'onSave'],
  template: '<div class="form" />',
};
const Dialog = {
  name: 'AppConfirmDialog',
  props: ['open', 'title', 'message'],
  emits: ['confirm', 'cancel'],
  template: '<div class="dlg" />',
};
const Indicator = { props: ['lastRefreshed'], template: '<i />' };

const prov = (o: Partial<ProviderInfo> = {}): ProviderInfo => ({
  name: 'LM Studio',
  active: true,
  activeModel: 'qwen',
  models: ['qwen', 'llama'],
  baseURL: 'http://x',
  ...o,
});
const agent = (o: Partial<DiscoveredAgent> = {}): DiscoveredAgent =>
  ({
    id: Math.random().toString(36),
    kind: 'copilot',
    name: 'Copilot',
    isRunning: false,
    detectionMethod: 'vscode-extension',
    ...o,
  }) as DiscoveredAgent;
const cfg = (o: Partial<ProviderConfig> = {}): ProviderConfig => ({
  id: 'c1',
  name: 'Mine',
  kind: 'ollama',
  baseUrl: 'http://o',
  apiKey: '',
  model: '',
  enabled: true,
  createdAt: '',
  updatedAt: '',
  ...o,
});

const fetchMock = vi.fn();
async function mountView(
  o: {
    providers?: ProviderInfo[];
    agents?: DiscoveredAgent[] | null;
    configs?: ProviderConfig[];
    query?: Record<string, unknown>;
    agentStatus?: number;
  } = {},
) {
  h.state.providers = o.providers ?? [];
  h.route.query = o.query ?? {};
  h.api.listProviderConfigs.mockResolvedValue(o.configs === undefined ? [] : o.configs);
  fetchMock.mockImplementation(() =>
    Promise.resolve(
      new Response(JSON.stringify(o.agents === undefined ? [] : o.agents), {
        status: o.agentStatus ?? 200,
      }),
    ),
  );
  const w = mount(ProvidersView, {
    global: {
      stubs: {
        ProviderStatus: Status,
        DiscoveredProvidersPanel: Panel,
        ProviderConfigForm: Form,
        AppConfirmDialog: Dialog,
        RefreshIndicator: Indicator,
      },
    },
  });
  await flushPromises();
  return w;
}
type W = Awaited<ReturnType<typeof mountView>>;
const btn = (w: W, t: string) => w.findAll('button').find((b) => b.text().includes(t))!;
const cards = (w: W) => w.findAll('section')[0].findAll('.rounded-xl.border.p-3');

beforeEach(() => {
  Object.values(h.api).forEach((f) => f.mockReset());
  h.refresh.mockReset();
  h.discoveryRefresh.mockReset();
  h.scanNow.mockReset();
  h.state = { providers: [], discovered: [], discoveryLoading: false, scanning: false };
  vi.stubGlobal('fetch', fetchMock);
});

describe('ProvidersView — registered providers', () => {
  it('counts active vs registered vs discovered', async () => {
    h.state.discovered = [{}, {}];
    const w = await mountView({ providers: [prov(), prov({ name: 'Down', active: false })] });
    const t = w.find('header').text();
    expect(t).toContain('1');
    expect(t).toContain('active of 2 registered');
    expect(t).toContain('2');
  });

  it('shows empty vs no-match states', async () => {
    expect((await mountView()).text()).toContain('No providers detected');
    const w = await mountView({ providers: [prov()] });
    await w.find('input[placeholder="Search providers…"]').setValue('zzz');
    expect(w.text()).toContain('No matching providers');
  });

  it('sorts active first (default) or by name', async () => {
    const w = await mountView({
      providers: [
        prov({ name: 'B', active: false }),
        prov({ name: 'C' }),
        prov({ name: 'A', active: false }),
      ],
    });
    const names = () => cards(w).map((c) => c.find('.font-semibold').text());
    expect(names()).toEqual(['C', 'A', 'B']);
    await w.find('header select').setValue('name');
    expect(names()).toEqual(['A', 'B', 'C']);
  });

  it('filters by status and by name or URL', async () => {
    const w = await mountView({
      providers: [
        prov({ name: 'Up', baseURL: 'http://up' }),
        prov({ name: 'Down', active: false, baseURL: 'http://down', error: 'refused' }),
      ],
    });
    await btn(w, 'Unreachable').trigger('click');
    expect(cards(w)).toHaveLength(1);
    expect(w.text()).toContain('refused');
    await btn(w, 'Active').trigger('click');
    expect(cards(w).map((c) => c.text())[0]).toContain('Up');
    await btn(w, 'All').trigger('click');
    await w.find('input[placeholder="Search providers…"]').setValue('http://dow');
    expect(cards(w)).toHaveLength(1);
  });

  it('shows the active model and highlights it among the models', async () => {
    const w = await mountView({ providers: [prov()] });
    expect(w.text()).toContain('Active model:');
    expect(w.find('.border-violet-500\\/30.bg-violet-500\\/10').text()).toBe('qwen');
  });

  it('refreshes both providers and discovery', async () => {
    const w = await mountView();
    await btn(w, 'Refresh').trigger('click');
    expect(h.refresh).toHaveBeenCalled();
    expect(h.discoveryRefresh).toHaveBeenCalled();
  });

  it('delegates scan to the discovery panel', async () => {
    const w = await mountView();
    w.findComponent(Panel).vm.$emit('scan');
    expect(h.scanNow).toHaveBeenCalled();
  });
});

describe('ProvidersView — AI coding tools', () => {
  it('shows the empty state, and tolerates failure, error status and null', async () => {
    expect((await mountView()).text()).toContain('No AI coding tools detected');
    expect((await mountView({ agents: null })).text()).toContain('No AI coding tools detected');
    expect((await mountView({ agentStatus: 500 })).text()).toContain('No AI coding tools detected');
    fetchMock.mockRejectedValue(new Error('offline'));
    const w = mount(ProvidersView, {
      global: {
        stubs: {
          ProviderStatus: Status,
          DiscoveredProvidersPanel: Panel,
          ProviderConfigForm: Form,
          AppConfirmDialog: Dialog,
          RefreshIndicator: Indicator,
        },
      },
    });
    await flushPromises(); // a rejected fetch must not surface as an unhandled rejection
    expect(w.text()).toContain('No AI coding tools detected');
  });

  const tools = () => [
    agent({ kind: 'copilot', name: 'Copilot', isRunning: true }),
    agent({ kind: 'continue', name: 'Continue', detectionMethod: 'vscode-extension' }),
    agent({
      kind: 'claude-cli',
      name: 'Claude',
      detectionMethod: 'process-scan',
      configPath: '/c.json',
    }),
    agent({ kind: 'mystery', name: '', detectionMethod: 'x' }),
  ];

  it('groups by kind, running groups first, with labels and descriptions', async () => {
    const w = await mountView({ agents: tools() });
    const t = w.text();
    expect(t.indexOf('GitHub Copilot')).toBeLessThan(t.indexOf('Claude CLI'));
    expect(t).toContain('AI pair programmer');
    expect(t).toContain('mystery'); // unknown kind falls back to the kind itself
    expect(t).toContain('/c.json');
  });

  it('labels each tool Active / Installed / Detected', async () => {
    const t = (await mountView({ agents: tools() })).text();
    expect(t).toContain('Active');
    expect(t).toContain('Installed');
    expect(t).toContain('Detected');
  });

  it('filters by state and by text', async () => {
    const w = await mountView({ agents: tools() });
    const count = () => w.findAll('.rounded-lg.border.p-3').length;
    await btn(w, 'Running').trigger('click');
    expect(count()).toBe(1);
    await btn(w, 'Installed').trigger('click');
    expect(count()).toBe(1);
    await btn(w, 'Detected').trigger('click');
    expect(count()).toBe(2);
    await w
      .findAll('button')
      .filter((b) => b.text() === 'All')[1]
      .trigger('click');
    await w.find('input[placeholder="Search tools…"]').setValue('claude');
    expect(count()).toBe(1);
    await w.find('input[placeholder="Search tools…"]').setValue('nothing');
    expect(w.text()).toContain('No matching tools');
  });

  it('sorts tools by name when asked', async () => {
    const w = await mountView({
      agents: [agent({ kind: 'cursor', name: 'Zed' }), agent({ kind: 'cursor', name: 'Abc' })],
    });
    await w.find('header select').setValue('name');
    const names = w.findAll('.rounded-lg.border.p-3 .font-medium').map((n) => n.text());
    expect(names).toEqual(['Abc', 'Zed']);
  });
});

describe('ProvidersView — configuration CRUD', () => {
  it('is collapsed by default and counts saved configs', async () => {
    const w = await mountView({ configs: [cfg()] });
    expect(w.text()).toContain('(1 saved)');
    expect(w.findAll('section').slice(-1)[0].find('div[style]').attributes('style')).toContain(
      'display: none',
    );
    await btn(w, 'Configuration').trigger('click');
    expect(
      w.findAll('section').slice(-1)[0].find('div[style]').attributes('style') ?? '',
    ).not.toContain('display: none');
  });

  it('shows the empty state and swallows a failed load', async () => {
    expect((await mountView()).text()).toContain('No provider configurations saved yet');
    h.api.listProviderConfigs.mockRejectedValue(new Error('x'));
    fetchMock.mockImplementation(() => Promise.resolve(new Response('[]')));
    const w = mount(ProvidersView, {
      global: {
        stubs: {
          ProviderStatus: Status,
          DiscoveredProvidersPanel: Panel,
          ProviderConfigForm: Form,
          AppConfirmDialog: Dialog,
          RefreshIndicator: Indicator,
        },
      },
    });
    await flushPromises();
    expect(w.text()).toContain('(0 saved)');
  });

  it('adds a provider through the form', async () => {
    const w = await mountView();
    await btn(w, 'Add Provider').trigger('click');
    const form = w.findComponent(Form);
    expect(form.props('modelValue')).toBeNull();
    h.api.listProviderConfigs.mockResolvedValue([cfg()]);
    await form.props('onSave')({ name: 'New' });
    await flushPromises();
    expect(h.api.addProviderConfig).toHaveBeenCalledWith({ name: 'New' });
    expect(h.api.updateProviderConfig).not.toHaveBeenCalled();
    expect(w.findComponent(Form).exists()).toBe(false);
    expect(w.text()).toContain('Mine');
  });

  it('edits an existing provider with an update', async () => {
    const w = await mountView({ configs: [cfg()] });
    await btn(w, 'Edit').trigger('click');
    await w.findComponent(Form).props('onSave')({ name: 'Renamed' });
    expect(h.api.updateProviderConfig).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'c1', name: 'Renamed', kind: 'ollama' }),
    );
    expect(h.api.addProviderConfig).not.toHaveBeenCalled();
  });

  it('closes the form without saving', async () => {
    const w = await mountView();
    await btn(w, 'Add Provider').trigger('click');
    w.findComponent(Form).props('onClose')();
    await flushPromises();
    expect(w.findComponent(Form).exists()).toBe(false);
  });

  it('confirms before removing, then removes and reloads', async () => {
    const w = await mountView({ configs: [cfg()] });
    await btn(w, 'Remove').trigger('click');
    expect(w.findComponent(Dialog).props('open')).toBe(true);
    expect(w.findComponent(Dialog).props('title')).toContain('Mine');
    expect(h.api.removeProviderConfig).not.toHaveBeenCalled();
    w.findComponent(Dialog).vm.$emit('confirm');
    await flushPromises();
    expect(h.api.removeProviderConfig).toHaveBeenCalledWith('c1');
    expect(w.findComponent(Dialog).props('open')).toBe(false);
  });

  it('cancelling the confirmation removes nothing', async () => {
    const w = await mountView({ configs: [cfg()] });
    await btn(w, 'Remove').trigger('click');
    w.findComponent(Dialog).vm.$emit('cancel');
    await flushPromises();
    expect(h.api.removeProviderConfig).not.toHaveBeenCalled();
  });

  it('marks disabled configs differently', async () => {
    const w = await mountView({ configs: [cfg({ enabled: false })] });
    expect(w.find('.bg-slate-600.rounded-full').exists()).toBe(true);
  });
});

describe('ProvidersView — promote and deep links create, not update (regression)', () => {
  it('promoting a discovered provider opens a pre-filled form that CREATES it', async () => {
    const w = await mountView();
    w.findComponent(Panel).vm.$emit('promote', {
      id: 'd1',
      name: 'Ollama',
      kind: 'ollama',
      baseUrl: 'http://127.0.0.1:11434',
    });
    await flushPromises();
    const form = w.findComponent(Form);
    expect(form.props('modelValue')).toMatchObject({
      name: 'Ollama',
      kind: 'ollama',
      baseUrl: 'http://127.0.0.1:11434',
      id: '',
    });
    await form.props('onSave')({ name: 'Ollama', kind: 'ollama' });
    expect(h.api.addProviderConfig).toHaveBeenCalledTimes(1);
    expect(h.api.updateProviderConfig).not.toHaveBeenCalled();
  });

  it('defaults an unknown discovered kind to openaicompat and a missing URL to empty', async () => {
    const w = await mountView();
    w.findComponent(Panel).vm.$emit('promote', { id: 'd', name: 'X', kind: '' });
    await flushPromises();
    expect(w.findComponent(Form).props('modelValue')).toMatchObject({
      kind: 'openaicompat',
      baseUrl: '',
    });
  });

  it('opens the pre-filled form from ?action=add&name=…&baseURL=…, expanded, and creates', async () => {
    const w = await mountView({
      query: {
        action: 'add',
        name: 'LM',
        kind: 'lmstudio',
        baseURL: 'http://127.0.0.1:1234/v1',
        model: 'm',
      },
    });
    const form = w.findComponent(Form);
    expect(form.props('modelValue')).toMatchObject({
      name: 'LM',
      kind: 'lmstudio',
      baseUrl: 'http://127.0.0.1:1234/v1',
      model: 'm',
    });
    expect(
      w.findAll('section').slice(-1)[0].find('div[style]').attributes('style') ?? '',
    ).not.toContain('display: none');
    await form.props('onSave')({ name: 'LM' });
    expect(h.api.addProviderConfig).toHaveBeenCalled();
    expect(h.api.updateProviderConfig).not.toHaveBeenCalled();
  });

  it('accepts baseUrl spelling, and defaults kind/name when only ?name is given', async () => {
    const w = await mountView({ query: { name: 'Only', baseUrl: 'http://b' } });
    expect(w.findComponent(Form).props('modelValue')).toMatchObject({
      name: 'Only',
      kind: 'openaicompat',
      baseUrl: 'http://b',
    });
    const bare = await mountView({ query: { action: 'add' } });
    expect(bare.findComponent(Form).props('modelValue')).toMatchObject({
      name: '',
      baseUrl: '',
      model: '',
    });
  });

  it('does not open a form without the query', async () => {
    expect((await mountView()).findComponent(Form).exists()).toBe(false);
  });

  it('searches providers that report no URL and tolerates a null config list', async () => {
    const w = await mountView({
      providers: [prov({ name: 'NoUrl', baseURL: undefined })],
      configs: null as unknown as ProviderConfig[],
    });
    await w.find('input[placeholder="Search providers…"]').setValue('nourl');
    expect(cards(w)).toHaveLength(1);
    await w.find('input[placeholder="Search providers…"]').setValue('zzz');
    expect(cards(w)).toHaveLength(0);
    expect(w.text()).toContain('(0 saved)');
  });
});
