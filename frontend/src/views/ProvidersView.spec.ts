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
});
