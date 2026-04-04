import { describe, it, expect, vi, beforeEach } from 'vitest';
import { shallowMount } from '@vue/test-utils';
import { flushPromises } from '@vue/test-utils';
import SettingsView from './SettingsView.vue';

// ── Wails mock ────────────────────────────────────────────────────────────────

const { mockGetRuntimeConfig, mockUpdateRuntimeConfig } = vi.hoisted(() => ({
  mockGetRuntimeConfig: vi.fn(),
  mockUpdateRuntimeConfig: vi.fn(),
}));

vi.mock('../types/wails', () => ({
  getRuntimeConfig: mockGetRuntimeConfig,
  updateRuntimeConfig: mockUpdateRuntimeConfig,
  getServerAddr: vi.fn().mockResolvedValue('http://127.0.0.1:63987'),
}));

vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl: vi.fn().mockResolvedValue('http://127.0.0.1:63987'),
}));

// ── Helpers ───────────────────────────────────────────────────────────────────

function makeConfig(overrides: Record<string, unknown> = {}) {
  return {
    queueCap: 50,
    apiTokenEnabled: false,
    mcpTokenEnabled: false,
    apiToken: '',
    mcpToken: '',
    ...overrides,
  };
}

const GLOBAL_STUBS = {
  global: {
    stubs: {
      AppConfirmDialog: true,
    },
  },
};

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('SettingsView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetRuntimeConfig.mockResolvedValue(makeConfig());
    mockUpdateRuntimeConfig.mockResolvedValue(makeConfig());
  });

  it('mounts without errors and renders Queue Settings and Security sections', async () => {
    const wrapper = shallowMount(SettingsView, GLOBAL_STUBS);
    await flushPromises();

    expect(wrapper.exists()).toBe(true);
    const html = wrapper.html();
    expect(html).toContain('Settings');
    expect(html).toContain('Queue');
    expect(html).toContain('Security');
  });

  it('calls getRuntimeConfig on mount and displays queue cap', async () => {
    mockGetRuntimeConfig.mockResolvedValue(makeConfig({ queueCap: 100 }));

    const wrapper = shallowMount(SettingsView, GLOBAL_STUBS);
    await flushPromises();

    expect(mockGetRuntimeConfig).toHaveBeenCalledTimes(1);
    expect(wrapper.html()).toContain('100');
  });

  it('clicking Generate Token button calls updateRuntimeConfig with rotateApiToken', async () => {
    mockGetRuntimeConfig.mockResolvedValue(makeConfig({ apiTokenEnabled: false }));
    mockUpdateRuntimeConfig.mockResolvedValue(
      makeConfig({ apiTokenEnabled: true, apiToken: 'new-token-abc' }),
    );

    const wrapper = shallowMount(SettingsView, GLOBAL_STUBS);
    await flushPromises();

    // Find the "Generate Token" / "Rotate" button for the API token row
    const buttons = wrapper.findAll('button');
    const tokenBtn = buttons.find(
      (b) =>
        b.text().toLowerCase().includes('generate token') ||
        b.text().toLowerCase().includes('rotate'),
    );
    expect(tokenBtn).toBeDefined();

    await tokenBtn!.trigger('click');
    await flushPromises();

    expect(mockUpdateRuntimeConfig).toHaveBeenCalledWith(
      expect.objectContaining({ rotateApiToken: true }),
    );
  });

  it('displays server address from resolveServerUrl on mount', async () => {
    const { resolveServerUrl } = await import('../composables/useServerUrl');
    vi.mocked(resolveServerUrl).mockResolvedValue('http://127.0.0.1:63987');

    const wrapper = shallowMount(SettingsView, GLOBAL_STUBS);
    await flushPromises();

    const html = wrapper.html();
    expect(html).toContain('Server Addresses');
    expect(html).toContain('63987');
  });

  it('shows configError banner when getRuntimeConfig throws', async () => {
    mockGetRuntimeConfig.mockRejectedValue(new Error('connection refused'));

    const wrapper = shallowMount(SettingsView, GLOBAL_STUBS);
    await flushPromises();

    expect(wrapper.html()).toContain('connection refused');
  });
});
