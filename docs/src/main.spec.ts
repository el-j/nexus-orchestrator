import { describe, it, expect, vi, beforeEach } from 'vitest';

const h = vi.hoisted(() => ({
  setup: undefined as undefined | ((ctx: unknown) => void),
  options: undefined as unknown,
}));
vi.mock('vite-ssg', () => ({
  ViteSSG: (_app: unknown, options: unknown, setup: (ctx: unknown) => void) => {
    h.options = options;
    h.setup = setup;
    return () => 'create-app';
  },
}));
vi.mock('primevue/config', () => ({ default: { name: 'PrimeVue' } }));
vi.mock('@primevue/themes/aura', () => ({ default: { name: 'Aura' } }));
vi.mock('primevue/toastservice', () => ({ default: { name: 'ToastService' } }));
vi.mock('primeicons/primeicons.css', () => ({}));
vi.mock('./assets/main.css', () => ({}));

beforeEach(() => {
  vi.resetModules();
});

const context = (isClient: boolean) => {
  const afterEach = vi.fn();
  return { app: { use: vi.fn() }, router: { afterEach }, isClient, afterEach };
};

describe('main', () => {
  it('exports the ViteSSG factory built from the site routes', async () => {
    const { createApp } = await import('./main');
    expect(createApp()).toBe('create-app');
    const { routes } = await import('./router');
    expect((h.options as { routes: unknown }).routes).toBe(routes);
  });

  it('installs PrimeVue (Aura, dark selector) and the toast service', async () => {
    await import('./main');
    const ctx = context(false);
    h.setup!(ctx);
    const [[primevue, opts], [toast]] = ctx.app.use.mock.calls;
    expect(primevue).toEqual({ name: 'PrimeVue' });
    expect(opts.theme.preset).toEqual({ name: 'Aura' });
    expect(opts.theme.options).toMatchObject({ darkModeSelector: '.dark', cssLayer: false });
    expect(toast).toEqual({ name: 'ToastService' });
  });

  it('keeps document.title in step with navigation, in the browser only', async () => {
    await import('./main');
    const server = context(false);
    h.setup!(server);
    expect(server.afterEach).not.toHaveBeenCalled(); // no document during pre-rendering

    const client = context(true);
    h.setup!(client);
    const hook = client.afterEach.mock.calls[0][0] as (to: { meta?: { title?: string } }) => void;
    hook({ meta: { title: 'Downloads' } });
    expect(document.title).toBe('Downloads');
    hook({ meta: {} });
    expect(document.title).toBe('Downloads'); // untitled routes keep the previous title
    hook({});
    expect(document.title).toBe('Downloads');
  });
});
