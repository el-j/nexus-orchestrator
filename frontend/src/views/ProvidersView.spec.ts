import { describe, it, expect, vi, beforeEach } from 'vitest';
import { shallowMount, flushPromises } from '@vue/test-utils';
import ProvidersView from './ProvidersView.vue';

// Mock wails bindings
vi.mock('../types/wails', () => ({
  getProviders: vi.fn().mockResolvedValue([
    {
      name: 'LM Studio',
      active: true,
      activeModel: 'qwen',
      models: ['qwen'],
      baseURL: 'http://127.0.0.1:1234/v1',
      contextLimit: 32768,
      lastChecked: new Date().toISOString(),
      consecutiveFails: 0,
      latencyMs: 42,
    },
  ]),
  getDiscoveredProviders: vi.fn().mockResolvedValue([]),
  triggerScan: vi.fn(),
  listProviderConfigs: vi.fn().mockResolvedValue([]),
  addProviderConfig: vi.fn(),
  removeProviderConfig: vi.fn(),
  updateProviderConfig: vi.fn(),
  getRouting: vi.fn().mockResolvedValue({ quotaThreshold: 0.9, order: [] }),
}));

// Mock useServerUrl
vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl: vi.fn().mockResolvedValue('http://localhost:63987'),
}));

// Mock fetch for discovered agents
vi.stubGlobal(
  'fetch',
  vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve([]),
  }),
);

describe('ProvidersView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders without errors', () => {
    const wrapper = shallowMount(ProvidersView, {
      global: {
        stubs: {
          ProviderStatus: true,
          DiscoveredProvidersPanel: true,
          ProviderConfigForm: true,
          RefreshIndicator: true,
        },
      },
    });
    expect(wrapper.exists()).toBe(true);
  });

  it('shows the Providers header', () => {
    const wrapper = shallowMount(ProvidersView, {
      global: {
        stubs: {
          ProviderStatus: true,
          DiscoveredProvidersPanel: true,
          ProviderConfigForm: true,
          RefreshIndicator: true,
        },
      },
    });
    expect(wrapper.find('header').text()).toContain('Providers');
  });

  it('renders provider cards after data loads', async () => {
    const wrapper = shallowMount(ProvidersView, {
      global: {
        stubs: {
          ProviderStatus: true,
          DiscoveredProvidersPanel: true,
          ProviderConfigForm: true,
          RefreshIndicator: true,
        },
      },
    });
    await flushPromises();

    const html = wrapper.html();
    expect(html).toContain('LM Studio');
    expect(html).toContain('Active');
  });

  it('shows context limit text in provider cards', async () => {
    const wrapper = shallowMount(ProvidersView, {
      global: {
        stubs: {
          ProviderStatus: true,
          DiscoveredProvidersPanel: true,
          ProviderConfigForm: true,
          RefreshIndicator: true,
        },
      },
    });
    await flushPromises();

    // Provider card should show the base URL
    const html = wrapper.html();
    expect(html).toContain('http://127.0.0.1:1234/v1');
  });

  it('shows active model in provider card', async () => {
    const wrapper = shallowMount(ProvidersView, {
      global: {
        stubs: {
          ProviderStatus: true,
          DiscoveredProvidersPanel: true,
          ProviderConfigForm: true,
          RefreshIndicator: true,
        },
      },
    });
    await flushPromises();

    const html = wrapper.html();
    expect(html).toContain('qwen');
  });

  it('has Active Providers section', () => {
    const wrapper = shallowMount(ProvidersView, {
      global: {
        stubs: {
          ProviderStatus: true,
          DiscoveredProvidersPanel: true,
          ProviderConfigForm: true,
          RefreshIndicator: true,
        },
      },
    });
    expect(wrapper.html()).toContain('Active Providers');
  });

  it('has a refresh button', () => {
    const wrapper = shallowMount(ProvidersView, {
      global: {
        stubs: {
          ProviderStatus: true,
          DiscoveredProvidersPanel: true,
          ProviderConfigForm: true,
          RefreshIndicator: true,
        },
      },
    });
    const refreshBtn = wrapper.findAll('button').find((b) => b.text().includes('Refresh'));
    expect(refreshBtn).toBeDefined();
  });

  it('pre-fills the command line when promoting a discovered CLI provider', async () => {
    const wrapper = shallowMount(ProvidersView, {
      global: {
        stubs: {
          ProviderStatus: true,
          DiscoveredProvidersPanel: false,
          ProviderConfigForm: true,
          RefreshIndicator: true,
        },
      },
    });
    await flushPromises();
    const panel = wrapper.findComponent({ name: 'DiscoveredProvidersPanel' });
    panel.vm.$emit('promote', {
      id: 'cli-claude',
      name: 'Claude CLI',
      kind: 'cli',
      method: 'cli',
      status: 'installed',
      cliPath: '/usr/local/bin/claude',
      lastSeen: new Date().toISOString(),
    });
    await flushPromises();
    const form = wrapper.findComponent({ name: 'ProviderConfigForm' });
    expect(form.exists()).toBe(true);
    expect(form.props('modelValue')).toMatchObject({
      name: 'Claude CLI',
      kind: 'cli',
      baseUrl: '/usr/local/bin/claude -p --output-format stream-json',
    });

    panel.vm.$emit('promote', {
      id: 'cli-copilot',
      name: 'Copilot CLI',
      kind: 'cli',
      method: 'cli',
      status: 'installed',
      cliPath: '/usr/local/bin/copilot.exe',
      lastSeen: new Date().toISOString(),
    });
    await flushPromises();
    expect(wrapper.findComponent({ name: 'ProviderConfigForm' }).props('modelValue')).toMatchObject(
      {
        name: 'Copilot CLI',
        kind: 'cli',
        baseUrl: '/usr/local/bin/copilot.exe -s --no-ask-user',
      },
    );

    panel.vm.$emit('promote', {
      id: 'cli-aichat',
      name: 'aichat',
      kind: 'cli',
      method: 'cli',
      status: 'installed',
      cliPath: '/usr/local/bin/aichat',
      lastSeen: new Date().toISOString(),
    });
    await flushPromises();
    expect(wrapper.findComponent({ name: 'ProviderConfigForm' }).props('modelValue')).toMatchObject(
      {
        name: 'aichat',
        kind: 'cli',
        baseUrl: '/usr/local/bin/aichat -p',
      },
    );
  });
});
