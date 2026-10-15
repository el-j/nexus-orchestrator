import { afterEach, describe, it, expect, vi } from 'vitest';
import router from './index';
import routes from './routes';

describe('router', () => {
  it('is built from the shared route table', () => {
    const names = router
      .getRoutes()
      .map((r) => r.name)
      .filter(Boolean);
    for (const r of routes.filter((x) => x.name)) expect(names).toContain(r.name);
  });

  it('navigates to a view by name', async () => {
    await router.push({ name: 'settings' });
    expect(router.currentRoute.value.path).toBe('/settings');
  });
});

describe('router base', () => {
  afterEach(() => {
    document.head.querySelectorAll('base').forEach((b) => b.remove());
    vi.resetModules();
  });

  it('lives under the <base href> the daemon puts into index.html', async () => {
    const base = document.createElement('base');
    base.setAttribute('href', '/app/');
    document.head.appendChild(base);
    vi.resetModules();
    const { default: scoped } = await import('./index');
    expect(scoped.resolve({ name: 'settings' }).href).toBe('/app/settings');
  });

  it('lives at the root without one (desktop app, dev server)', async () => {
    vi.resetModules();
    const { default: rooted } = await import('./index');
    expect(rooted.resolve({ name: 'settings' }).href).toBe('/settings');
  });
});
