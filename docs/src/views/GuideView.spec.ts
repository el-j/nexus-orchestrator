import { describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createMemoryHistory, createRouter } from 'vue-router';
import GuideView from './GuideView.vue';
import GuidesView from './GuidesView.vue';
import { guides } from '../data/guides';

const scrollToId = vi.hoisted(() => vi.fn());
vi.mock('../utils/scroll', () => ({ scrollToId }));

async function mountAt(path: string, view: object) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/guides', component: GuidesView },
      ...guides.map((g) => ({
        path: `/guides/${g.slug}`,
        component: GuideView,
        meta: { slug: g.slug },
      })),
    ],
  });
  await router.push(path);
  return mount(view, { global: { plugins: [router] } });
}

describe('GuideView', () => {
  it.each(guides.map((g) => [g.slug, g] as const))(
    'renders %s with a table of contents',
    async (slug, g) => {
      const w = await mountAt(`/guides/${slug}`, GuideView);
      expect(w.get('h1').text()).toBe(g.title);
      expect(w.get('[data-testid="md-doc"]').text().length).toBeGreaterThan(500);
      expect(w.findAll('nav a').length).toBeGreaterThan(0);
    },
  );

  it('indents the sub-sections of a guide in the table of contents', async () => {
    let indented = 0;
    for (const g of guides) {
      const w = await mountAt(`/guides/${g.slug}`, GuideView);
      indented += w.findAll('nav a.pl-3').length;
    }
    expect(indented).toBeGreaterThan(0);
  });

  it('jumps to a section from the table of contents, and links to the source file', async () => {
    scrollToId.mockReset();
    const w = await mountAt('/guides/plan-files', GuideView);
    await w.get('nav a').trigger('click');
    expect(scrollToId).toHaveBeenCalledOnce();
    expect(w.get('a[href$="/docs/plan-files.md"]').text()).toContain('docs/plan-files.md');
  });

  it('turns links between guides into site pages and other relative links into repository links', async () => {
    const w = await mountAt('/guides/how-work-reaches-nexus', GuideView);
    const hrefs = w.findAll('[data-testid="md-doc"] a').map((a) => a.attributes('href') ?? '');
    expect(hrefs.some((h) => h.includes('/guides/plan-files/'))).toBe(true);
    expect(hrefs.filter((h) => h.endsWith('.md') && !h.startsWith('http'))).toEqual([]);
  });
});

describe('GuidesView', () => {
  it('lists every guide with a link to its page', async () => {
    const w = await mountAt('/guides', GuidesView);
    expect(w.get('h1').text()).toBe('Guides');
    const links = w.findAll('a').map((a) => a.attributes('href'));
    for (const g of guides) expect(links).toContain(`/guides/${g.slug}`);
  });
});
