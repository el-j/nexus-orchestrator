import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { defineComponent, h } from 'vue';
import App from './App.vue';

const sse = vi.hoisted(() => ({ connect: vi.fn() }));
vi.mock('./composables/useGlobalSSE', () => ({ useGlobalSSE: () => ({ connect: sse.connect }) }));

const Boom = defineComponent({
  setup() {
    throw new Error('view exploded');
  },
  render: () => h('div'),
});
const Fine = { template: '<div data-testid="view">ok</div>' };

const stubs = {
  AppSidebar: { template: '<nav data-testid="sidebar" />' },
  LogPanel: { template: '<footer data-testid="logs" />' },
  UiToaster: { template: '<div data-testid="toast" />' },
  ErrorFallback: {
    props: ['error'],
    emits: ['retry'],
    template: '<div data-testid="fallback" @click="$emit(\'retry\')">{{ error.message }}</div>',
  },
};

function mountApp(view: unknown) {
  return mount(App, { global: { stubs: { ...stubs, RouterView: view as object } } });
}

beforeEach(() => {
  sse.connect.mockReset();
  vi.spyOn(console, 'error').mockImplementation(() => {});
});

describe('App', () => {
  it('opens the realtime stream once at startup', () => {
    mountApp(Fine);
    expect(sse.connect).toHaveBeenCalledTimes(1);
  });

  it('renders the shell: sidebar, routed view, log panel, toasts', () => {
    const w = mountApp(Fine);
    for (const id of ['sidebar', 'view', 'logs', 'toast']) {
      expect(w.find(`[data-testid="${id}"]`).exists(), id).toBe(true);
    }
    expect(w.find('[data-testid="fallback"]').exists()).toBe(false);
  });

  it('replaces the shell with the error fallback when a view throws, and recovers on retry', async () => {
    const w = mount(App, { global: { stubs: { ...stubs, RouterView: Boom } } });
    await w.vm.$nextTick();
    const fallback = w.find('[data-testid="fallback"]');
    expect(fallback.exists()).toBe(true);
    expect(fallback.text()).toContain('view exploded');
    expect(w.find('[data-testid="sidebar"]').exists()).toBe(false);
  });

  it('wraps a non-Error throwable', async () => {
    const Str = defineComponent({
      setup() {
        throw 'plain string';
      },
      render: () => h('div'),
    });
    const w = mount(App, { global: { stubs: { ...stubs, RouterView: Str } } });
    await w.vm.$nextTick();
    expect(w.find('[data-testid="fallback"]').text()).toContain('plain string');
  });

  it('clears the error when the fallback asks to retry', async () => {
    let attempts = 0;
    const Flaky = defineComponent({
      setup() {
        if (attempts++ === 0) throw new Error('first render fails');
      },
      render: () => h('div', { 'data-testid': 'view' }, 'ok'),
    });
    const w = mount(App, { global: { stubs: { ...stubs, RouterView: Flaky } } });
    await w.vm.$nextTick();
    await w.find('[data-testid="fallback"]').trigger('click');
    expect(w.find('[data-testid="sidebar"]').exists()).toBe(true);
  });
});
