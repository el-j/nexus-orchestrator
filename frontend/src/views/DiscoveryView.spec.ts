import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { ref } from 'vue';
import DiscoveryView from './DiscoveryView.vue';
import type { DiscoveredProvider } from '../types/discovery';

const h = vi.hoisted(() => ({
  push: vi.fn(),
  scanNow: vi.fn(),
  state: { discovered: [] as unknown[], loading: false, scanning: false },
}));
vi.mock('vue-router', () => ({ useRouter: () => ({ push: h.push }) }));
vi.mock('../composables/useDiscovery', () => ({
  useDiscovery: () => ({
    discovered: ref(h.state.discovered),
    loading: ref(h.state.loading),
    scanning: ref(h.state.scanning),
    scanNow: h.scanNow,
  }),
}));

const Panel = {
  name: 'DiscoveredProvidersPanel',
  props: ['providers', 'loading', 'scanning'],
  emits: ['scan', 'promote'],
  template: '<div data-testid="panel" />',
};

const provider = (o: Partial<DiscoveredProvider> = {}): DiscoveredProvider => ({
  id: 'p1',
  name: 'LM Studio',
  kind: 'lmstudio',
  method: 'port',
  status: 'reachable',
  baseUrl: 'http://127.0.0.1:1234/v1',
  lastSeen: '',
  ...o,
});

const mountView = () =>
  mount(DiscoveryView, { global: { stubs: { DiscoveredProvidersPanel: Panel } } });

beforeEach(() => {
  h.push.mockReset();
  h.scanNow.mockReset();
  h.state = { discovered: [], loading: false, scanning: false };
});

describe('DiscoveryView', () => {
  it('counts the detected tools', () => {
    h.state.discovered = [provider(), provider({ id: 'p2' })];
    expect(mountView().text()).toContain('2');
  });

  it('passes state down to the panel', () => {
    h.state = { discovered: [provider()], loading: true, scanning: true };
    const panel = mountView().findComponent(Panel);
    expect(panel.props()).toMatchObject({ loading: true, scanning: true });
    expect(panel.props('providers')).toHaveLength(1);
  });

  it('scans from the header button and from the panel', async () => {
    const w = mountView();
    await w.find('header button').trigger('click');
    w.findComponent(Panel).vm.$emit('scan');
    expect(h.scanNow).toHaveBeenCalledTimes(2);
  });

  it('shows progress and disables the header button while scanning', () => {
    h.state.scanning = true;
    const btn = mountView().find('header button');
    expect(btn.text()).toBe('Scanning…');
    expect(btn.attributes('disabled')).toBeDefined();
  });

  it('promoting sends the provider to the Providers page pre-filled', () => {
    const w = mountView();
    w.findComponent(Panel).vm.$emit('promote', provider());
    expect(h.push).toHaveBeenCalledWith({
      path: '/providers',
      query: {
        action: 'add',
        name: 'LM Studio',
        baseURL: 'http://127.0.0.1:1234/v1',
        kind: 'lmstudio',
      },
    });
  });
});
