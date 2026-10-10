import { describe, it, expect, vi, beforeEach } from 'vitest';

const h = vi.hoisted(() => ({
  use: vi.fn(),
  directive: vi.fn(),
  mount: vi.fn(),
  config: {} as { errorHandler?: (e: unknown, i: unknown, info: string) => void },
  createApp: vi.fn(),
}));

vi.mock('vue', async (orig) => {
  const actual = await orig<typeof import('vue')>();
  return { ...actual, createApp: h.createApp };
});
vi.mock('primevue/config', () => ({ default: { name: 'PrimeVue' } }));
vi.mock('@primevue/themes/aura', () => ({ default: { name: 'Aura' } }));
vi.mock('primevue/toastservice', () => ({ default: { name: 'Toast' } }));
vi.mock('primevue/confirmationservice', () => ({ default: { name: 'Confirm' } }));
vi.mock('primevue/tooltip', () => ({ default: { name: 'Tooltip' } }));
vi.mock('primeicons/primeicons.css', () => ({}));
vi.mock('./assets/main.css', () => ({}));
vi.mock('./App.vue', () => ({ default: { name: 'App' } }));
vi.mock('./router/index', () => ({ default: { name: 'router' } }));

beforeEach(() => {
  vi.resetModules();
  Object.values(h).forEach((f) => typeof f === 'function' && 'mockReset' in f && f.mockReset());
  h.config = {};
  h.createApp.mockImplementation(() => ({
    config: h.config,
    use: h.use,
    directive: h.directive,
    mount: h.mount,
  }));
});

describe('main', () => {
  it('installs PrimeVue (Aura, dark selector), services, the router and the tooltip, then mounts #app', async () => {
    await import('./main');
    expect(h.createApp).toHaveBeenCalledWith({ name: 'App' });
    const [[primevue, opts]] = h.use.mock.calls;
    expect(primevue).toEqual({ name: 'PrimeVue' });
    expect(opts.theme.preset).toEqual({ name: 'Aura' });
    expect(opts.theme.options).toMatchObject({ darkModeSelector: '.dark', cssLayer: false });
    expect(h.use.mock.calls.map((c) => c[0])).toEqual([
      { name: 'PrimeVue' },
      { name: 'Toast' },
      { name: 'Confirm' },
      { name: 'router' },
    ]);
    expect(h.directive).toHaveBeenCalledWith('tooltip', { name: 'Tooltip' });
    expect(h.mount).toHaveBeenCalledWith('#app');
  });

  it('logs Vue errors instead of crashing', async () => {
    const err = vi.spyOn(console, 'error').mockImplementation(() => {});
    await import('./main');
    h.config.errorHandler!(new Error('x'), null, 'render');
    expect(err).toHaveBeenCalledWith('[Vue] Unhandled error:', expect.any(Error), 'render');
    err.mockRestore();
  });

  it('swallows unhandled promise rejections after logging them', async () => {
    const err = vi.spyOn(console, 'error').mockImplementation(() => {});
    await import('./main');
    const event = new Event('unhandledrejection', { cancelable: true }) as Event & {
      reason?: unknown;
    };
    event.reason = 'nope';
    window.dispatchEvent(event);
    expect(err).toHaveBeenCalledWith('[Vue] Unhandled promise rejection:', 'nope');
    expect(event.defaultPrevented).toBe(true);
    err.mockRestore();
  });
});
