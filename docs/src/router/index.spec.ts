import { describe, it, expect } from 'vitest';
import { routes } from './index';
import { guides } from '../data/guides';

describe('routes', () => {
  it('covers every documented page, each with a title', () => {
    expect(routes.map((r) => r.path)).toEqual([
      '/',
      '/downloads',
      '/getting-started',
      '/architecture',
      '/api-reference',
      '/mcp-integration',
      '/guides',
      ...guides.map((g) => `/guides/${g.slug}`),
    ]);
    for (const r of routes) expect(String(r.meta?.title), r.path).toMatch(/nexus-orchestrator/);
  });

  it('has unique paths and titles', () => {
    expect(new Set(routes.map((r) => r.path)).size).toBe(routes.length);
    expect(new Set(routes.map((r) => r.meta?.title)).size).toBe(routes.length);
  });

  it.each(routes.filter((r) => typeof r.component === 'function'))(
    'lazy-loads $path',
    async (route) => {
      const load = route.component as () => Promise<{ default: unknown }>;
      expect((await load()).default).toBeTruthy();
    },
  );
});
