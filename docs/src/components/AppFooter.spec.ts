import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import { memoryRouter } from '../test/router';
import AppFooter from './AppFooter.vue';

async function mountFooter() {
  const router = memoryRouter();
  await router.isReady();
  return mount(AppFooter, { global: { plugins: [router] } });
}

describe('AppFooter', () => {
  it('has the three link columns', async () => {
    const w = await mountFooter();
    for (const heading of ['Docs', 'Download', 'Community']) expect(w.text()).toContain(heading);
  });

  it('links internal pages through the router and external ones to the right repository', async () => {
    const w = await mountFooter();
    const hrefs = w.findAll('a').map((a) => a.attributes('href') ?? '');
    for (const page of [
      '/getting-started',
      '/architecture',
      '/api-reference',
      '/mcp-integration',
      '/downloads',
    ]) {
      expect(hrefs, page).toContain(page);
    }
    const external = hrefs.filter((h) => h.startsWith('https://'));
    expect(external.length).toBeGreaterThanOrEqual(4);
    for (const h of external) expect(h, h).toMatch(/github\.com\/el-j\/nexus-orchestrator/);
  });

  it('shows the current year', async () => {
    expect((await mountFooter()).text()).toContain(String(new Date().getFullYear()));
  });
});
