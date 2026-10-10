import { describe, it, expect } from 'vitest';
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
