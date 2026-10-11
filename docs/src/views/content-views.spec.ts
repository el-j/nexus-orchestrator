import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, type VueWrapper } from '@vue/test-utils';
import { memoryRouter } from '../test/router';
import { routes } from '../router';
import HomeView from './HomeView.vue';
import ArchitectureView from './ArchitectureView.vue';
import GettingStartedView from './GettingStartedView.vue';
import McpIntegrationView from './McpIntegrationView.vue';

const scrollToId = vi.hoisted(() => vi.fn());
vi.mock('../utils/scroll', () => ({ scrollToId }));

// PrimeVue tabs need the PrimeVue plugin; the content is what matters here.
const slot = (tag = 'div') => ({ template: `<${tag}><slot /></${tag}>` });
const stubs = {
  Tabs: slot(),
  TabList: slot(),
  Tab: slot('button'),
  TabPanels: slot(),
  TabPanel: slot(),
};

const pages = [
  { name: 'HomeView', view: HomeView, h1: /./ },
  { name: 'ArchitectureView', view: ArchitectureView, h1: /Architecture/ },
  { name: 'GettingStartedView', view: GettingStartedView, h1: /Getting Started/ },
  { name: 'McpIntegrationView', view: McpIntegrationView, h1: /MCP Integration/ },
];

async function mountPage(view: object): Promise<VueWrapper> {
  const router = memoryRouter();
  await router.isReady();
  return mount(view, { global: { plugins: [router], stubs } });
}

beforeEach(() => scrollToId.mockReset());

describe.each(pages)('$name', ({ view, h1 }) => {
  it('renders a heading and substantial content', async () => {
    const w = await mountPage(view);
    expect(w.get('h1').text()).toMatch(h1);
    expect(w.text().length).toBeGreaterThan(800);
  });

  it('only links to pages that exist', async () => {
    const w = await mountPage(view);
    const known = new Set(routes.map((r) => r.path));
    for (const a of w.findAll('a')) {
      const href = a.attributes('href') ?? '';
      if (href === '' || href.startsWith('#')) continue;
      if (/^https?:/.test(href)) continue;
      expect(known.has(href), `${href} is not a route`).toBe(true);
    }
  });

  it('opens external links safely, in a new tab', async () => {
    const w = await mountPage(view);
    for (const a of w.findAll('a[target="_blank"]')) {
      expect(a.attributes('rel'), a.attributes('href')).toContain('noopener');
    }
  });

  it('never points at the old repository name', async () => {
    const w = await mountPage(view);
    expect(w.html()).not.toContain('el-j/nexusOrchestrator');
  });
});

describe('table-of-contents links', () => {
  it.each(pages.filter((p) => p.name !== 'HomeView'))(
    '$name scrolls without touching the router hash',
    async ({ view }) => {
      const w = await mountPage(view);
      const toc = w.findAll('aside a');
      expect(toc.length).toBeGreaterThan(3);
      const id = toc[0].attributes('href')!.slice(1);
      // every TOC entry targets a section that exists on the page
      for (const link of toc) {
        const target = link.attributes('href')!.slice(1);
        expect(w.find(`[id="${target}"]`).exists(), target).toBe(true);
      }
      await toc[0].trigger('click');
      expect(scrollToId).toHaveBeenCalledWith(id);
    },
  );
});

describe('content facts that must match the code', () => {
  it('the MCP page and the home page quote the real default ports', async () => {
    const mcp = await mountPage(McpIntegrationView);
    const home = await mountPage(HomeView);
    expect(mcp.text()).toContain('63988');
    expect(home.text()).toContain('63987');
  });

  it('the getting-started example shows the real submit response', async () => {
    const w = await mountPage(GettingStartedView);
    expect(w.html()).toContain('task_id');
  });
});
