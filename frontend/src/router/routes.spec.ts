import { ICONS } from '../ui/icons';
import { describe, it, expect } from 'vitest';
import { createRouter, createMemoryHistory } from 'vue-router';
import routes from './routes';

const named = routes.filter((r) => r.name);
const hub = routes.find((r) => r.path === '/projects')!;
const hubChildren = hub.children!;

describe('routes', () => {
  it('has unique names and paths', () => {
    expect(new Set(named.map((r) => r.name)).size).toBe(named.length);
    expect(new Set(routes.map((r) => r.path)).size).toBe(routes.length);
  });

  it('every sidebar item has a label and an icon', () => {
    const nav = routes.filter((r) => r.nav);
    // One sidebar item per thing that really exists; everything about a project is a tab of Projects.
    expect(nav.map((r) => r.label)).toEqual([
      'Mission Control',
      'Agents',
      'Providers',
      'Projects',
      'Settings',
    ]);
    for (const r of nav) {
      expect(r.label, String(r.name)).toBeTruthy();
      expect(Object.keys(ICONS), String(r.name)).toContain(r.icon);
    }
  });

  it('keeps the timeline reachable by name but out of the sidebar', () => {
    const live = hubChildren.find((r) => r.name === 'live-activity')!;
    expect((live as { nav?: boolean }).nav).toBeFalsy();
    expect(live.path).toBe('live-activity');
  });

  it('puts activity, plans, history, backlog and brain under the Projects hub', async () => {
    expect(hubChildren.map((r) => r.name)).toEqual([
      'projects',
      'live-activity',
      'plans',
      'history',
      'backlog',
      'brain',
    ]);
    const router = createRouter({ history: createMemoryHistory(), routes });
    for (const [name, path] of [
      ['projects', '/projects'],
      ['live-activity', '/projects/live-activity'],
      ['plans', '/projects/plans'],
      ['history', '/projects/history'],
      ['backlog', '/projects/backlog'],
      ['brain', '/projects/brain'],
    ]) {
      expect(router.resolve({ name }).path).toBe(path);
    }
  });

  it('sends the old top-level pages to their tab', async () => {
    const router = createRouter({ history: createMemoryHistory(), routes });
    for (const [from, to] of [
      ['/plans', '/projects/plans'],
      ['/history', '/projects/history'],
      ['/backlog', '/projects/backlog'],
      ['/brain', '/projects/brain'],
    ]) {
      await router.push(from);
      expect(router.currentRoute.value.fullPath).toBe(to);
    }
  });

  it.each([...named, ...hubChildren, hub].map((r) => [String(r.name ?? r.path), r] as const))(
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
    expect(router.resolve({ path: '/projects' }).matched.map((m) => m.path)).toEqual([
      '/projects',
      '/projects',
    ]);

    await router.push('/definitely/not/a/page');
    expect(router.currentRoute.value.fullPath).toBe('/');
    expect(router.currentRoute.value.name).toBe('mission-control');
  });
});
