import { describe, it, expect } from 'vitest';
import { createRouter, createMemoryHistory } from 'vue-router';
import routes from './routes';

const named = routes.filter((r) => r.name);

describe('routes', () => {
  it('has unique names and paths', () => {
    expect(new Set(named.map((r) => r.name)).size).toBe(named.length);
    expect(new Set(routes.map((r) => r.path)).size).toBe(routes.length);
  });

  it('every sidebar item has a label and an icon', () => {
    const nav = routes.filter((r) => r.nav);
    expect(nav.length).toBeGreaterThanOrEqual(9);
    for (const r of nav) {
      expect(r.label, String(r.name)).toBeTruthy();
      expect(r.icon, String(r.name)).toMatch(/^pi-/);
    }
  });

  it('keeps the timeline reachable by name but out of the sidebar', () => {
    const live = routes.find((r) => r.name === 'live-activity')!;
    expect(live.nav).toBeFalsy();
    expect(live.path).toBe('/projects/live-activity');
  });

  it.each(named.map((r) => [String(r.name), r] as const))(
    '%s lazy-loads a component',
    async (_n, route) => {
      const load = route.component as () => Promise<{ default: unknown }>;
      expect(typeof load).toBe('function');
      expect((await load()).default).toBeTruthy();
    },
  );

  it('resolves every named route and redirects unknown paths home', async () => {
    const router = createRouter({ history: createMemoryHistory(), routes });
    for (const r of named) expect(router.resolve({ name: r.name! }).path).toBe(r.path);

    await router.push('/definitely/not/a/page');
    expect(router.currentRoute.value.fullPath).toBe('/');
    expect(router.currentRoute.value.name).toBe('mission-control');
  });
});
